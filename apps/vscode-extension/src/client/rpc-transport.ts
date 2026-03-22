import type { ChildProcessWithoutNullStreams } from "node:child_process";
import { createInterface } from "node:readline";

interface JsonRpcRequest<TParams = unknown> {
  jsonrpc: "2.0";
  id: number;
  method: string;
  params: TParams;
}

interface JsonRpcSuccess<TResult = unknown> {
  jsonrpc: "2.0";
  id: number;
  result: TResult;
}

interface JsonRpcError {
  jsonrpc: "2.0";
  id: number | null;
  error: {
    code: number;
    message: string;
    data?: unknown;
  };
}

type JsonRpcResponse<TResult = unknown> =
  | JsonRpcSuccess<TResult>
  | JsonRpcError;

interface PendingRequest<TResult = unknown> {
  resolve: (value: TResult) => void;
  reject: (reason?: unknown) => void;
}

export class RpcTransport {
  private readonly process: ChildProcessWithoutNullStreams;
  private readonly pending = new Map<number, PendingRequest>();
  private nextId = 1;
  private disposed = false;

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
        const message = JSON.parse(trimmed) as JsonRpcResponse;
        this.handleMessage(message);
      } catch (error) {
        console.error("[rpc] failed to parse daemon stdout as JSON:", error);
        console.error("[rpc] raw line:", trimmed);
      }
    });

    this.process.on("error", (error) => {
      this.rejectAll(error);
    });

    this.process.on("exit", (code, signal) => {
      this.rejectAll(
        new Error(
          `Daemon exited before response. code=${code} signal=${signal}`,
        ),
      );
    });
  }

  public async request<TParams, TResult>(
    method: string,
    params: TParams,
  ): Promise<TResult> {
    if (this.disposed) {
      throw new Error("RPC transport is disposed.");
    }

    const id = this.nextId++;
    const payload: JsonRpcRequest<TParams> = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };

    const promise = new Promise<TResult>((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
    });

    this.process.stdin.write(`${JSON.stringify(payload)}\n`);

    return promise;
  }

  public dispose(): void {
    if (this.disposed) {
      return;
    }

    this.disposed = true;
    this.rejectAll(new Error("RPC transport disposed."));
  }

  private handleMessage(message: JsonRpcResponse): void {
    if (typeof message.id !== "number") {
      return;
    }

    const pending = this.pending.get(message.id);
    if (!pending) {
      console.warn(`[rpc] received response for unknown id=${message.id}`);
      return;
    }

    this.pending.delete(message.id);

    if ("error" in message) {
      pending.reject(
        new Error(`[rpc] ${message.error.code}: ${message.error.message}`),
      );
      return;
    }

    pending.resolve(message.result);
  }

  private rejectAll(error: unknown): void {
    for (const [, pending] of this.pending) {
      pending.reject(error);
    }

    this.pending.clear();
  }
}
