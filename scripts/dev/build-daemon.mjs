import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import path from "node:path";

const target = `${process.platform}-${process.arch}`;
const binary = process.platform === "win32" ? "vocoded.exe" : "vocoded";

mkdirSync(path.join("bin", target), { recursive: true });

// Go needs a writable build cache. In some Windows shells, %LocalAppData% is
// missing, which makes `go build` fail. Point GOCACHE at a repo-local folder
// to keep builds self-contained and reliable.
const goCache = path.join(process.cwd(), ".gocache");
mkdirSync(goCache, { recursive: true });

const result = spawnSync(
  "go",
  ["build", "-buildvcs=false", "-o", path.join("bin", target, binary), "./cmd/vocoded"],
  {
    env: {
      ...process.env,
      GOCACHE: goCache,
    },
    stdio: "inherit",
  },
);

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}
