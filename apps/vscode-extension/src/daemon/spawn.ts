import { type ChildProcessWithoutNullStreams, spawn } from "node:child_process";
import * as path from "node:path";
import type * as vscode from "vscode";

import { resolveDaemonPath, resolveTreeSitterPath } from "./paths";

export interface SpawnedDaemon {
  process: ChildProcessWithoutNullStreams;
  binaryPath: string;
}

export function spawnDaemon(context: vscode.ExtensionContext): SpawnedDaemon {
  const binaryPath = resolveDaemonPath(context);
  const treeSitterPath = resolveTreeSitterPath(context);
  const env = { ...process.env };
  if (!env.VOCODE_TREE_SITTER_BIN && treeSitterPath) {
    env.VOCODE_TREE_SITTER_BIN = treeSitterPath;
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
