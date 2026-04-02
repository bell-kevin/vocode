import type { VoiceTranscriptParams } from "@vocode/protocol";
import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import type { DaemonClient } from "../daemon/client";
import { FAILED_TO_PROCESS_TRANSCRIPT } from "./messages";
import {
  transcriptWorkspaceFolderOpen,
  transcriptWorkspaceRoot,
} from "./workspace-root";

/**
 * Runs `voice.transcript` for a pending sidebar row: document symbols, same params as the
 * voice pipeline, then {@link MainPanelStore.markHandled} / {@link markError}. Caller must
 * {@link MainPanelStore.enqueueCommitted} and {@link markProcessing} first.
 */
export async function runDaemonTranscriptForPendingId(
  services: ExtensionServices,
  client: DaemonClient,
  editor: vscode.TextEditor,
  pendingId: number,
  textForDaemon: string,
): Promise<void> {
  const { mainPanelStore, voiceSession } = services;

  mainPanelStore.beginVoiceTranscriptRpc(pendingId);
  try {
    const activeFile = editor.document.uri.fsPath;
    const sel = editor.selection;

    const vocodeCfg = vscode.workspace.getConfiguration("vocode");
    const daemonConfig: NonNullable<VoiceTranscriptParams["daemonConfig"]> = {
      sessionIdleResetMs: vocodeCfg.get<number>("sessionIdleResetMs", 1800000),
      maxGatheredBytes: 120_000,
      maxGatheredExcerpts: 12,
    };

    const baseParams = {
      text: textForDaemon,
      activeFile,
      focusedWorkspacePath: activeFile,
      workspaceRoot: transcriptWorkspaceRoot(activeFile),
      workspaceFolderOpen: transcriptWorkspaceFolderOpen(),
      cursorPosition: {
        line: editor.selection.active.line,
        character: editor.selection.active.character,
      },
      activeSelection: {
        startLine: sel.start.line,
        startChar: sel.start.character,
        endLine: sel.end.line,
        endChar: sel.end.character,
      },
      contextSessionId: voiceSession.contextSessionId(),
      daemonConfig,
    };

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

    const paramsWithSymbols: VoiceTranscriptParams = {
      ...baseParams,
      activeFileSymbols: (() => {
        const out: NonNullable<VoiceTranscriptParams["activeFileSymbols"]> = [];
        flattenSymbols(docSymbols, out);
        return out;
      })(),
    };

    const result = await client.transcript(paramsWithSymbols);

    if (
      result.success &&
      result.transcriptOutcome === "needs_workspace_folder"
    ) {
      const open = "Open Folder";
      const pick = await vscode.window.showInformationMessage(
        result.summary?.trim() || "Open a folder to use voice file selection.",
        open,
      );
      if (pick === open) {
        await vscode.commands.executeCommand(
          "workbench.action.files.openFolder",
        );
      }
    }

    if (!result.success) {
      mainPanelStore.markError(pendingId, FAILED_TO_PROCESS_TRANSCRIPT);
      return;
    }

    mainPanelStore.markHandled(pendingId, {
      summary: result.summary?.trim() || undefined,
      transcriptOutcome: result.transcriptOutcome,
      uiDisposition: result.uiDisposition,
      searchResults: result.searchResults,
      activeSearchIndex: result.activeSearchIndex ?? null,
      answerText: result.answerText ?? null,
      contextSessionId: paramsWithSymbols.contextSessionId,
    });
  } catch (err) {
    const message =
      err instanceof Error
        ? err.message
        : "Unknown error while running the transcript.";
    mainPanelStore.markError(pendingId, message);
  } finally {
    mainPanelStore.endVoiceTranscriptRpc(pendingId);
  }
}
