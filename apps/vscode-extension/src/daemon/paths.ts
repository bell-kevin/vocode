import * as fs from "node:fs";
import * as path from "node:path";
import type * as vscode from "vscode";

export function getPlatformBinaryName(): string {
  if (process.platform === "win32") return "vocode-cored.exe";
  return "vocode-cored";
}

export function getPlatformToolBinaryName(baseName: string): string {
  if (process.platform === "win32") return `${baseName}.exe`;
  return baseName;
}

export function getPlatformBinarySubdir(): string {
  return `${process.platform}-${process.arch}`;
}

/** Go build writes here (see scripts/dev/build-core.mjs); same path is used in the packaged VSIX. */
export function getBundledDaemonPath(context: vscode.ExtensionContext): string {
  return path.join(
    context.extensionPath,
    "bin",
    getPlatformBinarySubdir(),
    getPlatformBinaryName(),
  );
}

/** Legacy layout if someone still has apps/core/bin from an older build. */
export function getLegacyCoreDaemonPath(
  context: vscode.ExtensionContext,
): string {
  return path.resolve(
    context.extensionPath,
    "..",
    "..",
    "apps",
    "core",
    "bin",
    getPlatformBinarySubdir(),
    getPlatformBinaryName(),
  );
}

export function resolveDaemonPath(context: vscode.ExtensionContext): string {
  const bundled = getBundledDaemonPath(context);
  if (fs.existsSync(bundled)) {
    console.log(`[vocode] using daemon: ${bundled}`);
    return bundled;
  }

  const legacy = getLegacyCoreDaemonPath(context);
  if (fs.existsSync(legacy)) {
    console.log(`[vocode] using legacy apps/core/bin daemon: ${legacy}`);
    return legacy;
  }

  throw new Error(
    `Could not locate Vocode daemon binary for ${getPlatformBinarySubdir()}`,
  );
}

export function resolveRipgrepPath(
  context: vscode.ExtensionContext,
): string | undefined {
  const binaryName = getPlatformToolBinaryName("rg");
  const rel = path.join(
    "tools",
    "ripgrep",
    getPlatformBinarySubdir(),
    binaryName,
  );
  const devPath = path.resolve(context.extensionPath, "..", "..", rel);
  if (fs.existsSync(devPath)) {
    console.log(`[vocode] using dev ripgrep: ${devPath}`);
    return devPath;
  }
  const bundledPath = path.join(context.extensionPath, rel);
  if (fs.existsSync(bundledPath)) {
    console.log(`[vocode] using bundled ripgrep: ${bundledPath}`);
    return bundledPath;
  }
  return undefined;
}
