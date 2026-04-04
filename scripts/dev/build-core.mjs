import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const target = `${process.platform}-${process.arch}`;
const binary =
  process.platform === "win32" ? "vocode-cored.exe" : "vocode-cored";

// Emit next to the VS Code extension so `resolveDaemonPath` prod layout matches dev without a copy step.
const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.join(scriptDir, "..", "..");
const outDir = path.join(repoRoot, "apps", "vscode-extension", "bin", target);
mkdirSync(outDir, { recursive: true });

// Go needs a writable build cache. In some Windows shells, %LocalAppData% is
// missing, which makes `go build` fail. Point GOCACHE at a repo-local folder
// to keep builds self-contained and reliable.
// GOCACHE under apps/core (cwd when invoked via pnpm --filter @vocode/core build).
const goCache = path.join(process.cwd(), ".gocache");
mkdirSync(goCache, { recursive: true });

const result = spawnSync(
  "go",
  [
    "build",
    "-buildvcs=false",
    "-o",
    path.join(outDir, binary),
    "./cmd/vocode-cored",
  ],
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
