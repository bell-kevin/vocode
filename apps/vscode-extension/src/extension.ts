import * as vscode from "vscode";

import { registerAllCommands } from "./commands";
import { presentTranscriptResult } from "./commands/send-transcript/present-result";
import {
  type ExtensionServices,
  VoiceSessionController,
} from "./commands/services";
import { DaemonClient } from "./daemon/client";
import { spawnDaemon } from "./daemon/spawn";
import { VoiceStatusIndicator } from "./ui/status-bar";
import {
  TranscriptPanelViewProvider,
  transcriptPanelViewType,
} from "./ui/transcript-panel";
import { MicrophoneCapture } from "./voice/microphone";
import { TranscriptStore } from "./voice/transcript-store";
import { VoiceSidecarClient } from "./voice-sidecar/client";
import { spawnVoiceSidecar } from "./voice-sidecar/spawn";

function workspaceRootPath(): string | undefined {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

function createServices(
  context: vscode.ExtensionContext,
  voiceStatus: VoiceStatusIndicator,
  transcriptStore: TranscriptStore,
): ExtensionServices {
  const microphone = new MicrophoneCapture();
  const debugAudioLogging = vscode.workspace
    .getConfiguration("vocode")
    .get<boolean>("debugAudioLogging", false);

  const audioChunkSubscription = microphone.onAudioChunk(({ data }) => {
    if (debugAudioLogging) {
      console.debug(
        `Vocode microphone chunk captured (${data.byteLength} bytes)`,
      );
    }
  });

  context.subscriptions.push(microphone, audioChunkSubscription);
  try {
    const daemon = spawnDaemon(context);
    console.log(`Vocode daemon started from ${daemon.binaryPath}`);

    const voice = spawnVoiceSidecar(context);
    console.log(`Vocode voice sidecar started from ${voice.binaryPath}`);

    const voiceSession = new VoiceSessionController();
    const client = new DaemonClient(daemon.process);
    const voiceSidecar = new VoiceSidecarClient(voice.process);

    let inFlightTranscripts = 0;

    voiceSidecar.onAudioMeter((evt) => {
      transcriptStore.setAudioMeter(evt.speaking, evt.rms);
    });

    voiceSidecar.onError((evt) => {
      const message =
        typeof evt.message === "string" ? evt.message : "unknown error";
      // Always clear partial / meter UI — do not gate on isRunning (avoids a stuck "Live" card).
      transcriptStore.setVoiceListening(false);
      voiceStatus.setIdle();
      if (voiceSession.isRunning()) {
        voiceSession.stop();
        void vscode.window.showWarningMessage(
          `Vocode voice sidecar error: ${message}`,
        );
      }
      // Sync stdin protocol so a future start is not confused with an orphaned session.
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
      transcriptStore.setVoiceListening(false);
    });

    voiceSidecar.onTranscript((evt) => {
      if (evt.committed !== true) {
        transcriptStore.onPartial(evt.text);
        if (!voiceSession.isRunning()) {
          return;
        }
        return;
      }

      const pendingId = transcriptStore.enqueueCommitted(evt.text);

      // Daemon work only while listening; the store still records late commits for the panel.
      if (!voiceSession.isRunning()) {
        return;
      }

      if (pendingId === null) {
        return;
      }

      // Only final/committed transcript hypotheses should be forwarded to the core daemon.
      const editor = vscode.window.activeTextEditor;
      if (!editor) {
        transcriptStore.markError(pendingId);
        void vscode.window.showWarningMessage(
          "Open a text editor so Vocode can run actions against the active file.",
        );
        return;
      }

      const activeFile = editor.document.uri.fsPath;
      const text = evt.text;

      if (inFlightTranscripts === 0) {
        voiceStatus.setProcessing();
      }
      inFlightTranscripts++;

      transcriptStore.markProcessing(pendingId);

      void (async () => {
        try {
          const result = await client.transcript({
            text,
            activeFile,
            workspaceRoot: workspaceRootPath(),
          });
          await presentTranscriptResult(result, activeFile);
          transcriptStore.markHandled(pendingId);
        } catch (err) {
          transcriptStore.markError(pendingId);
          const message =
            err instanceof Error
              ? err.message
              : "unknown voice->transcript error";
          void vscode.window.showWarningMessage(
            `Vocode voice error: ${message}`,
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
      client,
      voiceStatus,
      voiceSession,
      microphone,
      voiceSidecar,
      transcriptStore,
    };
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Unknown daemon startup error";

    console.error(message);
    void vscode.window.showErrorMessage(
      `Failed to start Vocode daemon: ${message}`,
    );

    return {
      client: null,
      voiceStatus,
      voiceSession: new VoiceSessionController(),
      microphone,
      voiceSidecar: null,
      transcriptStore,
    };
  }
}

export function activate(context: vscode.ExtensionContext) {
  console.log("Vocode extension activated");

  const voiceStatus = new VoiceStatusIndicator();
  const transcriptStore = new TranscriptStore();
  const transcriptPanel = new TranscriptPanelViewProvider(
    context.extensionUri,
    transcriptStore,
  );

  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider(
      transcriptPanelViewType,
      transcriptPanel,
    ),
    transcriptPanel,
  );

  const services = createServices(context, voiceStatus, transcriptStore);

  context.subscriptions.push(voiceStatus, ...registerAllCommands(services), {
    dispose: () => {
      services.client?.dispose();
    },
  });
}

export function deactivate() {}
