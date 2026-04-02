import * as vscode from "vscode";

import type { ExtensionServices } from "../commands/services";
import type { VoiceSidecarConfigPatch } from "../voice/client";
import { runDaemonTranscriptForPendingId } from "./run-daemon-transcript";

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
    const clarify = mainPanelStore.consumeClarifyPromptAnswerForSend(evt.text);
    const clarifyTextToSend = clarify?.sendText ?? undefined;
    // If this utterance is answering a clarification prompt, attribute the completion
    // to the original instruction (not the navigation/answer filler).
    const pendingId =
      clarify !== null
        ? mainPanelStore.enqueueCommitted(clarify.displayText)
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

    const text = clarifyTextToSend ?? evt.text;

    if (inFlightTranscripts === 0) {
      voiceStatus.setProcessing();
    }
    inFlightTranscripts++;

    mainPanelStore.markProcessing(pendingId);

    void (async () => {
      try {
        await runDaemonTranscriptForPendingId(
          services,
          client,
          editor,
          pendingId,
          text,
        );
      } finally {
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
