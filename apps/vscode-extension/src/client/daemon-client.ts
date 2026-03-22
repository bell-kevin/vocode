import type { ChildProcessWithoutNullStreams } from "node:child_process";
import type {
  EditApplyParams,
  EditApplyResult,
  PingParams,
  PingResult,
} from "@vocode/protocol/ts";

import { RpcTransport } from "./rpc-transport";

export class DaemonClient {
  private readonly transport: RpcTransport;

  constructor(process: ChildProcessWithoutNullStreams) {
    this.transport = new RpcTransport(process);
  }

  public ping(params: PingParams = {}): Promise<PingResult> {
    return this.transport.request<PingParams, PingResult>("ping", params);
  }

  public applyEdit(params: EditApplyParams): Promise<EditApplyResult> {
    return this.transport.request<EditApplyParams, EditApplyResult>(
      "edit/apply",
      params,
    );
  }

  public dispose(): void {
    this.transport.dispose();
  }
}
