import type { CommandDirective } from "@vocode/protocol";
import * as vscode from "vscode";

import type { TranscriptApplyContext } from "../../voice-transcript/context";
import type { DirectiveDispatchOutcome } from "../dispatch";
import { runAllowedCommand } from "./execute-command";

function commandDirectiveDisplayLine(d: CommandDirective): string {
  const cmd = d.command.trim();
  const parts = [cmd, ...(d.args ?? [])];
  return parts.join(" ");
}

/** Runs one allowed command directive (extension executes; daemon validated shape). */
export async function dispatchCommand(
  params: CommandDirective | undefined,
  ctx: TranscriptApplyContext,
): Promise<DirectiveDispatchOutcome> {
  if (!params) {
    return { ok: false, message: "missing command directive" };
  }

  const ui = ctx.commandApplyUi;
  const displayLine = commandDirectiveDisplayLine(params);
  if (ui !== undefined) {
    ui.onStart(displayLine);
  }

  const outcome = await runAllowedCommand(params, (chunk) => {
    if (ui === undefined) {
      return;
    }
    const prefix = chunk.stream === "stderr" ? "[stderr] " : "";
    ui.onOutput(prefix + chunk.text);
  });
  if (!outcome.ok) {
    const stderr = outcome.stderr.trim();
    return {
      ok: false,
      message: stderr
        ? `${(outcome.message ?? "command failed").trim()}: ${stderr}`
        : (outcome.message?.trim() ??
          "command exited non-zero or failed to run"),
    };
  }
  if (params.detached === true) {
    void vscode.window.showInformationMessage(
      "Vocode: opened a terminal tab — check the Terminal panel for output; press Ctrl+C there to stop the dev server.",
    );
    return { ok: true };
  }
  const stdoutLine = outcome.stdout.trim();
  if (stdoutLine.length > 0) {
    void vscode.window.showInformationMessage(`Vocode: ${stdoutLine}`);
  }
  return { ok: true };
}
