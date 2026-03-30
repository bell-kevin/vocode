import type { ChildProcessWithoutNullStreams } from "node:child_process";
import { createInterface } from "node:readline";

interface JsonRpcRequest {
  jsonrpc: "2.0";
  id: number;
  method: string;
  params: unknown;
}

interface JsonRpcSuccess {
  jsonrpc: "2.0";
  id: number;
  result: unknown;
}

interface JsonRpcSuccessWithNullId {
  jsonrpc: "2.0";
  id: number | null;
  result: unknown;
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

type JsonRpcResponse = JsonRpcSuccess | JsonRpcSuccessWithNullId | JsonRpcError;

interface JsonRpcRequestDispatch {
  handler: (params: unknown) => Promise<unknown> | unknown;
}

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}

export class RpcTransport {
  private readonly process: ChildProcessWithoutNullStreams;
  private readonly pending = new Map<number, PendingRequest>();
  private readonly requestHandlers = new Map<string, JsonRpcRequestDispatch>();
  private nextId = 1;
  private disposed = false;
  private writeChain: Promise<void> = Promise.resolve();

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
        const message = JSON.parse(trimmed) as unknown;
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

  public registerRequestHandler(
    method: string,
    handler: (params: unknown) => Promise<unknown> | unknown,
  ): void {
    this.requestHandlers.set(method, { handler });
  }

  public request(method: string, params: unknown): Promise<unknown> {
    if (this.disposed) {
      return Promise.reject(new Error("RPC transport is disposed."));
    }

    const id = this.nextId++;
    const payload: JsonRpcRequest = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };

    const promise = new Promise<unknown>((resolve, reject) => {
      this.pending.set(id, {
        resolve,
        reject,
      });
    });

    try {
      this.enqueueWrite(`${JSON.stringify(payload)}\n`);
    } catch (error) {
      this.pending.delete(id);
      return Promise.reject(error);
    }

    return promise;
  }

  public dispose(): void {
    if (this.disposed) {
      return;
    }

    this.disposed = true;
    this.rejectAll(new Error("RPC transport disposed."));
  }

  private handleMessage(message: unknown): void {
    if (!message || typeof message !== "object") {
      return;
    }
    const m = message as Record<string, unknown>;

    const id = m.id;
    if (typeof id !== "number") {
      // For our duplex protocol, responses always have numeric ids.
      return;
    }

    // Daemon -> extension: JSON-RPC request
    if (typeof m.method === "string") {
      const method = m.method;
      const params = m.params;
      const handlerDispatch = this.requestHandlers.get(method);

      if (!handlerDispatch) {
        this.sendErrorResponse(id, -32601, "Method not found");
        return;
      }

      void Promise.resolve()
        .then(() => handlerDispatch.handler(params))
        .then((result) => this.sendSuccessResponse(id, result))
        .catch((err) => {
          const msg = err instanceof Error ? err.message : String(err);
          this.sendErrorResponse(id, -32000, msg);
        });
      return;
    }

    // Extension -> daemon: JSON-RPC response
    const pending = this.pending.get(id);
    if (!pending) {
      console.warn(`[rpc] received response for unknown id=${id}`);
      return;
    }

    this.pending.delete(id);

    if ("error" in m && m.error && typeof m.error === "object") {
      const errObj = m.error as Record<string, unknown>;
      const code = typeof errObj.code === "number" ? errObj.code : -32000;
      const msg =
        typeof errObj.message === "string"
          ? errObj.message
          : "Unknown RPC error";
      pending.reject(new Error(`[rpc] ${code}: ${msg}`));
      return;
    }

    if (!("result" in m)) {
      // Invalid message shape; ignore.
      return;
    }

    const result = (m as Record<string, unknown>).result;
    pending.resolve(result);
  }

  private enqueueWrite(raw: string): void {
    // Serialize all writes onto stdin so JSON lines never interleave.
    this.writeChain = this.writeChain.then(() => {
      this.process.stdin.write(raw);
    });
  }

  private sendSuccessResponse(id: number, result: unknown): void {
    const payload = { jsonrpc: "2.0", id, result };
    this.enqueueWrite(`${JSON.stringify(payload)}\n`);
  }

  private sendErrorResponse(
    id: number,
    code: number,
    message: string,
  ): void {
    const payload = {
      jsonrpc: "2.0",
      id,
      error: {
        code,
        message,
      },
    };
    this.enqueueWrite(`${JSON.stringify(payload)}\n`);
  }

  private rejectAll(error: unknown): void {
    for (const [, pending] of this.pending) {
      pending.reject(error);
    }

    this.pending.clear();
  }
}
