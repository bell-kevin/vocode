import type { VoiceTranscriptParams } from "@vocode/protocol";
import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import type { DaemonClient } from "../daemon/client";
import {
  FAILED_TO_PROCESS_TRANSCRIPT,
  userFacingTranscriptRpcError,
} from "./messages";
import {
  transcriptWorkspaceFolderOpen,
  transcriptWorkspaceRoot,
  transcriptWorkspaceRootWithoutActiveEditor,
} from "./workspace-root";

/**
 * Runs `voice.transcript` for a pending sidebar row: document symbols when an editor is active,
 * then {@link MainPanelStore.markHandled} / {@link markError}. Caller must
 * {@link MainPanelStore.enqueueCommitted} and {@link markProcessing} first.
 *
 * `editor` may be undefined (e.g. user focused the panel or Explorer): workspace root comes from
 * the first open folder when available; routes that need an active file fail on the daemon side.
 */
export async function runDaemonTranscriptForPendingId(
  services: ExtensionServices,
  client: DaemonClient,
  editor: vscode.TextEditor | undefined,
  pendingId: number,
  textForDaemon: string,
): Promise<void> {
  const { mainPanelStore, voiceSession } = services;

  mainPanelStore.beginVoiceTranscriptRpc(pendingId);
  try {
    const vocodeCfg = vscode.workspace.getConfiguration("vocode");
    const daemonConfig: NonNullable<VoiceTranscriptParams["daemonConfig"]> = {
      sessionIdleResetMs: vocodeCfg.get<number>("sessionIdleResetMs", 1800000),
      maxGatheredBytes: 120_000,
      maxGatheredExcerpts: 12,
    };

    let paramsWithSymbols: VoiceTranscriptParams;

    if (editor) {
      const activeFile = editor.document.uri.fsPath;
      const sel = editor.selection;

      const baseParams = {
        text: textForDaemon,
        activeFile,
        focusedWorkspacePath: activeFile,
        workspaceRoot: transcriptWorkspaceRoot(activeFile),
        hostPlatform: process.platform,
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

      paramsWithSymbols = {
        ...baseParams,
        activeFileSymbols: (() => {
          const out: NonNullable<VoiceTranscriptParams["activeFileSymbols"]> =
            [];
          flattenSymbols(docSymbols, out);
          return out;
        })(),
      };
    } else {
      const root = transcriptWorkspaceRootWithoutActiveEditor();
      paramsWithSymbols = {
        text: textForDaemon,
        workspaceRoot: root,
        focusedWorkspacePath: root,
        hostPlatform: process.platform,
        workspaceFolderOpen: transcriptWorkspaceFolderOpen(),
        contextSessionId: voiceSession.contextSessionId(),
        daemonConfig,
      };
    }

    const result = await client.transcript(paramsWithSymbols);

    if (result.success && result.workspace?.needsFolder === true) {
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
      const errMsg = result.summary?.trim() || FAILED_TO_PROCESS_TRANSCRIPT;
      mainPanelStore.markError(pendingId, errMsg);
      return;
    }

    mainPanelStore.markHandled(pendingId, {
      summary: result.summary?.trim() || undefined,
      uiDisposition: result.uiDisposition,
      search: result.search,
      question: result.question,
      clarify: result.clarify,
      fileSelection: result.fileSelection,
      workspace: result.workspace,
      contextSessionId: paramsWithSymbols.contextSessionId,
    });
  } catch (err) {
    mainPanelStore.markError(pendingId, userFacingTranscriptRpcError(err));
  } finally {
    mainPanelStore.endVoiceTranscriptRpc(pendingId);
  }
}
