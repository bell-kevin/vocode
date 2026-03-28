import { spawn } from "node:child_process";
import type { CommandDirective } from "@vocode/protocol";

const allowedExecutables = new Set<string>([
  "cmd.exe",
  "powershell.exe",
  "powershell",
  "pwsh",
  // `echo` is a builtin on Windows; but the daemon stub uses `cmd.exe /c echo`.
  // Keeping `echo` allows the unix-like environments where `echo` is a real binary.
  "echo",
]);

function validateCommandParams(params: CommandDirective): string | undefined {
  const cmd = params.command.trim();
  if (!cmd) {
    return "command cannot be empty";
  }

  // Match daemon policy: command must be a single executable token.
  if (/[\t\r\n ]/.test(cmd)) {
    return "command must be a single executable name";
  }

  if (!allowedExecutables.has(cmd.toLowerCase())) {
    return "command is not allowed";
  }

  for (const arg of params.args ?? []) {
    if (arg.includes("\u0000")) {
      return "command args contain invalid characters";
    }
  }

  return undefined;
}

export async function runAllowedCommand(params: CommandDirective): Promise<{
  ok: boolean;
  stdout: string;
  stderr: string;
  message: string;
}> {
  const err = validateCommandParams(params);
  if (err) {
    return { ok: false, stdout: "", stderr: "", message: err };
  }

  const args = params.args ?? [];

  const child = spawn(params.command, args, {
    windowsHide: true,
  });

  let stdout = "";
  let stderr = "";

  child.stdout?.on("data", (d) => {
    stdout += d.toString();
  });
  child.stderr?.on("data", (d) => {
    stderr += d.toString();
  });

  let timedOut = false;
  let timeoutHandle: NodeJS.Timeout | undefined;
  if (params.timeoutMs != null && params.timeoutMs > 0) {
    timeoutHandle = setTimeout(() => {
      timedOut = true;
      child.kill();
    }, params.timeoutMs);
  }

  const exitCode: number | null = await new Promise((resolve) => {
    let resolved = false;

    const finish = (code: number | null) => {
      if (resolved) return;
      resolved = true;
      resolve(code);
    };

    child.on("error", () => finish(-1)); // Catch spawn errors
    child.on("close", (code) => finish(code));
  });

  if (timeoutHandle) {
    clearTimeout(timeoutHandle);
  }

  if (timedOut) {
    return {
      ok: false,
      stdout,
      stderr,
      message: "command timed out",
    };
  }

  const code = exitCode ?? -1;
  if (code !== 0) {
    return {
      ok: false,
      stdout,
      stderr,
      message: `command exited with code ${code}`,
    };
  }

  return { ok: true, stdout, stderr, message: "" };
}
