import { cpSync, existsSync } from "node:fs";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const standalone = path.join(root, ".next", "standalone");
if (!existsSync(path.join(standalone, "server.js"))) {
  throw new Error("Production build missing. Run npm run build first.");
}
for (const asset of ["public", ".next/static"]) {
  if (existsSync(path.join(root, asset)))
    cpSync(path.join(root, asset), path.join(standalone, asset), { recursive: true });
}
const server = spawn(process.execPath, [path.join(standalone, "server.js")], {
  cwd: root,
  stdio: "inherit",
  windowsHide: true,
  env: { ...process.env, HOSTNAME: process.env.HOSTNAME || "127.0.0.1", PORT: process.env.PORT || "3000" },
});
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => server.kill(signal));
server.on("error", (error) => {
  console.error(error.message);
  process.exitCode = 1;
});
server.on("exit", (code) => {
  process.exitCode = code ?? 1;
});
