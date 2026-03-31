import * as fs from "node:fs";
import * as path from "node:path";
import type * as vscode from "vscode";

export function getPlatformBinaryName(): string {
  if (process.platform === "win32") return "vocoded.exe";
  return "vocoded";
}

export function getPlatformToolBinaryName(baseName: string): string {
  if (process.platform === "win32") return `${baseName}.exe`;
  return baseName;
}

export function getPlatformBinarySubdir(): string {
  return `${process.platform}-${process.arch}`;
}

export function getDevDaemonPath(context: vscode.ExtensionContext): string {
  return path.resolve(
    context.extensionPath,
    "..",
    "..",
    "apps",
    "daemon",
    "bin",
    getPlatformBinarySubdir(),
    getPlatformBinaryName(),
  );
}

export function getProdDaemonPath(context: vscode.ExtensionContext): string {
  return path.join(
    context.extensionPath,
    "bin",
    getPlatformBinarySubdir(),
    getPlatformBinaryName(),
  );
}

export function resolveDaemonPath(context: vscode.ExtensionContext): string {
  const devPath = getDevDaemonPath(context);
  if (fs.existsSync(devPath)) {
    console.log(`[vocode] using dev daemon: ${devPath}`);
    return devPath;
  }

  const prodPath = getProdDaemonPath(context);
  if (fs.existsSync(prodPath)) {
    console.log(`[vocode] using bundled daemon: ${prodPath}`);
    return prodPath;
  }

  throw new Error(
    `Could not locate Vocode daemon binary for ${getPlatformBinarySubdir()}`,
  );
}

export function resolveTreeSitterPath(
  context: vscode.ExtensionContext,
): string | undefined {
  const binaryName = getPlatformToolBinaryName("tree-sitter");
  const rel = path.join(
    "tools",
    "tree-sitter",
    getPlatformBinarySubdir(),
    binaryName,
  );
  const devPath = path.resolve(context.extensionPath, "..", "..", rel);
  if (fs.existsSync(devPath)) {
    console.log(`[vocode] using dev tree-sitter: ${devPath}`);
    return devPath;
  }
  const bundledPath = path.join(context.extensionPath, rel);
  if (fs.existsSync(bundledPath)) {
    console.log(`[vocode] using bundled tree-sitter: ${bundledPath}`);
    return bundledPath;
  }
  return undefined;
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
