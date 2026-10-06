// Start the server. `PORT` defaults to 3000; `DB_PATH` defaults to data/app.db.
import { createApp } from "./server.js";

const port = Number(process.env.PORT ?? 3000);
const server = createApp();
server.listen(port, () => {
  console.log(`Forfeit is listening on http://localhost:${port}`);
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => server.close(() => process.exit(0)));
}
