import * as vscode from "vscode";

import { registerAllCommands } from "./commands";
import { presentTranscriptResult } from "./commands/send-transcript/present-result";
import {
  type ExtensionServices,
  VoiceSessionController,
} from "./commands/services";
import { DaemonClient } from "./daemon/client";
import { spawnDaemon } from "./daemon/spawn";
import { TranscriptSidebarProvider } from "./ui/sidebar-provider";
import { VoiceStatusIndicator } from "./ui/status-bar";
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

    voiceSidecar.onError((evt) => {
      // Ensure user sees sidecar failures even when no transcript ever arrives.
      if (!voiceSession.isRunning()) return;
      void vscode.window.showWarningMessage(
        `Vocode voice sidecar error: ${evt.message}`,
      );
      voiceSession.stop();
      voiceStatus.setIdle();
    });

    voiceSidecar.onTranscript((evt) => {
      const kind = evt.committed === true ? "final" : "partial";
      transcriptStore.add(evt.text, kind);

      if (!voiceSession.isRunning()) {
        return;
      }
      // Only final/committed transcript hypotheses should be forwarded to the core daemon.
      if (evt.committed !== true) return;

      const editor = vscode.window.activeTextEditor;
      if (!editor) {
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

      void (async () => {
        try {
          const result = await client.transcript({
            text,
            activeFile,
            workspaceRoot: workspaceRootPath(),
          });
          await presentTranscriptResult(result, activeFile);
        } catch (err) {
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
    };
  }
}

export function activate(context: vscode.ExtensionContext) {
  console.log("Vocode extension activated");

  const voiceStatus = new VoiceStatusIndicator();
  const transcriptStore = new TranscriptStore();
  const transcriptSidebarProvider = new TranscriptSidebarProvider(
    transcriptStore,
  );

  context.subscriptions.push(
    vscode.window.registerTreeDataProvider(
      "vocode.liveTranscriptView",
      transcriptSidebarProvider,
    ),
  );

  const services = createServices(context, voiceStatus, transcriptStore);

  context.subscriptions.push(
    voiceStatus,
    transcriptSidebarProvider,
    ...registerAllCommands(services),
    {
      dispose: () => {
        services.client?.dispose();
      },
    },
  );
}

export function deactivate() {}
