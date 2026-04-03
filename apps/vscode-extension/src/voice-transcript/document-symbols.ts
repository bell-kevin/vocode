import type { HostGetDocumentSymbolsResult } from "@vocode/protocol";
import * as vscode from "vscode";

type WireSymbol = HostGetDocumentSymbolsResult["symbols"][number];

function rangeToWire(r: vscode.Range): {
  startLine: number;
  startChar: number;
  endLine: number;
  endChar: number;
} {
  return {
    startLine: r.start.line,
    startChar: r.start.character,
    endLine: r.end.line,
    endChar: r.end.character,
  };
}

function walkSymbols(syms: vscode.DocumentSymbol[], out: WireSymbol[]): void {
  for (const s of syms) {
    out.push({
      name: s.name,
      kind: vscode.SymbolKind[s.kind] ?? String(s.kind),
      range: rangeToWire(s.range),
      selectionRange: rangeToWire(s.selectionRange),
    });
    if (s.children?.length) {
      walkSymbols(s.children, out);
    }
  }
}

/** Flatten VS Code document symbols to the wire shape used by vocode-cored. */
export function flattenDocumentSymbols(
  syms: vscode.DocumentSymbol[] | undefined,
): WireSymbol[] {
  const out: WireSymbol[] = [];
  if (syms?.length) {
    walkSymbols(syms, out);
  }
  return out;
}
