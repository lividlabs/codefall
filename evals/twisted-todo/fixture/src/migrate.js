// Apply pending migrations and exit. `scripts/local.sh update` runs this.
import { openDatabase, migrate, DEFAULT_DB_PATH } from "./db.js";

const db = openDatabase(DEFAULT_DB_PATH);
const applied = migrate(db);
db.close();
console.log(applied.length ? `applied: ${applied.join(", ")}` : "migrations: nothing to do");
