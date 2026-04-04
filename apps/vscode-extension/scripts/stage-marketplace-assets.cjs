"use strict";

/**
 * vocode-cored is built directly into this package (apps/vscode-extension/bin) by build-core.mjs.
 * This script merges in vocode-voiced (if built), copies ripgrep, LICENSE, and embeds protocol for the VSIX.
 *
 * Optional: pnpm --filter @vocode/voice build (Windows/macOS/Linux depending on host)
 * Repo root: pnpm provision:ripgrep (populates tools/ripgrep/<platform>-<arch>/)
 */

const fs = require("node:fs");
const path = require("node:path");

const extRoot = path.join(__dirname, "..");
const repoRoot = path.join(extRoot, "..", "..");
const target = `${process.platform}-${process.arch}`;

const voiceBinDir = path.join(repoRoot, "apps", "voice", "bin", target);
const rgDir = path.join(repoRoot, "tools", "ripgrep", target);
const outBinDir = path.join(extRoot, "bin", target);
const outRgDir = path.join(extRoot, "tools", "ripgrep", target);
const coreBinary =
  process.platform === "win32" ? "vocode-cored.exe" : "vocode-cored";
const corePath = path.join(outBinDir, coreBinary);

function rmrf(p) {
  fs.rmSync(p, { recursive: true, force: true });
}

function copyDir(src, dest) {
  if (!fs.existsSync(src)) {
    return false;
  }
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.cpSync(src, dest, { recursive: true });
  return true;
}

rmrf(path.join(extRoot, "tools", "ripgrep"));

if (!fs.existsSync(corePath)) {
  console.error(
    `[stage-marketplace-assets] Missing core daemon at ${corePath}\n` +
      "  Run: pnpm --filter @vocode/core build",
  );
  process.exit(1);
}

if (!copyDir(voiceBinDir, outBinDir)) {
  console.warn(
    `[stage-marketplace-assets] No voice sidecar at ${voiceBinDir} (voice features will not work in this VSIX until you build it).`,
  );
}

if (!copyDir(rgDir, outRgDir)) {
  console.error(
    `[stage-marketplace-assets] Missing ripgrep at ${rgDir}\n` +
      "  Run from repo root: pnpm provision:ripgrep",
  );
  process.exit(1);
}

const licenseSrc = path.join(repoRoot, "LICENSE");
const licenseDest = path.join(extRoot, "LICENSE");
if (fs.existsSync(licenseSrc)) {
  fs.copyFileSync(licenseSrc, licenseDest);
}

// Ship protocol inside dist/ (only dist/daemon/client.js requires it at runtime) so the VSIX needs no node_modules.
const protoDist = path.join(repoRoot, "packages", "protocol", "dist");
if (!fs.existsSync(protoDist)) {
  console.error(
    `[stage-marketplace-assets] Missing ${protoDist}\n` +
      "  Run: pnpm --filter @vocode/protocol build",
  );
  process.exit(1);
}
const protoPkg = path.join(extRoot, "dist", "protocol-pkg");
fs.rmSync(protoPkg, { recursive: true, force: true });
fs.mkdirSync(protoPkg, { recursive: true });
fs.cpSync(protoDist, path.join(protoPkg, "dist"), { recursive: true });
fs.writeFileSync(
  path.join(protoPkg, "package.json"),
  `${JSON.stringify(
    {
      name: "@vocode/protocol",
      version: "0.0.0",
      type: "commonjs",
      main: "./dist/index.js",
      types: "./dist/index.d.ts",
    },
    null,
    2,
  )}\n`,
);

const clientJs = path.join(extRoot, "dist", "daemon", "client.js");
if (fs.existsSync(clientJs)) {
  const code = fs.readFileSync(clientJs, "utf8");
  const needle = 'require("@vocode/protocol")';
  const replacement = 'require("../protocol-pkg")';
  if (!code.includes(needle)) {
    if (!code.includes(replacement)) {
      console.warn(
        `[stage-marketplace-assets] Expected ${needle} in daemon/client.js — skip protocol rewrite`,
      );
    }
  } else {
    fs.writeFileSync(clientJs, code.split(needle).join(replacement));
  }
}

console.log(
  `[stage-marketplace-assets] Staged bin + tools for ${target} → ${extRoot}`,
);
