import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import { PassThrough } from "node:stream";
import { test } from "node:test";

import { RpcTransport } from "./rpc-transport";

function makeFakeProcess() {
  class FakeProcess extends EventEmitter {
    public stdout = new PassThrough();
    public stdin = new PassThrough();
  }
  return new FakeProcess();
}

async function waitFor(
  fn: () => boolean,
  timeoutMs = 1000,
): Promise<void> {
  const start = Date.now();
  // eslint-disable-next-line no-constant-condition
  while (true) {
    if (fn()) return;
    if (Date.now() - start > timeoutMs) {
      throw new Error("waitFor: timeout");
    }
    await new Promise((r) => setTimeout(r, 10));
  }
}

test("rpc-transport: handles incoming requests and returns responses", async () => {
  const proc = makeFakeProcess() as any;
  const transport = new RpcTransport(proc);

  transport.registerRequestHandler("host.applyDirectives", async () => {
    return { items: [{ status: "ok" }] };
  });

  let output = "";
  proc.stdin.on("data", (chunk: Buffer) => {
    output += chunk.toString("utf8");
  });

  proc.stdout.write(
    JSON.stringify({
      jsonrpc: "2.0",
      id: 123,
      method: "host.applyDirectives",
      params: {
        applyBatchId: "b1",
        activeFile: "test.js",
        directives: [],
      },
    }) + "\n",
  );

  await waitFor(() => output.includes("\n"), 1000);

  const line = output.trim().split("\n")[0];
  const parsed = JSON.parse(line) as any;

  assert.equal(parsed.jsonrpc, "2.0");
  assert.equal(parsed.id, 123);
  assert.deepEqual(parsed.result, { items: [{ status: "ok" }] });
});

test("rpc-transport: responds with method-not-found for unknown requests", async () => {
  const proc = makeFakeProcess() as any;
  const transport = new RpcTransport(proc);

  let output = "";
  proc.stdin.on("data", (chunk: Buffer) => {
    output += chunk.toString("utf8");
  });

  proc.stdout.write(
    JSON.stringify({
      jsonrpc: "2.0",
      id: 7,
      method: "unknown.method",
      params: {},
    }) + "\n",
  );

  await waitFor(() => output.includes("\n"), 1000);

  const line = output.trim().split("\n")[0];
  const parsed = JSON.parse(line) as any;

  assert.equal(parsed.jsonrpc, "2.0");
  assert.equal(parsed.id, 7);
  assert.ok(parsed.error);
  assert.equal(parsed.error.code, -32601);
  assert.equal(parsed.error.message, "Method not found");
});

