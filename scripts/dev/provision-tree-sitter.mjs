import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(import.meta.url);

const root = process.cwd();
const platformTarget = `${process.platform}-${process.arch}`;
const binaryName =
  process.platform === "win32" ? "tree-sitter.exe" : "tree-sitter";
const targetDir = path.join(root, "tools", "tree-sitter", platformTarget);
const targetPath = path.join(targetDir, binaryName);

function resolveFromNpmPackage() {
  const out = [];
  try {
    const pkgJsonPath = require.resolve("tree-sitter-cli/package.json", {
      paths: [root],
    });
    const pkgDir = path.dirname(pkgJsonPath);
    const candidates = [
      path.join(pkgDir, "tree-sitter.exe"),
      path.join(pkgDir, "tree-sitter"),
      path.join(pkgDir, "bin", "tree-sitter"),
      path.join(pkgDir, "bin", "tree-sitter.exe"),
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
    const pkgJsonPath = require.resolve("tree-sitter-cli/package.json", {
      paths: [root],
    });
    const pkgDir = path.dirname(pkgJsonPath);
    const installScript = path.join(pkgDir, "install.js");
    if (!existsSync(installScript)) return false;
    const result = spawnSync(process.execPath, [installScript], {
      cwd: pkgDir,
      stdio: ["ignore", "pipe", "pipe"],
      encoding: "utf8",
    });
    if (result.status !== 0) {
      const stderr = (result.stderr ?? "").trim();
      if (stderr) {
        console.warn(`[vocode] tree-sitter-cli install failed: ${stderr}`);
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
  // Self-heal: tree-sitter-cli package exists but binary may not have been
  // fetched yet (e.g. stale install state). Try package install script once.
  if (ensureNpmPackageBinary()) {
    candidates.push(...resolveFromNpmPackage());
  }
}

if (candidates.length === 0) {
  console.warn(
    "[vocode] tree-sitter not provisioned: install `tree-sitter-cli` (pnpm)",
  );
  process.exit(0);
}

const sourcePath = candidates.find((candidate) => verifyBinary(candidate));
if (!sourcePath) {
  console.warn("[vocode] no executable tree-sitter binary found");
  console.warn(
    "[vocode] if using pnpm, run `pnpm approve-builds` and reinstall so tree-sitter-cli can fetch its native binary",
  );
  process.exit(0);
}

mkdirSync(targetDir, { recursive: true });
copyFileSync(sourcePath, targetPath);
console.log(`[vocode] provisioned tree-sitter: ${targetPath}`);
