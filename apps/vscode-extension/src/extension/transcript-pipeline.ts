import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import { applyTranscriptResult } from "../transcript/apply-result";
import {
  mergeCarriedTranscriptParams,
  recordTranscriptApplyCycle,
} from "../transcript/carry";
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

    void (async () => {
      try {
        const pos = editor.selection.active;
        const result = await client.transcript(
          mergeCarriedTranscriptParams({
            text,
            activeFile,
            workspaceRoot: transcriptWorkspaceRoot(activeFile),
            cursorPosition: { line: pos.line, character: pos.character },
            contextSessionId: voiceSession.contextSessionId(),
          }),
        );
        const outcomes = await applyTranscriptResult(result, activeFile);
        recordTranscriptApplyCycle(result, outcomes);
        const firstFailed = outcomes.find((o) => o.status === "failed");
        if (!result.success || firstFailed) {
          const msg = !result.success
            ? FAILED_TO_PROCESS_TRANSCRIPT
            : firstFailed?.message && firstFailed.message !== "not attempted"
              ? firstFailed.message
              : "A directive failed to apply.";
          mainPanelStore.markError(pendingId, msg);
        } else {
          mainPanelStore.markHandled(pendingId, {
            summary: result.summary?.trim() || undefined,
          });
        }
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
