import type { ChildProcessWithoutNullStreams } from "node:child_process";
import type { HostApplyParams, HostApplyResult } from "@vocode/protocol";
import * as vscode from "vscode";

import { registerAllCommands } from "./commands";
import {
  type ExtensionServices,
  VoiceSessionController,
} from "./commands/services";
import {
  ANTHROPIC_API_KEY_SECRET,
  ELEVENLABS_API_KEY_SECRET,
  OPENAI_API_KEY_SECRET,
} from "./config/spawn-env";
import { DaemonClient } from "./daemon/client";
import { spawnDaemon } from "./daemon/spawn";
import {
  MainPanelViewProvider,
  mainPanelViewType,
} from "./ui/panel/main-panel";
import { MainPanelStore } from "./ui/panel/main-panel-store";
import { VoiceStatusIndicator } from "./ui/status-bar";
import { VoiceSidecarClient } from "./voice/client";
import { spawnVoiceSidecar } from "./voice/spawn";
import { applyDirectives } from "./voice-transcript/apply-directives";
import { attachTranscriptPipeline } from "./voice-transcript/pipeline";

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

        const outcomes = await applyDirectives(
          params.directives,
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
    services.disposeTranscriptPipeline =
      attachTranscriptPipeline(services).dispose;
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

  const mainPanel = new MainPanelViewProvider(
    context.extensionUri,
    mainPanelStore,
    context,
    services,
  );

  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider(mainPanelViewType, mainPanel),
    mainPanel,
  );

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
      services.disposeTranscriptPipeline =
        attachTranscriptPipeline(services).dispose;
    } finally {
      voiceRestartInFlight = false;
    }
  };

  await wireVocodeBackend(context, services, daemonProcRef, voiceProcRef);

  // Some daemon settings are passed via spawn env (provider/model/base URL) and therefore
  // require a restart to take effect. Keep the running backend consistent with settings.
  let daemonConfigRestartTimeout: NodeJS.Timeout | undefined;
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      const affectsDaemonSpawnEnv =
        e.affectsConfiguration("vocode.daemonAgentProvider") ||
        e.affectsConfiguration("vocode.daemonOpenaiModel") ||
        e.affectsConfiguration("vocode.daemonOpenaiBaseUrl") ||
        e.affectsConfiguration("vocode.daemonAnthropicModel") ||
        e.affectsConfiguration("vocode.daemonAnthropicBaseUrl") ||
        e.affectsConfiguration("vocode.daemonVoiceLogTranscript") ||
        e.affectsConfiguration("vocode.daemonVoiceTranscriptQueueSize") ||
        e.affectsConfiguration("vocode.daemonVoiceTranscriptCoalesceMs") ||
        e.affectsConfiguration("vocode.daemonVoiceTranscriptMaxMergeJobs") ||
        e.affectsConfiguration("vocode.daemonVoiceTranscriptMaxMergeChars");

      if (!affectsDaemonSpawnEnv) {
        return;
      }

      if (daemonConfigRestartTimeout) {
        clearTimeout(daemonConfigRestartTimeout);
      }
      daemonConfigRestartTimeout = setTimeout(() => {
        void services.restartVocode?.();
      }, 250);
    }),
    {
      dispose: () => {
        if (daemonConfigRestartTimeout) {
          clearTimeout(daemonConfigRestartTimeout);
        }
      },
    },
  );

  // Secrets apply immediately: restart the backend when the ElevenLabs key changes
  // so the running sidecar/daemon always see the latest configuration.
  context.subscriptions.push(
    context.secrets.onDidChange((e) => {
      if (e.key === ELEVENLABS_API_KEY_SECRET) {
        // Only the voice sidecar consumes ELEVENLABS_API_KEY.
        void services.restartVoiceSidecar?.();
        return;
      }
      if (
        e.key === OPENAI_API_KEY_SECRET ||
        e.key === ANTHROPIC_API_KEY_SECRET
      ) {
        // Daemon consumes cloud model keys from env; restart to pick up latest values.
        void services.restartVocode?.();
      }
    }),
  );

  context.subscriptions.push(voiceStatus, ...registerAllCommands(services), {
    dispose: () => {
      services.client?.dispose();
    },
  });
}

export function deactivate() {}
