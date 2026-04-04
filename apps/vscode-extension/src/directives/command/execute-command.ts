import { spawn } from "node:child_process";
import path from "node:path";
import type { CommandDirective } from "@vocode/protocol";
import * as vscode from "vscode";

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

export type CommandStreamChunk = {
  stream: "stdout" | "stderr";
  text: string;
};

export async function runAllowedCommand(
  params: CommandDirective,
  onStreamChunk?: (chunk: CommandStreamChunk) => void,
): Promise<{
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

  if (params.detached === true) {
    return Promise.resolve(
      runDetachedInIntegratedTerminal(params.command, args, wd),
    );
  }

  const child = spawn(params.command, args, {
    windowsHide: true,
    cwd: wd !== "" ? wd : undefined,
  });

  let stdout = "";
  let stderr = "";
  let spawnErr: Error | undefined;

  child.stdout?.on("data", (d) => {
    const text = d.toString();
    stdout += text;
    onStreamChunk?.({ stream: "stdout", text });
  });
  child.stderr?.on("data", (d) => {
    const text = d.toString();
    stderr += text;
    onStreamChunk?.({ stream: "stderr", text });
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

    child.on("error", (err: Error) => {
      spawnErr = err;
      finish(-1);
    });
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

  if (spawnErr !== undefined) {
    return {
      ok: false,
      stdout,
      stderr,
      message: `failed to start process: ${spawnErr.message}`,
    };
  }

  const code = exitCode ?? -1;
  if (code !== 0) {
    const tail = stdout.trimEnd();
    const tailHint =
      tail.length > 0 && stderr.trim().length === 0
        ? ` (stdout: ${tail.length > 400 ? `${tail.slice(-400)}…` : tail})`
        : "";
    return {
      ok: false,
      stdout,
      stderr,
      message: `command exited with code ${code}${tailHint}`,
    };
  }

  return { ok: true, stdout, stderr, message: "" };
}

/**
 * Dev-server style commands: open a dedicated integrated terminal so the user can
 * read logs, use Expo's interactive menu, and stop with Ctrl+C or the trash icon.
 */
function runDetachedInIntegratedTerminal(
  command: string,
  args: string[],
  wd: string,
): {
  ok: boolean;
  stdout: string;
  stderr: string;
  message: string;
} {
  try {
    const argsForShell = sanitizeArgsForInteractiveTerminal(command, args);
    const line = argvToShellLine(command.trim(), argsForShell);
    const term = vscode.window.createTerminal({
      name: "Vocode",
      cwd: wd === "" ? undefined : wd,
    });
    term.sendText(line, true);
    term.show(true);
    return { ok: true, stdout: "", stderr: "", message: "" };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return {
      ok: false,
      stdout: "",
      stderr: "",
      message: msg,
    };
  }
}

/** Drop -NonInteractive for PowerShell in a real TTY so tools like Expo can prompt / show QR. */
function sanitizeArgsForInteractiveTerminal(
  command: string,
  args: string[],
): string[] {
  const base = path.basename(command).toLowerCase();
  const isPwsh =
    base === "powershell.exe" ||
    base === "powershell" ||
    base === "pwsh" ||
    base === "pwsh.exe";
  if (!isPwsh) {
    return args;
  }
  return args.filter((a) => a !== "-NonInteractive");
}

/** Join argv into one line for sendText, with minimal quoting for spaces/meta chars. */
function argvToShellLine(command: string, args: string[]): string {
  const q = (a: string) => {
    if (process.platform === "win32") {
      if (!/[\s"%^&<>|]/.test(a)) {
        return a;
      }
      return `"${a.replace(/"/g, '\\"')}"`;
    }
    if (!/[\s'"$`!\\]/.test(a)) {
      return a;
    }
    return `'${a.replace(/'/g, `'\\''`)}'`;
  };
  return [command, ...args.map(q)].join(" ");
}
