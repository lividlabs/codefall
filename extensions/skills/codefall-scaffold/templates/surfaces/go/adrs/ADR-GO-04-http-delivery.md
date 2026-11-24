# ADR-GO-04: HTTP Delivery

## Status

Accepted — <date>

## Context

A Go surface that serves HTTP needs a router, and its handlers need one way to report a failure.
ADR-BASE-01 puts both in `presentation/` and decides neither. This ADR applies to a surface that
answers HTTP requests, whatever else it does. A surface that answers none has no use for it.

The standard library's handler signature allows a bug that no tool reports. A handler writes an
error response and must then return. When the `return` is forgotten, the handler runs on into its
success path and writes a second response:

```go
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
    order, err := h.getOrder.Execute(r.Context(), r.PathValue("id"))
    if err != nil {
        http.Error(w, "order not found", http.StatusNotFound)
        // no return: the handler carries on with a zero order
    }
    json.NewEncoder(w).Encode(order)
}
```

This compiles, and `go vet` passes it.

The routers considered were the standard library's `http.ServeMux`, `chi`, Gin, Echo, and Fiber.
Gin, Echo, and Fiber each give a handler a context type of their own in place of
`http.ResponseWriter` and `*http.Request`, so every handler is written against the framework, and
Fiber does not run on `net/http` at all. `ServeMux` has matched methods and path parameters since
Go 1.22 and needs no dependency, but it has no route groups and no middleware stack, so each
project would write its own. `chi` is a router and nothing more. Its handlers are `http.Handler`s,
its middleware is `func(http.Handler) http.Handler`, and its sub-router is an `http.Handler` a
component can hand to the composition root.

## Decision

### Router

- **`chi` v5** (`github.com/go-chi/chi/v5`). The code stays on `net/http` types everywhere except
  where routes are registered.
- **chi is imported in two places only**: a component's `presentation/` package, which registers
  that component's routes, and the shared module `internal/shared/httpx/`, which builds the root
  router.
- A handler reads a path parameter with the standard library's `r.PathValue("id")`, which chi fills
  in. No handler imports chi.
- Middleware is `func(http.Handler) http.Handler`. There is no error-returning middleware type.

### Handlers return errors

`internal/shared/httpx/` holds an adapter the project owns. It is not a dependency.

```go
type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if err := h(w, r); err != nil {
        writeError(w, r, err)
    }
}
```

- **Every handler is an `httpx.Handler`.** It writes its response and returns `nil`, or it returns
  an error and writes nothing. A handler never writes an error response itself.

### One function turns an error into a response

```go
type Error struct {
    Status int
    Detail string
}

func NewError(status int, detail string) *Error {
    return &Error{Status: status, Detail: detail}
}

func (e *Error) Error() string {
    return fmt.Sprintf("%d %s: %s", e.Status, http.StatusText(e.Status), e.Detail)
}

type problem struct {
    Title  string `json:"title"`
    Status int    `json:"status"`
    Detail string `json:"detail,omitempty"`
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
    status, detail := http.StatusInternalServerError, ""

    var typed *Error
    if errors.As(err, &typed) {
        status, detail = typed.Status, typed.Detail
    }

    if status >= http.StatusInternalServerError {
        slog.ErrorContext(r.Context(), "request failed",
            "error", err.Error(),
            "method", r.Method,
            "path", r.URL.Path,
            "request_id", middleware.GetReqID(r.Context()),
        )
    }

    body, _ := json.Marshal(problem{Title: http.StatusText(status), Status: status, Detail: detail})

    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(status)
    _, _ = w.Write(body)
}
```

- **`*httpx.Error` carries the status and a detail that is safe to send to the client.**
  `errors.As` finds it through any depth of `%w` wrapping.
- **Any other error is a 500 with no detail.** Its text is logged and never sent. Logging is the
  standard library's `log/slog`, through the default logger the composition root sets.
- **The body is RFC 9457 problem details**, sent as `application/problem+json`: `title` is the
  status text, `status` is the code, and `detail` is the error's own.

  ```json
  {"title":"Not Found","status":404,"detail":"order not found"}
  ```

- **A panic and an unknown path get the same body**, a 500 and a 404, each written by
  `writeError`. `httpx` sets the router's not-found handler, and it holds its own recover
  middleware in place of chi's `Recoverer`, which prints the stack itself and sends an empty body.
  The middleware passes `writeError` an error holding the panic's value and its stack, so both are
  logged. It panics again on `http.ErrAbortHandler`, which is `net/http`'s own signal to abort a
  response.
- **A known path asked for with the wrong method keeps chi's own 405**, which has no body. chi's
  response carries the `Allow` header HTTP requires on a 405, and a handler set in its place loses
  the header. This is the one response `writeError` does not write.

### The handler gives a domain error its status

`domain/` and `application/` name no transport (ADR-GO-02, rule 5), so a use case returns a domain
error and never an `*httpx.Error`. The handler is where the two meet:

```go
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) error {
    order, err := h.getOrder.Execute(r.Context(), r.PathValue("id"))

    switch {
    case errors.Is(err, domain.ErrOrderNotFound):
        return httpx.NewError(http.StatusNotFound, "order not found")
    case err != nil:
        return err
    }

    return httpx.Encode(w, http.StatusOK, orderResponse{ID: order.ID, Total: order.Total})
}
```

A domain error the handler has no case for becomes a logged 500.

### JSON helpers

```go
func Decode[T any](r *http.Request) (T, error)
func Encode(w http.ResponseWriter, status int, value any) error
```

- **`Decode`** reads the request body into a `T`. A body that does not parse into one is a 400.
- **`Encode`** writes the response as `application/json`. It marshals before it writes, so a
  value that cannot be marshalled returns its error with nothing sent. A write that fails after
  that is dropped: the client has gone, and returning the error would send a second response.

### Each component registers its own routes

A component's routes live in `presentation/routes.go`, the one file in the component that imports
chi. Its paths are relative to where the composition root mounts them:

```go
func Routes(handlers *Handlers) http.Handler {
    router := chi.NewRouter()
    router.Method(http.MethodGet, "/{id}", httpx.Handler(handlers.GetOrder))

    return router
}
```

A handler depends on an interface for the use case it calls, declared in `presentation/`, where it
is consumed, with the one method the handler needs:

```go
type GetOrderUseCase interface {
    Execute(ctx context.Context, id string) (domain.Order, error)
}
```

ADR-GO-01 puts a gateway's interface in `application/` because a use case consumes it there. This
is the same rule one ring out.

The component's facade registers the use case under that interface and exports the routes as an
`http.Handler`:

```go
func Register(injector do.Injector) {
    do.Provide(injector, func(i do.Injector) (presentation.GetOrderUseCase, error) {
        return application.NewGetOrder(do.MustInvoke[application.OrderRepository](i)), nil
    })
}

func Routes(injector do.Injector) http.Handler {
    return presentation.Routes(
        presentation.NewHandlers(do.MustInvoke[presentation.GetOrderUseCase](injector)))
}
```

The composition root mounts each component on the root router and provides the server through the
injector, as it provides every other service (ADR-GO-01):

```go
do.Provide(injector, func(i do.Injector) (*http.Server, error) {
    router := httpx.NewRouter()
    router.Mount("/orders", orders.Routes(i))

    return &http.Server{Addr: address(), Handler: router, ReadHeaderTimeout: 5 * time.Second}, nil
})
```

- **`httpx.NewRouter()`** returns the root router as a `*chi.Mux`, so the composition root calls
  `Mount` on it without importing chi. It holds chi's `RequestID` middleware, then the recover
  middleware, the not-found handler, and one route of its own: `GET /healthz`, which answers 200
  with `{"status":"ok"}`.
- **The server is a service.** `*http.Server` has the `Shutdown(context.Context) error` method
  `samber/do` calls on shutdown, so requests in flight finish before the services under them close.
- **`address()`** is where the server listens, the port in `PORT` or 8080:

  ```go
  func address() string {
      if port := os.Getenv("PORT"); port != "" {
          return ":" + port
      }

      return ":8080"
  }
  ```

### Enforcement

One `depguard` rule, beside the layer rules of ADR-GO-02:

```yaml
http-router:
  files:
    - "!**/internal/presentation/**"
    - "!**/internal/shared/httpx/**"
  deny:
    - pkg: github.com/go-chi/chi
      desc: "chi is imported only where routes are registered (ADR-GO-04)"
```

`domain/` and `application/` need no entry here: their strict allow-lists already reject chi.

### Tests

- The standard library's `testing` and `net/http/httptest`, and no test library. A handler's test
  mounts the component's routes on `httpx.NewRouter()`, the way the composition root does, and
  passes the handlers a hand-written fake use case (ADR-GO-01).

## Consequences

- A handler reports an error by returning it. It has no error response to write first, so it has
  no `return` to forget after one.
- Replacing chi touches `routes.go` in each component and `router.go` in `httpx`. No handler
  changes.
- A route is registered as `router.Method(http.MethodGet, "/{id}", httpx.Handler(h.GetOrder))`.
  chi's shorter `router.Get` takes an `http.HandlerFunc`, which an `httpx.Handler` is not.
- A handler that writes part of a response and then returns an error sends two. `httpx.Encode`
  marshals first for this reason; a handler that writes by hand returns its errors before it writes.
- `writeError` logs through the default logger, so a test that reads the log replaces the default
  for its own duration.
- A client still meets error bodies this app did not write: a proxy or a load balancer in front of
  it answers a 502 or a 504 in its own format. One error shape is what the app sends, not all a
  client can receive.
- A deliberate chi import outside the two places fails `golangci-lint run` **while still
  compiling**, the same proof ADR-GO-02 asks of the layer rules.
- This ADR decides nothing about request validation, an OpenAPI document, gRPC, or a queue
  consumer. A project that needs one records it in an ADR of its own.

## Related

- ADR-BASE-01 — Clean Architecture (`presentation/` is where delivery lives)
- ADR-GO-01 — Dependency injection (the injector that provides the server; hand-written fakes)
- ADR-GO-02 — Boundary enforcement (why a use case cannot return an `*httpx.Error`)
- ADR-GO-03 — Optional values (errors stay `(T, error)` with `%w`, which `errors.As` reads here)
- [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) — Problem Details for HTTP APIs
