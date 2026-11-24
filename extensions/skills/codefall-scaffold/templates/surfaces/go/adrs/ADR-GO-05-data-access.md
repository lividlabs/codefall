# ADR-GO-05: Data Access

## Status

Accepted — <date>

## Context

A Go surface that keeps its data in a relational database needs a way to run SQL, a driver, and a
way to change the schema. ADR-BASE-01 puts all three in `infrastructure/`, behind interfaces that
`application/` declares, and decides none of them. This ADR applies to any surface that reads or
writes a relational database, whether it is a service, a worker, or a CLI. A surface that uses none
has no use for it.

Two rules already in force narrow the choice. `domain/` and `application/` import no database
library (ADR-GO-02), so a row is converted to a domain entity in `infrastructure/` whichever library
reads it. And a component's facade is its whole public API (ADR-BASE-02), so a table that another
component reads is a way around the facade: each component owns its tables, with no data model
shared between components. ADR-BASE-03 spells that rule out for a surface that could be split into
services, and this ADR keeps it on every surface. An *object-relational mapper* (ORM), a library
that maps tables to objects and follows the relations between them, earns its keep from one model
of the whole database, which is what the second rule forbids.

The libraries considered were `sqlc`, GORM, `ent`, `bun`, and `pgx` used alone. GORM and `bun`
build each query while the program runs, so a wrong column name is found when the code runs. `ent`
generates a typed client, and its schema is one graph for the whole app. `pgx` alone leaves every
query's Go to be written by hand. `sqlc` reads SQL, checks each query against the schema when it
generates, and writes the Go.

For migrations the tools considered were goose and Atlas. Atlas writes a migration from a
description of the schema, and its free edition does not manage views, functions, triggers, or
extensions. goose runs SQL files in order, and it is a Go library that compiles into the app.

**The database is the project's to choose, and PostgreSQL is the default.** This ADR is written for
PostgreSQL. A project on MySQL or SQLite keeps `sqlc` and goose and amends this ADR: `sqlc`
generates for `database/sql`, and the driver is that database's own. A database `sqlc` does not
read has no use for this ADR, and the project records its own.

## Decision

### Libraries

- **`sqlc`** generates the Go for every query, for `pgx/v5`. It is not an ORM: the SQL is written
  by hand.
- **`pgx` v5** is the driver, and its **`pgxpool`** is the connection pool.
- **goose** applies the migrations, as a library compiled into the app.
- All three are imported in `infrastructure/`, in a component's facade, and in the composition
  root, and nowhere else.

### Each component owns its SQL

```
internal/<component>/internal/infrastructure/
  migrations/      goose files, embedded in the binary
  queries/         the SQL sqlc reads
  db/              the Go sqlc writes, committed
sqlc.yaml          at the module root, one entry per component
```

- A component's tables live in a PostgreSQL schema of its own, named for the component, and its
  migrations are tracked in a version table of its own, `goose_<component>`. The version table is
  in the database's default schema, because goose makes it before it runs the migration that makes
  the component's schema.
- **One `sqlc.yaml` entry per component, whose `schema` is that component's migrations alone:**

  ```yaml
  version: "2"
  sql:
    - engine: postgresql
      schema: internal/orders/internal/infrastructure/migrations
      queries: internal/orders/internal/infrastructure/queries
      gen:
        go:
          package: db
          out: internal/orders/internal/infrastructure/db
          sql_package: pgx/v5
          emit_pointers_for_null_types: true
          overrides:
            - db_type: "uuid"
              go_type:
                import: "github.com/google/uuid"
                type: "UUID"
            - db_type: "uuid"
              nullable: true
              go_type:
                import: "github.com/google/uuid"
                type: "UUID"
                pointer: true
            - db_type: "timestamptz"
              go_type: "time.Time"
            - db_type: "timestamptz"
              nullable: true
              go_type:
                type: "time.Time"
                pointer: true
  ```

  A query that names another component's table then fails `sqlc generate` with
  `schema "orders" does not exist`.
- **The settings make a generated row hold standard types**: a nullable column is a pointer, a
  `timestamptz` is a `time.Time`, and a `uuid` is a `uuid.UUID` from `github.com/google/uuid`.
- **`sqlc` is installed with `mise`**, pinned to a version in `mise.toml`, and is not a dependency
  in `go.mod`.
  Run `sqlc generate` after a query or a migration changes, and commit what it writes. CI runs
  `sqlc diff`, which fails when the committed code is not what the SQL generates.

### The repository turns a row into an entity

```go
func orderID(id string) (uuid.UUID, error) {
    key, err := uuid.Parse(id)
    if err != nil {
        return uuid.UUID{}, domain.ErrOrderNotFound
    }

    return key, nil
}

func (r *PostgresOrderRepository) Find(ctx context.Context, id string) (domain.Order, error) {
    key, err := orderID(id)
    if err != nil {
        return domain.Order{}, err
    }

    row, err := r.in(ctx).GetOrder(ctx, key)
    if errors.Is(err, pgx.ErrNoRows) {
        return domain.Order{}, domain.ErrOrderNotFound
    }
    if err != nil {
        return domain.Order{}, fmt.Errorf("select order: %w", err)
    }

    return toOrder(row), nil
}

func toOrder(row db.OrdersOrder) domain.Order {
    return domain.Order{
        ID:        row.ID.String(),
        Total:     int(row.TotalCents),
        Note:      mo.PointerToOption(row.Note),
        CreatedAt: row.CreatedAt,
        ShippedAt: mo.PointerToOption(row.ShippedAt),
    }
}
```

- **A nullable column is a `mo.Option` in the entity** (ADR-GO-03): `mo.PointerToOption` on the
  way out of the database, and the option's `ToPointer` on the way in.
- **The repository parses an id before it asks the database.** An id that is not a UUID names
  nothing, so it is not found. Passed through unparsed, it comes back from PostgreSQL as an error
  that reaches the client as a 500.
- **The repository returns the domain's errors, never `pgx`'s.** `pgx.ErrNoRows` becomes the
  component's not-found error here, which is what the caller in `presentation/` matches with
  `errors.Is`.

### Keys

- **A table's key is the project's decision**: its type, and whether the database or the
  application generates it. This ADR does not make that decision, and scaffold does not ask.
- **Where nobody has decided, a table gets a `uuid` key that the database generates**, and the
  insert returns the row, so the application makes no id:

  ```sql
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  ```

  ```sql
  -- name: InsertOrder :one
  INSERT INTO orders.orders (total_cents, note)
  VALUES ($1, $2)
  RETURNING id, total_cents, note, created_at, shipped_at;
  ```

  ```go
  func (r *PostgresOrderRepository) Save(ctx context.Context, order domain.Order) (domain.Order, error) {
      row, err := r.in(ctx).InsertOrder(ctx, db.InsertOrderParams{
          TotalCents: int64(order.Total),
          Note:       order.Note.ToPointer(),
      })
      if err != nil {
          return domain.Order{}, fmt.Errorf("insert order: %w", err)
      }

      return toOrder(row), nil
  }
  ```

- `uuidv7()` is in PostgreSQL 18 and later, and its ids sort by the time they were made. Before 18,
  the default is `gen_random_uuid()`.
- **An id made in the application comes through an interface declared in `application/`**, because
  the strict lists keep an id library out of `domain/` and `application/` (ADR-GO-02).
- A project that decides on another key records it in an ADR of its own. Whatever the key's type,
  the repository turns the id it is given into that type before it queries.

### Transactions

`application/` declares what a use case needs to make several writes one:

```go
type Transactor interface {
    InTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

```go
func (uc *ShipOrder) Execute(ctx context.Context, id string) error {
    err := uc.tx.InTransaction(ctx, func(ctx context.Context) error {
        if err := uc.orders.MarkShipped(ctx, id); err != nil {
            return err
        }

        return uc.orders.RecordEvent(ctx, id, "shipped")
    })
    if err != nil {
        return fmt.Errorf("ship order %q: %w", id, err)
    }

    return nil
}
```

`infrastructure/` implements it. The transaction travels in the context, under a key only that
package can read, and every repository method takes its queries from the context it is given:

```go
type txKey struct{}

func (t *PgxTransactor) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    return pgx.BeginFunc(ctx, t.pool, func(tx pgx.Tx) error {
        return fn(context.WithValue(ctx, txKey{}, tx))
    })
}

func (r *PostgresOrderRepository) in(ctx context.Context) *db.Queries {
    if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
        return r.queries.WithTx(tx)
    }

    return r.queries
}
```

- The transaction commits when `fn` returns `nil` and rolls back when it returns an error.
- **Every repository method takes its queries through `in(ctx)`, from the component's first
  query**, whether or not anything opens a transaction yet. `Transactor` and its implementation
  arrive with the first use case that makes two writes, and by then no method has to change to
  join it.
- **A transaction belongs to one component.** Each component that needs one has its own
  `Transactor`, and no transaction covers two components' writes.

### Migrations

- A migration is a goose SQL file, `00001_create_orders.sql`, with a `-- +goose Up` section and a
  `-- +goose Down` section. The number is five digits, so the order goose runs the files in is the
  order `sqlc` reads them in.
- Each component applies its own:

  ```go
  //go:embed migrations/*.sql
  var embedded embed.FS

  func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
      migrations, err := fs.Sub(embedded, "migrations")
      if err != nil {
          return fmt.Errorf("orders migrations: %w", err)
      }

      sqlDB := stdlib.OpenDBFromPool(pool)
      defer func() { _ = sqlDB.Close() }()

      provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations, goose.WithTableName("goose_orders"))
      if err != nil {
          return fmt.Errorf("orders migrations: %w", err)
      }

      applied, err := provider.Up(ctx)
      if err != nil {
          return fmt.Errorf("orders migrations: %w", err)
      }

      slog.InfoContext(ctx, "migrated", "component", "orders", "applied", len(applied))

      return nil
  }
  ```

  The facade exports it for the composition root:

  ```go
  func Migrate(ctx context.Context, injector do.Injector) error {
      return infrastructure.Migrate(ctx, do.MustInvoke[*pgxpool.Pool](injector))
  }
  ```

- **Migrating is a command of the app: `<app> migrate` applies every component's migrations and
  exits, and nothing else the app does migrates.** A CLI adds `migrate` beside its other commands. A
  service checks for the argument before it serves, as below. The command applies the components'
  migrations in turn, stops at the first that fails, and logs how many each applied, so a second
  run shows `applied=0`:

  ```go
  func migrate(ctx context.Context, injector do.Injector) error {
      for _, apply := range []func(context.Context, do.Injector) error{
          orders.Migrate,
      } {
          if err := apply(ctx, injector); err != nil {
              return err
          }
      }

      return nil
  }
  ```

### The pool

The composition root opens the pool before it builds the injector and closes it after the injector
has shut down, because `pgxpool.Pool` has no method `samber/do` calls on shutdown:

```go
pool, err := openPool(ctx)
if err != nil {
    return err
}
defer pool.Close()

injector := do.New()
do.ProvideValue(injector, pool)

orders.Register(injector)

if len(os.Args) > 1 && os.Args[1] == "migrate" {
    return migrate(ctx, injector)
}
```

```go
func openPool(ctx context.Context) (*pgxpool.Pool, error) {
    url := os.Getenv("DATABASE_URL")
    if url == "" {
        return nil, errors.New("DATABASE_URL is not set")
    }

    config, err := pgxpool.ParseConfig(url)
    if err != nil {
        return nil, fmt.Errorf("read DATABASE_URL: %w", err)
    }

    config.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
        conn.TypeMap().RegisterType(&pgtype.Type{
            Name:  "timestamptz",
            OID:   pgtype.TimestamptzOID,
            Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
        })

        return nil
    }

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    return pool, nil
}
```

- **An unset `DATABASE_URL` stops the app as it starts.** Passed through empty, it lets the app
  start and fail at its first query.
- **Every connection returns a `timestamptz` in UTC.** Left alone, `pgx` returns it in the
  process's local time zone, and a response carries whatever offset the server happens to have.

### Enforcement

One `depguard` rule, beside the rules of ADR-GO-02:

```yaml
database-driver:
  files:
    - "**/internal/presentation/**"
  deny:
    - pkg: github.com/jackc/pgx
      desc: "the database is reached through a repository in infrastructure/ (ADR-GO-05)"
```

`domain/` and `application/` need no entry here: their strict allow-lists already reject `pgx`.

### Tests

- A use case's test passes it a hand-written fake repository (ADR-GO-01) and needs no database.
- **A repository's test runs against a real PostgreSQL**, on the server `TEST_DATABASE_URL` names,
  and is skipped when the variable is not set.
- **Each test makes a database of its own**, applies the component's migrations to it, and drops it
  when the test ends. A test never touches a database it did not make, so the variable may name
  the server the app's own database is on:

  ```go
  func database(t *testing.T) *pgxpool.Pool {
      t.Helper()

      url := os.Getenv("TEST_DATABASE_URL")
      if url == "" {
          t.Skip("TEST_DATABASE_URL is not set")
      }

      ctx := context.Background()

      server, err := pgx.Connect(ctx, url)
      if err != nil {
          t.Fatalf("connect: %v", err)
      }

      name := fmt.Sprintf("orders_test_%d", time.Now().UnixNano())
      quoted := pgx.Identifier{name}.Sanitize()

      if _, err := server.Exec(ctx, "CREATE DATABASE "+quoted); err != nil {
          t.Fatalf("create database: %v", err)
      }
      t.Cleanup(func() {
          _, _ = server.Exec(ctx, "DROP DATABASE "+quoted+" WITH (FORCE)")
          _ = server.Close(ctx)
      })

      config, err := pgxpool.ParseConfig(url)
      if err != nil {
          t.Fatalf("read TEST_DATABASE_URL: %v", err)
      }
      config.ConnConfig.Database = name

      pool, err := pgxpool.NewWithConfig(ctx, config)
      if err != nil {
          t.Fatalf("open the test database: %v", err)
      }
      t.Cleanup(pool.Close)

      if err := infrastructure.Migrate(ctx, pool); err != nil {
          t.Fatalf("migrate: %v", err)
      }

      return pool
  }
  ```

## Consequences

- A query that does not match the schema fails `sqlc generate`, before the code compiles. So does a
  query that reaches into another component's tables.
- Every change to the schema is a migration written by hand, and `sqlc` learns the schema from
  those files. There is no second description of the schema to keep in step.
- Generation is a step someone runs. Forgetting it leaves the committed code behind the SQL, which
  `sqlc diff` in CI reports.
- A repository call made inside `InTransaction` with a context other than the one `fn` received
  runs outside the transaction, and nothing reports it.
- The overrides cover `uuid` and `timestamptz`. Any other PostgreSQL type that `sqlc` maps to a
  `pgtype` type, such as `numeric` or `date`, arrives as that type until the project adds an
  override for it.
- `go test ./...` passes on a machine with no database, because the repository tests are skipped
  there. CI sets `TEST_DATABASE_URL`, or those tests never run.
- A test that is killed before it ends leaves its database on the server, named
  `<component>_test_` and a number.
- The test's pool is opened without the composition root's settings, so a test sees a
  `timestamptz` in the local zone. Compare instants, not zones.
- A deliberate `pgx` import in `presentation/` fails `golangci-lint run` **while still compiling**,
  the same proof ADR-GO-02 asks of the layer rules.
- This ADR decides nothing about caching, a search index, or a message queue. A project that needs
  one records it in an ADR of its own.

## Related

- ADR-BASE-01 — Clean Architecture (repository interfaces live in `application/`)
- ADR-BASE-02 — Package-by-component (the facade a shared table would go around)
- ADR-BASE-03 — Extraction readiness, on a surface that has it (each component owns its data; one
  transaction, one component)
- ADR-GO-01 — Dependency injection (the injector the pool is provided to; hand-written fakes)
- ADR-GO-02 — Boundary enforcement (why `application/` cannot import `pgx`)
- ADR-GO-03 — Optional values (`mo.Option` for a nullable column)
- ADR-GO-04 — HTTP delivery, on a surface that has it (the handler that gives the repository's
  domain error its status)
