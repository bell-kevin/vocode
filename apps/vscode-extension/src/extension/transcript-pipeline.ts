import type { VoiceTranscriptParams } from "@vocode/protocol";
import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import { FAILED_TO_PROCESS_TRANSCRIPT } from "../transcript/messages";
import { transcriptWorkspaceRoot } from "../transcript/workspace-root";
import type { VoiceSidecarConfigPatch } from "../voice/client";

/**
 * Binds voice sidecar events and transcript → daemon → apply flow to the given
 * client/sidecar pair. Call again only after replacing `services.client` and
 * `services.voiceSidecar` (e.g. backend restart).
 */
export function attachTranscriptPipeline(
  services: ExtensionServices,
): vscode.Disposable {
  const { client, voiceSidecar, voiceSession, voiceStatus, mainPanelStore } =
    services;
  if (!client || !voiceSidecar) {
    return { dispose: () => {} };
  }

  let inFlightTranscripts = 0;

  const buildVoiceSidecarConfigPatch = (): VoiceSidecarConfigPatch => {
    const vocodeCfg = vscode.workspace.getConfiguration("vocode");
    return {
      // Secret stored in SecretStorage; we do not read it here.
      sttModelId: vocodeCfg.get<string>(
        "elevenLabsSttModelId",
        "scribe_v2_realtime",
      ),
      sttLanguage: vocodeCfg.get<string>("elevenLabsSttLanguage", "en"),
      vadDebug: vocodeCfg.get<boolean>("voiceVadDebug", false) === true,
      vadThresholdMultiplier: vocodeCfg.get<number>(
        "voiceVadThresholdMultiplier",
        1.65,
      ),
      vadMinEnergyFloor: vocodeCfg.get<number>("voiceVadMinEnergyFloor", 100),
      vadStartMs: vocodeCfg.get<number>("voiceVadStartMs", 60),
      vadEndMs: vocodeCfg.get<number>("voiceVadEndMs", 750),
      vadPrerollMs: vocodeCfg.get<number>("voiceVadPrerollMs", 320),
      sttCommitResponseTimeoutMs: vocodeCfg.get<number>(
        "voiceSttCommitResponseTimeoutMs",
        5000,
      ),
      streamMinChunkMs: vocodeCfg.get<number>("voiceStreamMinChunkMs", 200),
      streamMaxChunkMs: vocodeCfg.get<number>("voiceStreamMaxChunkMs", 500),
      streamMaxUtteranceMs: vocodeCfg.get<number>(
        "voiceStreamMaxUtteranceMs",
        0,
      ),
    };
  };

  // Push initial sidecar tuning, then keep it in sync without restarting `vocode-voiced`.
  let lastSentSerialized: string | undefined;
  const sendConfig = (): void => {
    const patch = buildVoiceSidecarConfigPatch();
    const serialized = JSON.stringify(patch);
    if (serialized === lastSentSerialized) {
      return;
    }
    lastSentSerialized = serialized;
    voiceSidecar.setConfig(patch);
  };

  sendConfig();

  let configTimeout: NodeJS.Timeout | undefined;
  const configListener = vscode.workspace.onDidChangeConfiguration((e) => {
    if (!e.affectsConfiguration("vocode")) {
      return;
    }
    if (configTimeout) {
      clearTimeout(configTimeout);
    }
    configTimeout = setTimeout(() => {
      sendConfig();
    }, 200);
  });

  voiceSidecar.onAudioMeter((evt) => {
    mainPanelStore.setAudioMeter(evt.speaking, evt.rms);
  });

  voiceSidecar.onError((evt) => {
    const message =
      typeof evt.message === "string" ? evt.message : "unknown error";
    mainPanelStore.setVoiceListening(false);
    voiceStatus.setIdle();
    if (voiceSession.isRunning()) {
      voiceSession.stop();
      void vscode.window.showWarningMessage(
        `Vocode voice sidecar error: ${message}`,
      );
    }
    voiceSidecar.stop();
  });

  voiceSidecar.onState((evt) => {
    if (evt.state !== "stopped" && evt.state !== "shutdown") {
      return;
    }
    if (evt.state === "shutdown") {
      configListener.dispose();
    }
    if (!voiceSession.isRunning()) {
      return;
    }
    voiceSession.stop();
    voiceStatus.setIdle();
    mainPanelStore.setVoiceListening(false);
  });

  voiceSidecar.onTranscript((evt) => {
    if (evt.committed !== true) {
      mainPanelStore.onPartial(evt.text);
      if (!voiceSession.isRunning()) {
        return;
      }
      return;
    }

    // If the daemon previously asked a clarify question, treat the next committed utterance
    // as the answer and send a combined instruction to the daemon.
    const clarifyTextToSend =
      mainPanelStore.consumeClarifyPromptAnswer(evt.text) ?? undefined;
    // If this utterance is answering a clarification prompt, do not enqueue it as its own transcript.
    const pendingId =
      clarifyTextToSend !== undefined
        ? mainPanelStore.enqueueCommitted("Answering clarification…")
        : mainPanelStore.enqueueCommitted(evt.text);

    if (!voiceSession.isRunning()) {
      return;
    }

    if (pendingId === null) {
      return;
    }

    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      const message =
        "Open a text editor so Vocode can run actions against the active file.";
      mainPanelStore.markError(pendingId, message);
      void vscode.window.showWarningMessage(message);
      return;
    }

    const activeFile = editor.document.uri.fsPath;
    const sel = editor.selection;
    const text = clarifyTextToSend ?? evt.text;

    if (inFlightTranscripts === 0) {
      voiceStatus.setProcessing();
    }
    inFlightTranscripts++;

    mainPanelStore.markProcessing(pendingId);

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
      text,
      activeFile,
      workspaceRoot: transcriptWorkspaceRoot(activeFile),
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

    void (async () => {
      mainPanelStore.beginVoiceTranscriptRpc(pendingId);
      try {
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
            const out: NonNullable<VoiceTranscriptParams["activeFileSymbols"]> =
              [];
            flattenSymbols(docSymbols, out);
            return out;
          })(),
        };

        const result = await client.transcript(paramsWithSymbols);

        if (!result.success) {
          mainPanelStore.markError(pendingId, FAILED_TO_PROCESS_TRANSCRIPT);
          return;
        }

        mainPanelStore.markHandled(pendingId, {
          summary: result.summary?.trim() || undefined,
          transcriptOutcome: result.transcriptOutcome,
          searchResults: result.searchResults,
          activeSearchIndex: result.activeSearchIndex ?? null,
          answerText: result.answerText ?? null,
        });
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : "Unknown error while running the transcript.";
        mainPanelStore.markError(pendingId, message);
      } finally {
        mainPanelStore.endVoiceTranscriptRpc(pendingId);
        inFlightTranscripts = Math.max(0, inFlightTranscripts - 1);
        if (voiceSession.isRunning() && inFlightTranscripts === 0) {
          voiceStatus.setListening();
        }
      }
    })();
  });

  return {
    dispose: () => {
      configListener.dispose();
      if (configTimeout) {
        clearTimeout(configTimeout);
      }
      voiceSidecar.onAudioMeter(() => {});
      voiceSidecar.onError(() => {});
      voiceSidecar.onState(() => {});
      voiceSidecar.onTranscript(() => {});
    },
  };
}
