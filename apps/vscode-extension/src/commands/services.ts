import type { DaemonClient } from "../daemon/client";
import type { VoiceStatusIndicator } from "../ui/status-bar";
import type { MicrophoneCapture } from "../voice/microphone";
import type { TranscriptStore } from "../voice/transcript-store";
import type { VoiceSidecarClient } from "../voice-sidecar/client";

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
  microphone: MicrophoneCapture;
  voiceSidecar: VoiceSidecarClient | null;
  transcriptStore: TranscriptStore;
}
