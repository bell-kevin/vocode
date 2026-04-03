import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import path from "node:path";

// Keep go run happy in shells where %LocalAppData% is missing.
const goCache = path.join(process.cwd(), ".gocache");
mkdirSync(goCache, { recursive: true });

const result = spawnSync("go", ["run", "./cmd/vocode-cored"], {
  env: {
    ...process.env,
    GOCACHE: goCache,
  },
  stdio: "inherit",
});

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}
