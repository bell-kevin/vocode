import { type ChildProcessWithoutNullStreams, spawn } from "node:child_process";
import * as path from "node:path";
import type * as vscode from "vscode";

import { applyVocodeSpawnEnvironment } from "../config/spawn-env";
import { resolveDaemonPath, resolveRipgrepPath, resolveTreeSitterPath } from "./paths";

export interface SpawnedDaemon {
  process: ChildProcessWithoutNullStreams;
  binaryPath: string;
}

export async function spawnDaemon(
  context: vscode.ExtensionContext,
): Promise<SpawnedDaemon> {
  const binaryPath = resolveDaemonPath(context);
  const treeSitterPath = resolveTreeSitterPath(context);
  const ripgrepPath = resolveRipgrepPath(context);
  const env = { ...process.env };
  await applyVocodeSpawnEnvironment(context, env);
  // Always use extension-resolved provisioned binary; ignore inherited env overrides.
  if (treeSitterPath) {
    env.VOCODE_TREE_SITTER_BIN = treeSitterPath;
  } else {
    delete env.VOCODE_TREE_SITTER_BIN;
  }
  if (ripgrepPath) {
    env.VOCODE_RG_BIN = ripgrepPath;
  } else {
    delete env.VOCODE_RG_BIN;
  }

  const proc = spawn(binaryPath, [], {
    cwd: path.dirname(binaryPath),
    stdio: "pipe",
    env,
  });

  proc.stdout.on("data", (data: Buffer) => {
    console.log("[vocoded stdout]", data.toString());
  });

  proc.stderr.on("data", (data: Buffer) => {
    console.error(`[vocoded stderr] ${data.toString()}`);
  });

  proc.on("error", (error: Error) => {
    console.error(`[vocoded spawn error] ${error.message}`);
  });

  proc.on("exit", (code: number | null, signal: NodeJS.Signals | null) => {
    console.log(`vocoded exited with code=${code} signal=${signal}`);
  });

  context.subscriptions.push({
    dispose: () => {
      if (!proc.killed) {
        proc.kill();
      }
    },
  });

  return {
    process: proc,
    binaryPath,
  };
}
