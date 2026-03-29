import { type ChildProcessWithoutNullStreams, spawn } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as vscode from "vscode";

import { applyVocodeSpawnEnvironment } from "../config/spawn-env";
import { resolveVoiceSidecarPath } from "./paths";
import { applyWorkspaceDotEnv } from "./workspace-env";

export interface SpawnedVoiceSidecar {
  process: ChildProcessWithoutNullStreams;
  binaryPath: string;
}

export async function spawnVoiceSidecar(
  context: vscode.ExtensionContext,
): Promise<SpawnedVoiceSidecar> {
  const binaryPath = resolveVoiceSidecarPath(context);

  const msysRoot = process.env.MSYS2_ROOT ?? "C:\\tools\\msys64";
  const mingw64Bin = path.join(msysRoot, "mingw64", "bin");
  const env = { ...process.env };
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  applyWorkspaceDotEnv(env, workspaceRoot);
  await applyVocodeSpawnEnvironment(context, env);

  const logProtocolStdout =
    vscode.workspace
      .getConfiguration("vocode")
      .get<boolean>("voiceSidecarLogProtocol") === true;

  console.log(
    "[vocode] voice sidecar spawn env:",
    "VOCODE_VOICE_VAD_DEBUG=",
    env.VOCODE_VOICE_VAD_DEBUG ?? "(unset)",
  );

  if (process.platform === "win32" && fs.existsSync(mingw64Bin)) {
    env.PATH = `${mingw64Bin};${env.PATH ?? ""}`;
  }

  const proc = spawn(binaryPath, [], {
    cwd: path.dirname(binaryPath),
    stdio: "pipe",
    env,
  });

  proc.stdout.on("data", (data: Buffer) => {
    if (logProtocolStdout) {
      console.log("[vocode-voiced stdout]", data.toString());
    }
  });

  proc.stderr.on("data", (data: Buffer) => {
    console.error(`[vocode-voiced stderr] ${data.toString()}`);
  });

  proc.on("error", (error: Error) => {
    console.error(`[vocode-voiced spawn error] ${error.message}`);
  });

  proc.on("exit", (code: number | null, signal: NodeJS.Signals | null) => {
    console.log(`vocode-voiced exited with code=${code} signal=${signal}`);
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
