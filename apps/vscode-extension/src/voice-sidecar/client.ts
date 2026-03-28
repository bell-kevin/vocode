import type { ChildProcessWithoutNullStreams } from "node:child_process";
import { createInterface } from "node:readline";
import * as vscode from "vscode";

interface VoiceSidecarEvent {
  type: string;
  state?: string;
  message?: string;
  version?: string;
  /** Present on `ready` when built from current sources (stale binary omits this). */
  features?: {
    transcript_committed_field?: boolean;
    audio_meter?: boolean;
  };
  text?: string;
  committed?: boolean;
  speaking?: boolean;
  rms?: number;
}

const staleSidecarMessage =
  "Vocode voice sidecar is outdated. From the repo root run: pnpm --filter @vocode/voice build, then reload the window (or restart the Extension Development Host).";

export interface VoiceSidecarTranscriptEvent {
  type: "transcript";
  text: string;
  timestamp?: number;
  committed?: boolean;
}

export interface VoiceSidecarAudioMeterEvent {
  type: "audio_meter";
  speaking: boolean;
  /** Normalized 0–1 RMS from sidecar. */
  rms: number;
}

export class VoiceSidecarClient {
  private readonly process: ChildProcessWithoutNullStreams;
  private disposed = false;
  private warnedStaleSidecar = false;
  private transcriptHandler?: (evt: VoiceSidecarTranscriptEvent) => void;
  private stateHandler?: (evt: { type: "state"; state: string }) => void;
  private errorHandler?: (evt: { type: "error"; message: string }) => void;
  private audioMeterHandler?: (evt: VoiceSidecarAudioMeterEvent) => void;

  constructor(process: ChildProcessWithoutNullStreams) {
    this.process = process;

    const rl = createInterface({
      input: this.process.stdout,
      crlfDelay: Number.POSITIVE_INFINITY,
    });

    rl.on("line", (line) => {
      const trimmed = line.trim();
      if (!trimmed) {
        return;
      }
      try {
        const evt = JSON.parse(trimmed) as VoiceSidecarEvent;
        this.dispatchSidecarEvent(evt);
      } catch (err) {
        console.error("[vocode-voiced] failed to parse stdout as JSON:", err);
        console.error("[vocode-voiced] raw line:", trimmed);
      }
    });
  }

  private dispatchSidecarEvent(evt: VoiceSidecarEvent): void {
    if (evt.type === "ready") {
      const f = evt.features;
      const ok =
        f?.transcript_committed_field === true && f?.audio_meter === true;
      console.log(
        `[vocode-voiced] ready version=${evt.version ?? "?"} features=${ok ? "ok" : "STALE_OR_OLD"}`,
      );
      if (!ok && !this.warnedStaleSidecar) {
        this.warnedStaleSidecar = true;
        console.warn(`[vocode-voiced] ${staleSidecarMessage}`);
        void vscode.window.showWarningMessage(staleSidecarMessage);
      }
      return;
    }
    if (evt.type === "state") {
      console.log(`[vocode-voiced] state=${evt.state ?? "?"}`);
      if (typeof evt.state === "string" && evt.state) {
        this.stateHandler?.({ type: "state", state: evt.state });
      }
      return;
    }
    if (evt.type === "error") {
      console.warn(`[vocode-voiced] error: ${evt.message ?? "unknown"}`);
      const message = typeof evt.message === "string" ? evt.message : "unknown";
      this.errorHandler?.({ type: "error", message });
      return;
    }
    if (evt.type === "audio_meter") {
      const speaking = evt.speaking === true;
      const raw = evt.rms;
      const rms =
        typeof raw === "number" && Number.isFinite(raw)
          ? Math.min(1, Math.max(0, raw))
          : 0;
      this.audioMeterHandler?.({
        type: "audio_meter",
        speaking,
        rms,
      });
      return;
    }
    if (evt.type === "transcript") {
      const text = typeof evt.text === "string" ? evt.text : "";
      const committed = this.getCommitted(evt);
      if (typeof evt.committed !== "boolean" && !this.warnedStaleSidecar) {
        this.warnedStaleSidecar = true;
        console.warn(
          "[vocode-voiced] transcript line missing boolean `committed` — sidecar binary is not current.",
        );
        console.warn(`[vocode-voiced] ${staleSidecarMessage}`);
        void vscode.window.showWarningMessage(staleSidecarMessage);
      }
      if (!text) return;
      this.transcriptHandler?.({
        type: "transcript",
        text,
        committed,
      });
    }
  }

  public start(): void {
    this.send({ type: "start" });
  }

  public stop(): void {
    this.send({ type: "stop" });
  }

  public shutdown(): void {
    this.send({ type: "shutdown" });
  }

  public dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    this.shutdown();
  }

  public onTranscript(handler: (evt: VoiceSidecarTranscriptEvent) => void) {
    this.transcriptHandler = handler;
  }

  public onState(handler: (evt: { type: "state"; state: string }) => void) {
    this.stateHandler = handler;
  }

  public onError(handler: (evt: { type: "error"; message: string }) => void) {
    this.errorHandler = handler;
  }

  public onAudioMeter(handler: (evt: VoiceSidecarAudioMeterEvent) => void) {
    this.audioMeterHandler = handler;
  }

  private send(msg: { type: string }): void {
    if (this.disposed) return;
    try {
      this.process.stdin.write(`${JSON.stringify(msg)}\n`);
    } catch (err) {
      console.error("[vocode-voiced] failed to write stdin:", err);
    }
  }

  private getCommitted(evt: VoiceSidecarEvent): boolean | undefined {
    return typeof evt.committed === "boolean" ? evt.committed : undefined;
  }
}
