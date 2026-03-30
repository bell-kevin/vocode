import * as vscode from "vscode";

import type { VoiceTranscriptParams } from "@vocode/protocol";

import type { ExtensionServices } from "../commands/services";
import { FAILED_TO_PROCESS_TRANSCRIPT } from "../transcript/messages";
import { transcriptWorkspaceRoot } from "../transcript/workspace-root";

/**
 * Binds voice sidecar events and transcript → daemon → apply flow to the given
 * client/sidecar pair. Call again only after replacing `services.client` and
 * `services.voiceSidecar` (e.g. backend restart).
 */
export function attachTranscriptPipeline(services: ExtensionServices): void {
  const { client, voiceSidecar, voiceSession, voiceStatus, mainPanelStore } =
    services;
  if (!client || !voiceSidecar) {
    return;
  }

  let inFlightTranscripts = 0;

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

    const pendingId = mainPanelStore.enqueueCommitted(evt.text);

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
    const text = evt.text;

    if (inFlightTranscripts === 0) {
      voiceStatus.setProcessing();
    }
    inFlightTranscripts++;

    mainPanelStore.markProcessing(pendingId);

    const pos = editor.selection.active;
    const vocodeCfg = vscode.workspace.getConfiguration("vocode");
    const daemonConfig: NonNullable<
      VoiceTranscriptParams["daemonConfig"]
    > = {
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
      cursorPosition: { line: pos.line, character: pos.character },
      contextSessionId: voiceSession.contextSessionId(),
      daemonConfig,
    };

    void (async () => {
      try {
        const result = await client.transcript(baseParams);

        if (!result.success) {
          mainPanelStore.markError(pendingId, FAILED_TO_PROCESS_TRANSCRIPT);
          return;
        }
        if ((result.directives?.length ?? 0) > 0) {
          mainPanelStore.markError(
            pendingId,
            "Daemon returned directives unexpectedly.",
          );
          return;
        }

        mainPanelStore.markHandled(pendingId, {
          summary: result.summary?.trim() || undefined,
        });
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : "Unknown error while running the transcript.";
        mainPanelStore.markError(pendingId, message);
      } finally {
        inFlightTranscripts = Math.max(0, inFlightTranscripts - 1);
        if (voiceSession.isRunning() && inFlightTranscripts === 0) {
          voiceStatus.setListening();
        }
      }
    })();
  });
}
