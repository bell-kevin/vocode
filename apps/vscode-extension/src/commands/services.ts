import { randomUUID } from "node:crypto";

import type { DaemonClient } from "../daemon/client";
import type { MainPanelStore } from "../ui/main-panel-store";
import type { VoiceStatusIndicator } from "../ui/status-bar";
import type { VoiceSidecarClient } from "../voice/client";

export class VoiceSessionController {
  private activeSessionId: number | null = null;
  private nextSessionId = 1;
  /** Daemon retained gathered context (excerpts/symbols/notes) is keyed by this id between voice.transcript RPCs. */
  private contextSessionUUID: string | null = null;

  start(): number {
    this.stop();

    this.contextSessionUUID = randomUUID();
    const id = this.nextSessionId++;
    this.activeSessionId = id;
    return id;
  }

  stop(): void {
    this.activeSessionId = null;
    this.contextSessionUUID = null;
  }

  /** Opaque id: same value on each transcript while listening so gathered context accumulates in the daemon. */
  contextSessionId(): string | undefined {
    return this.contextSessionUUID ?? undefined;
  }

  isActive(sessionId: number): boolean {
    return this.activeSessionId === sessionId;
  }

  isRunning(): boolean {
    return this.activeSessionId !== null;
  }
}

export interface ExtensionServices {
  client: DaemonClient | null;
  voiceStatus: VoiceStatusIndicator;
  voiceSession: VoiceSessionController;
  voiceSidecar: VoiceSidecarClient | null;
  mainPanelStore: MainPanelStore;
}
