import { PassThrough, type Readable } from "node:stream";
import * as vscode from "vscode";

export interface MicrophoneAudioChunk {
  readonly data: Uint8Array;
  readonly mimeType: string;
  readonly timestamp: number;
}

type MediaRecorderState = "inactive" | "recording" | "paused";

interface BlobLike {
  readonly size: number;
  arrayBuffer(): Promise<ArrayBuffer>;
}

interface MediaRecorderEventLike {
  readonly data?: BlobLike;
}

interface MediaStreamTrackLike {
  stop(): void;
}

interface MediaStreamLike {
  getTracks(): MediaStreamTrackLike[];
}

interface MediaRecorderLike {
  readonly state: MediaRecorderState;
  readonly mimeType?: string;
  ondataavailable: ((event: MediaRecorderEventLike) => void) | null;
  onerror: ((event: { error?: unknown }) => void) | null;
  start(timeslice?: number): void;
  stop(): void;
}

interface MediaDevicesLike {
  getUserMedia(constraints: unknown): Promise<MediaStreamLike>;
}

interface NavigatorLike {
  readonly mediaDevices?: MediaDevicesLike;
}

type MediaRecorderConstructor = new (
  stream: MediaStreamLike,
  options?: { mimeType?: string },
) => MediaRecorderLike;

export class MicrophoneCapture implements vscode.Disposable {
  private readonly chunkEmitter =
    new vscode.EventEmitter<MicrophoneAudioChunk>();
  private readonly stream = new PassThrough();

  private mediaStream: MediaStreamLike | null = null;
  private recorder: MediaRecorderLike | null = null;

  public readonly onAudioChunk = this.chunkEmitter.event;

  public get audioStream(): Readable {
    return this.stream;
  }

  public isCapturing(): boolean {
    return this.recorder?.state === "recording";
  }

  public async start(): Promise<void> {
    if (this.isCapturing()) {
      return;
    }

    const navigatorLike = (globalThis as { navigator?: NavigatorLike })
      .navigator;
    const mediaDevices = navigatorLike?.mediaDevices;
    const MediaRecorderClass = (
      globalThis as { MediaRecorder?: MediaRecorderConstructor }
    ).MediaRecorder;

    if (!mediaDevices?.getUserMedia || !MediaRecorderClass) {
      throw new Error(
        "Microphone capture is unavailable in this VS Code host.",
      );
    }

    const mediaStream = await mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
      },
    });

    try {
      const recorder = new MediaRecorderClass(mediaStream);

      recorder.ondataavailable = (event) => {
        void this.handleData(
          event,
          recorder.mimeType ?? "application/octet-stream",
        );
      };

      recorder.onerror = (event) => {
        const details =
          event.error instanceof Error ? event.error.message : "unknown error";
        void vscode.window.showWarningMessage(
          `Vocode microphone capture error: ${details}`,
        );
      };

      recorder.start(250);

      this.mediaStream = mediaStream;
      this.recorder = recorder;
    } catch (error) {
      for (const track of mediaStream.getTracks()) {
        try {
          track.stop();
        } catch {
          // Ignore errors while stopping individual tracks; preserve original error.
        }
      }
      throw error;
    }
  }

  public stop(): void {
    if (this.recorder && this.recorder.state !== "inactive") {
      this.recorder.stop();
    }

    this.recorder = null;

    if (this.mediaStream) {
      for (const track of this.mediaStream.getTracks()) {
        track.stop();
      }
    }

    this.mediaStream = null;
  }

  public dispose(): void {
    this.stop();
    this.chunkEmitter.dispose();
    this.stream.end();
  }

  private async handleData(
    event: MediaRecorderEventLike,
    mimeType: string,
  ): Promise<void> {
    if (!event.data || event.data.size <= 0) {
      return;
    }

    const data = new Uint8Array(await event.data.arrayBuffer());
    this.stream.write(Buffer.from(data));

    this.chunkEmitter.fire({
      data,
      mimeType,
      timestamp: Date.now(),
    });
  }
}
