import type { ChildProcessWithoutNullStreams } from "node:child_process";
import type {
  EditApplyParams,
  EditApplyResult,
  PingParams,
  PingResult,
} from "@vocode/protocol";
import { isEditApplyResult, isPingResult } from "@vocode/protocol";

import {
  isVoiceTranscriptResult,
  type VoiceTranscriptParams,
  type VoiceTranscriptResult,
} from "./requests";
import { RpcTransport } from "./rpc-transport";

export class DaemonClient {
  private readonly transport: RpcTransport;

  constructor(process: ChildProcessWithoutNullStreams) {
    this.transport = new RpcTransport(process);
  }

  public async sendRequest<T>(
    method: string,
    params: unknown,
    isResult?: (value: unknown) => value is T,
  ): Promise<T> {
    const result = await this.transport.request(method, params);

    if (isResult && !isResult(result)) {
      throw new Error(`Invalid ${method} response from daemon.`);
    }

    return result as T;
  }

  public async ping(params: PingParams = {}): Promise<PingResult> {
    return this.sendRequest<PingResult>("ping", params, isPingResult);
  }

  public async applyEdit(params: EditApplyParams): Promise<EditApplyResult> {
    return this.sendRequest<EditApplyResult>(
      "edit/apply",
      params,
      isEditApplyResult,
    );
  }

  public async voiceTranscript(
    params: VoiceTranscriptParams,
  ): Promise<VoiceTranscriptResult> {
    return this.sendRequest<VoiceTranscriptResult>(
      "voice.transcript",
      params,
      isVoiceTranscriptResult,
    );
  }

  public dispose(): void {
    this.transport.dispose();
  }
}
