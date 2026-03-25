import type { DaemonClient } from "../client/daemon-client";
import type { VoiceStatusIndicator } from "../ui/status-bar";

export class VoiceSessionController {
  private activeSessionId: number | null = null;
  private nextSessionId = 1;

  start(): number {
    this.stop();

    const id = this.nextSessionId++;
    this.activeSessionId = id;
    return id;
  }

  stop(): void {
    this.activeSessionId = null;
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
}
