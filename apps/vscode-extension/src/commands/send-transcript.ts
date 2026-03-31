import type { VoiceTranscriptParams } from "@vocode/protocol";
import * as vscode from "vscode";

import type { DaemonClient } from "../daemon/client";
import { FAILED_TO_PROCESS_TRANSCRIPT } from "../transcript/messages";
import { transcriptWorkspaceRoot } from "../transcript/workspace-root";
import type { ExtensionServices } from "./services";
import type { CommandDefinition } from "./types";

export const sendTranscriptCommand: CommandDefinition = {
  id: "vocode.sendTranscript",
  requiresDaemon: true,
  run: (client, services) => sendTranscript(client, services),
};

async function sendTranscript(
  client: DaemonClient,
  services: ExtensionServices,
): Promise<void> {
  if (!services.voiceSession.isRunning()) {
    void vscode.window.showWarningMessage(
      "Voice is not active. Run 'Vocode: Start Voice' first.",
    );
    return;
  }

  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    void vscode.window.showWarningMessage(
      "Open a text editor so Vocode can run edit directives against the active file.",
    );
    return;
  }

  const text = await vscode.window.showInputBox({
    title: "Vocode Voice Transcript",
    prompt: "Enter transcript text to send to the daemon",
    placeHolder: "Refactor this function to handle empty input safely",
    ignoreFocusOut: true,
  });

  if (!services.voiceSession.isRunning()) {
    services.voiceStatus.setIdle();
    return;
  }

  const trimmedText = text?.trim();
  if (!trimmedText) {
    return;
  }

  const activePath = editor.document.uri.fsPath;

  try {
    services.voiceStatus.setProcessing();

    const pos = editor.selection.active;
    const sel = editor.selection;
    const docSymbols = (await vscode.commands.executeCommand(
      "vscode.executeDocumentSymbolProvider",
      editor.document.uri,
    )) as vscode.DocumentSymbol[] | undefined;

    const flattenSymbols = (
      syms: vscode.DocumentSymbol[] | undefined,
      out: VoiceTranscriptParams["activeFileSymbols"],
    ) => {
      if (!syms) return;
      for (const s of syms) {
        out?.push({
          name: s.name,
          kind: String(s.kind),
          range: {
            startLine: s.range.start.line,
            startChar: s.range.start.character,
            endLine: s.range.end.line,
            endChar: s.range.end.character,
          },
          selectionRange: {
            startLine: s.selectionRange.start.line,
            startChar: s.selectionRange.start.character,
            endLine: s.selectionRange.end.line,
            endChar: s.selectionRange.end.character,
          },
        });
        if (s.children?.length) flattenSymbols(s.children, out);
      }
    };

    const vocodeCfg = vscode.workspace.getConfiguration("vocode");
    const daemonConfig: NonNullable<VoiceTranscriptParams["daemonConfig"]> = {
      maxPlannerTurns: vocodeCfg.get<number>("maxPlannerTurns", 8),
      maxIntentsPerBatch: vocodeCfg.get<number>("maxIntentsPerBatch", 16),
      maxIntentDispatchRetries: vocodeCfg.get<number>(
        "maxIntentDispatchRetries",
        2,
      ),
      maxContextRounds: vocodeCfg.get<number>("maxContextRounds", 2),
      maxContextBytes: vocodeCfg.get<number>("maxContextBytes", 12000),
      maxConsecutiveContextRequests: vocodeCfg.get<number>(
        "maxConsecutiveContextRequests",
        3,
      ),
      maxTranscriptRepairRpcs: vocodeCfg.get<number>(
        "maxTranscriptRepairRpcs",
        8,
      ),
      sessionIdleResetMs: vocodeCfg.get<number>("sessionIdleResetMs", 1800000),
      // Daemon defaults; these caps are not user-configurable today.
      maxGatheredBytes: 120_000,
      maxGatheredExcerpts: 12,
    };

    const baseParams = {
      text: trimmedText,
      activeFile: activePath,
      workspaceRoot: transcriptWorkspaceRoot(activePath),
      cursorPosition: { line: pos.line, character: pos.character },
      activeSelection: {
        startLine: sel.start.line,
        startChar: sel.start.character,
        endLine: sel.end.line,
        endChar: sel.end.character,
      },
      activeFileSymbols: (() => {
        const out: NonNullable<VoiceTranscriptParams["activeFileSymbols"]> = [];
        flattenSymbols(docSymbols, out);
        return out;
      })(),
      contextSessionId: services.voiceSession.contextSessionId(),
      daemonConfig,
    };

    const result = await client.transcript(baseParams);
    if (!result.success) {
      services.mainPanelStore.recordCompletedTranscript(trimmedText, {
        errorMessage: FAILED_TO_PROCESS_TRANSCRIPT,
      });
      return;
    }

    services.mainPanelStore.recordCompletedTranscript(trimmedText, {
      summary: result.summary?.trim() || undefined,
      transcriptOutcome: result.transcriptOutcome,
      searchResults: result.searchResults,
      activeSearchIndex: result.activeSearchIndex ?? null,
      answerText: result.answerText ?? null,
    });
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Failed to send transcript.";
    services.mainPanelStore.recordCompletedTranscript(trimmedText, {
      errorMessage: message,
    });
  } finally {
    if (services.voiceSession.isRunning()) {
      services.voiceStatus.setListening();
    } else {
      services.voiceStatus.setIdle();
    }
  }
}
