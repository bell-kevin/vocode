import * as vscode from "vscode";

import { registerAllCommands } from "./commands";
import {
  type ExtensionServices,
  VoiceSessionController,
} from "./commands/services";
import { DaemonClient } from "./daemon/client";
import { spawnDaemon } from "./daemon/spawn";
import { applyTranscriptResult } from "./transcript/apply-result";
import { VoiceStatusIndicator } from "./ui/status-bar";
import {
  TranscriptPanelViewProvider,
  transcriptPanelViewType,
} from "./ui/transcript-panel";
import { TranscriptStore } from "./ui/transcript-store";
import { VoiceSidecarClient } from "./voice/client";
import { spawnVoiceSidecar } from "./voice/spawn";

function workspaceRootPath(): string | undefined {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

function createServices(
  context: vscode.ExtensionContext,
  voiceStatus: VoiceStatusIndicator,
  transcriptStore: TranscriptStore,
): ExtensionServices {
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
        const message =
          "Open a text editor so Vocode can run actions against the active file.";
        transcriptStore.markError(pendingId, message);
        void vscode.window.showWarningMessage(message);
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
          await applyTranscriptResult(result, activeFile);
          transcriptStore.markHandled(pendingId, {
            summary:
              result.accepted && result.summary
                ? result.summary.trim() || undefined
                : undefined,
          });
        } catch (err) {
          const message =
            err instanceof Error
              ? err.message
              : "unknown voice->transcript error";
          transcriptStore.markError(pendingId, message);
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
