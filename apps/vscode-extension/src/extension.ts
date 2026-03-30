import type { ChildProcessWithoutNullStreams } from "node:child_process";
import * as vscode from "vscode";

import { registerAllCommands } from "./commands";
import {
  type ExtensionServices,
  VoiceSessionController,
} from "./commands/services";
import { DaemonClient } from "./daemon/client";
import { spawnDaemon } from "./daemon/spawn";
import { attachTranscriptPipeline } from "./extension/transcript-pipeline";
import { MainPanelViewProvider, mainPanelViewType } from "./ui/main-panel";
import { MainPanelStore } from "./ui/main-panel-store";
import { VoiceStatusIndicator } from "./ui/status-bar";
import { VoiceSidecarClient } from "./voice/client";
import { spawnVoiceSidecar } from "./voice/spawn";
import { applyTranscriptResult } from "./transcript/apply-result";
import { ELEVENLABS_API_KEY_SECRET } from "./config/spawn-env";
import type {
  HostApplyParams,
  HostApplyResult,
  VoiceTranscriptResult,
} from "@vocode/protocol";

function safeKillProcess(proc: ChildProcessWithoutNullStreams | null): void {
  if (!proc || proc.killed) {
    return;
  }
  try {
    proc.kill();
  } catch {
    // Process may already be gone.
  }
}

async function wireVocodeBackend(
  context: vscode.ExtensionContext,
  services: ExtensionServices,
  daemonProcRef: { current: ChildProcessWithoutNullStreams | null },
  voiceProcRef: { current: ChildProcessWithoutNullStreams | null },
): Promise<void> {
  try {
    const daemon = await spawnDaemon(context);
    console.log(`Vocode daemon started from ${daemon.binaryPath}`);

    const voice = await spawnVoiceSidecar(context);
    console.log(`Vocode voice sidecar started from ${voice.binaryPath}`);

    services.client = new DaemonClient(daemon.process);
    services.voiceSidecar = new VoiceSidecarClient(voice.process);

    services.client.registerRequestHandler(
      "host.applyDirectives",
      async (unknownParams): Promise<HostApplyResult> => {
        const params = unknownParams as HostApplyParams;
        if (
          !params ||
          typeof params.applyBatchId !== "string" ||
          typeof params.activeFile !== "string" ||
          !Array.isArray(params.directives)
        ) {
          throw new Error(
            "host.applyDirectives: invalid params (expected applyBatchId, activeFile, directives)",
          );
        }

        const voiceResult: VoiceTranscriptResult = {
          success: true,
          directives: params.directives,
        };
        const outcomes = await applyTranscriptResult(
          voiceResult,
          params.activeFile,
        );

        return {
          items: outcomes.map((o) => ({
            status: o.status,
            ...(o.message !== undefined && o.message !== ""
              ? { message: o.message }
              : {}),
          })),
        };
      },
    );

    daemonProcRef.current = daemon.process;
    voiceProcRef.current = voice.process;

    services.disposeTranscriptPipeline?.();
    services.disposeTranscriptPipeline = attachTranscriptPipeline(services).dispose;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Unknown daemon startup error";

    console.error(message);
    void vscode.window.showErrorMessage(
      `Failed to start Vocode daemon: ${message}`,
    );

    services.client = null;
    services.voiceSidecar = null;
    daemonProcRef.current = null;
    voiceProcRef.current = null;
  }
}

export async function activate(context: vscode.ExtensionContext) {
  console.log("Vocode extension activated");

  const voiceStatus = new VoiceStatusIndicator();
  const mainPanelStore = new MainPanelStore();
  const mainPanel = new MainPanelViewProvider(
    context.extensionUri,
    mainPanelStore,
    context,
  );

  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider(mainPanelViewType, mainPanel),
    mainPanel,
  );

  const voiceSession = new VoiceSessionController();
  const daemonProcRef: {
    current: ChildProcessWithoutNullStreams | null;
  } = { current: null };
  const voiceProcRef: {
    current: ChildProcessWithoutNullStreams | null;
  } = { current: null };

  const services: ExtensionServices = {
    client: null,
    voiceStatus,
    voiceSession,
    voiceSidecar: null,
    mainPanelStore,
  };

  let restartInFlight = false;
  services.restartVocode = async () => {
    if (restartInFlight) {
      return;
    }
    restartInFlight = true;
    try {
      voiceSession.stop();
      services.voiceSidecar?.stop();
      mainPanelStore.setVoiceListening(false);
      voiceStatus.setIdle();
      services.client?.dispose();
      services.voiceSidecar?.dispose();
      safeKillProcess(daemonProcRef.current);
      safeKillProcess(voiceProcRef.current);
      daemonProcRef.current = null;
      voiceProcRef.current = null;
      services.client = null;
      services.voiceSidecar = null;
      services.disposeTranscriptPipeline?.();
      services.disposeTranscriptPipeline = undefined;

      await wireVocodeBackend(context, services, daemonProcRef, voiceProcRef);

      if (services.client) {
        void vscode.window.showInformationMessage(
          "Vocode daemon and voice sidecar restarted.",
        );
      }
    } finally {
      restartInFlight = false;
    }
  };

  let voiceRestartInFlight = false;
  services.restartVoiceSidecar = async () => {
    if (voiceRestartInFlight) {
      return;
    }
    voiceRestartInFlight = true;
    try {
      // If voice is active, stop listening before swapping the process.
      voiceSession.stop();
      services.voiceSidecar?.stop();
      mainPanelStore.setVoiceListening(false);
      voiceStatus.setIdle();

      services.voiceSidecar?.dispose();
      safeKillProcess(voiceProcRef.current);
      voiceProcRef.current = null;
      services.voiceSidecar = null;

      const voice = await spawnVoiceSidecar(context);
      console.log(`Vocode voice sidecar restarted from ${voice.binaryPath}`);

      voiceProcRef.current = voice.process;
      services.voiceSidecar = new VoiceSidecarClient(voice.process);

      services.disposeTranscriptPipeline?.();
      services.disposeTranscriptPipeline = attachTranscriptPipeline(services).dispose;
    } finally {
      voiceRestartInFlight = false;
    }
  };

  await wireVocodeBackend(context, services, daemonProcRef, voiceProcRef);

  // Secrets apply immediately: restart the backend when the ElevenLabs key changes
  // so the running sidecar/daemon always see the latest configuration.
  context.subscriptions.push(
    context.secrets.onDidChange((e) => {
      if (e.key !== ELEVENLABS_API_KEY_SECRET) {
        return;
      }
      // Only the voice sidecar consumes ELEVENLABS_API_KEY.
      void services.restartVoiceSidecar?.();
    }),
  );

  context.subscriptions.push(voiceStatus, ...registerAllCommands(services), {
    dispose: () => {
      services.client?.dispose();
    },
  });
}

export function deactivate() {}
