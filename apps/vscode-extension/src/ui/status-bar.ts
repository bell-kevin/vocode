import * as vscode from "vscode";

export type VoiceUiState = "idle" | "listening" | "processing";

const STATE_META: Record<
  VoiceUiState,
  { icon: string; label: string; tooltip: string }
> = {
  idle: {
    icon: "$(debug-pause)",
    label: "Idle",
    tooltip: "Vocode is idle",
  },
  listening: {
    icon: "$(unmute)",
    label: "Listening",
    tooltip: "Vocode is listening for your request",
  },
  processing: {
    icon: "$(sync~spin)",
    label: "Processing",
    tooltip: "Vocode is processing your request",
  },
};

export class VoiceStatusIndicator implements vscode.Disposable {
  private readonly item: vscode.StatusBarItem;
  private state: VoiceUiState = "idle";

  constructor() {
    this.item = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100,
    );
    this.item.name = "Vocode Status";
    this.item.command = "vocode.startVoice";
    this.render();
    this.item.show();
  }

  setState(next: VoiceUiState): void {
    if (this.state === next) {
      return;
    }

    this.state = next;
    this.render();
  }

  setIdle(): void {
    this.setState("idle");
  }

  setListening(): void {
    this.setState("listening");
  }

  setProcessing(): void {
    this.setState("processing");
  }

  dispose(): void {
    this.item.dispose();
  }

  private render(): void {
    const meta = STATE_META[this.state];
    this.item.text = `${meta.icon} Vocode: ${meta.label}`;
    this.item.command =
      this.state === "idle" ? "vocode.startVoice" : "vocode.stopVoice";
    this.item.tooltip =
      this.state === "idle"
        ? `${meta.tooltip}. Click to start voice.`
        : `${meta.tooltip}. Click to stop voice.`;
  }
}
