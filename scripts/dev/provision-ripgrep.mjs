import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(import.meta.url);

const root = process.cwd();
const platformTarget = `${process.platform}-${process.arch}`;
const binaryName = process.platform === "win32" ? "rg.exe" : "rg";
const targetDir = path.join(root, "tools", "ripgrep", platformTarget);
const targetPath = path.join(targetDir, binaryName);

function resolveFromNpmPackage() {
  const out = [];
  try {
    const pkgJsonPath = require.resolve("@vscode/ripgrep/package.json", {
      paths: [root],
    });
    const pkgDir = path.dirname(pkgJsonPath);
    const candidates = [
      path.join(pkgDir, "bin", "rg.exe"),
      path.join(pkgDir, "bin", "rg"),
      path.join(pkgDir, "rg.exe"),
      path.join(pkgDir, "rg"),
    ];
    for (const candidate of candidates) {
      if (existsSync(candidate)) out.push(candidate);
    }
  } catch {
    // Optional dependency or package may be missing.
  }
  return out;
}

function ensureNpmPackageBinary() {
  try {
    const pkgJsonPath = require.resolve("@vscode/ripgrep/package.json", {
      paths: [root],
    });
    const pkgDir = path.dirname(pkgJsonPath);
    const postinstall = path.join(pkgDir, "lib", "postinstall.js");
    if (!existsSync(postinstall)) return false;
    const result = spawnSync(process.execPath, [postinstall], {
      cwd: pkgDir,
      stdio: ["ignore", "pipe", "pipe"],
      encoding: "utf8",
    });
    if (result.status !== 0) {
      const stderr = (result.stderr ?? "").trim();
      if (stderr) {
        console.warn(`[vocode] @vscode/ripgrep postinstall failed: ${stderr}`);
      }
      return false;
    }
    return true;
  } catch {
    return false;
  }
}

function verifyBinary(binaryPath) {
  const result = spawnSync(binaryPath, ["--version"], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  return result.status === 0;
}

const candidates = [];
candidates.push(...resolveFromNpmPackage());
if (candidates.length === 0) {
  // Self-heal: @vscode/ripgrep package exists but binary may not have been
  // fetched yet (e.g. pnpm ignored postinstall scripts). Try postinstall once.
  if (ensureNpmPackageBinary()) {
    candidates.push(...resolveFromNpmPackage());
  }
}
if (candidates.length === 0) {
  console.warn("[vocode] ripgrep not provisioned: install `@vscode/ripgrep` (pnpm)");
  process.exit(0);
}

const sourcePath = candidates.find((candidate) => verifyBinary(candidate));
if (!sourcePath) {
  console.warn("[vocode] no executable ripgrep binary found in @vscode/ripgrep");
  process.exit(0);
}

mkdirSync(targetDir, { recursive: true });
copyFileSync(sourcePath, targetPath);
console.log(`[vocode] provisioned ripgrep: ${targetPath}`);

