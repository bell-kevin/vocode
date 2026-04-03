import { spawn } from "node:child_process";
import path from "node:path";

import type { CommandDirective } from "@vocode/protocol";

/** Shell entrypoints only; real dev CLIs run inside -c / -Command. */
const allowedBasenames = new Set<string>([
  "cmd.exe",
  "cmd",
  "powershell.exe",
  "powershell",
  "pwsh",
  "pwsh.exe",
  "sh",
  "bash",
  "zsh",
  "fish",
]);

function allowedCommandName(cmd: string): boolean {
  const t = cmd.trim();
  if (!t) {
    return false;
  }
  const base = path.basename(t).toLowerCase();
  return allowedBasenames.has(base) || allowedBasenames.has(t.toLowerCase());
}

function validateCommandParams(params: CommandDirective): string | undefined {
  const cmd = params.command.trim();
  if (!cmd) {
    return "command cannot be empty";
  }

  if (/[\t\r\n ]/.test(cmd)) {
    return "command must be a single executable token";
  }

  if (!allowedCommandName(cmd)) {
    return "command must be an allowed shell (cmd, powershell, pwsh, sh, bash, zsh, fish)";
  }

  for (const arg of params.args ?? []) {
    if (arg.includes("\u0000")) {
      return "command args contain invalid characters";
    }
  }

  const wd = params.workingDirectory?.trim() ?? "";
  if (wd !== "") {
    if (!path.isAbsolute(wd)) {
      return "workingDirectory must be an absolute path";
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
  const wd = params.workingDirectory?.trim() ?? "";

  const child = spawn(params.command, args, {
    windowsHide: true,
    cwd: wd !== "" ? wd : undefined,
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

    child.on("error", () => finish(-1));
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
