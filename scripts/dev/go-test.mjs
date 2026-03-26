import { spawn } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..", "..");
const defaultCacheDir = path.join(repoRoot, ".cache", "go-build");

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true });
}

function main() {
  const env = { ...process.env };
  const isWin = process.platform === "win32";

  // Windows uses "Path" (case-insensitive) but Node's env object keys can vary.
  // Preserve whichever one exists, and keep them in sync when we modify PATH.
  const getPath = () => env.Path ?? env.PATH ?? "";
  const setPath = (value) => {
    if ("Path" in env || isWin) env.Path = value;
    env.PATH = value;
  };

  const args = process.argv.slice(2);
  const forceCgo = args.includes("--force-cgo");
  const goArgs = forceCgo ? args.filter((a) => a !== "--force-cgo") : args;

  if (!env.GOCACHE || env.GOCACHE.trim() === "") {
    ensureDir(defaultCacheDir);
    env.GOCACHE = defaultCacheDir;
  }

  // Go still may want TEMP/TMP; ensure at least one exists.
  if (!env.TEMP && !env.TMP) {
    env.TEMP = os.tmpdir();
  }

  if (forceCgo) {
    env.CGO_ENABLED = "1";
  }

  if (forceCgo && process.platform === "win32") {
    const msysRoot = process.env.MSYS2_ROOT ?? "C:\\tools\\msys64";
    const mingwBin = path.join(msysRoot, "mingw64", "bin");
    const mingwPkgConfigDir = path.join(
      msysRoot,
      "mingw64",
      "lib",
      "pkgconfig",
    );
    const pkgConfigExe = path.join(mingwBin, "pkg-config.exe");
    const gccExe = path.join(mingwBin, "gcc.exe");

    const ok =
      fs.existsSync(pkgConfigExe) &&
      fs.existsSync(mingwPkgConfigDir) &&
      fs.existsSync(gccExe);

    if (ok) {
      const pathSep = ";";
      setPath(mingwBin + pathSep + getPath());
      env.PKG_CONFIG_PATH =
        mingwPkgConfigDir +
        (env.PKG_CONFIG_PATH ? pathSep + env.PKG_CONFIG_PATH : "");
      env.PKG_CONFIG = pkgConfigExe;
      env.CC = gccExe;
    }
  }

  const finalGoArgs = goArgs.length > 0 ? goArgs : ["test", "./..."];

  const child = spawn("go", finalGoArgs, {
    stdio: "inherit",
    env,
    windowsHide: true,
  });

  child.on("exit", (code) => {
    process.exit(code ?? 1);
  });

  child.on("error", (err) => {
    console.error(err);
    process.exit(1);
  });
}

main();
