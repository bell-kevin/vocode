import type { DaemonClient } from "../client/daemon-client";
import type { VoiceStatusIndicator } from "../ui/status-bar";

export interface ExtensionServices {
  client: DaemonClient | null;
  voiceStatus: VoiceStatusIndicator;
}
