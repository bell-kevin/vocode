import type {
  HostGetDocumentSymbolsParams,
  HostGetDocumentSymbolsResult,
  HostReadFileParams,
  HostReadFileResult,
  HostWorkspaceSymbolSearchParams,
  HostWorkspaceSymbolSearchResult,
} from "@vocode/protocol";
import * as vscode from "vscode";

import { flattenDocumentSymbols } from "./document-symbols";

export async function handleHostReadFile(
  params: HostReadFileParams,
): Promise<HostReadFileResult> {
  const uri = vscode.Uri.file(params.path);
  const bytes = await vscode.workspace.fs.readFile(uri);
  return { text: new TextDecoder("utf-8").decode(bytes) };
}

export async function handleHostGetDocumentSymbols(
  params: HostGetDocumentSymbolsParams,
): Promise<HostGetDocumentSymbolsResult> {
  const uri = vscode.Uri.file(params.path);
  const raw = await vscode.commands.executeCommand<
    vscode.DocumentSymbol[] | undefined
  >("vscode.executeDocumentSymbolProvider", uri);
  return { symbols: flattenDocumentSymbols(raw) };
}

function symbolMatchesQuery(
  query: string,
  name: string,
  containerName: string,
): boolean {
  const q = query.trim().toLowerCase();
  if (!q) {
    return false;
  }
  const hay = `${name} ${containerName}`.toLowerCase().replace(/\s+/g, " ");
  const parts = q.split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return false;
  }
  return parts.every((p) => hay.includes(p));
}

export async function handleHostWorkspaceSymbolSearch(
  params: HostWorkspaceSymbolSearchParams,
): Promise<HostWorkspaceSymbolSearchResult> {
  const raw = await vscode.commands.executeCommand<
    vscode.SymbolInformation[] | undefined
  >("vscode.executeWorkspaceSymbolProvider", params.query);
  const hits: HostWorkspaceSymbolSearchResult["hits"] = [];
  const maxHits = 20;
  for (const s of raw ?? []) {
    const name = s.name ?? "";
    const container = s.containerName ?? "";
    if (!symbolMatchesQuery(params.query, name, container)) {
      continue;
    }
    const path = s.location.uri.fsPath;
    const r = s.location.range;
    const start = r.start;
    const end = r.end;
    const matchLength =
      start.line === end.line
        ? Math.max(1, end.character - start.character)
        : 1;
    hits.push({
      path,
      line: start.line,
      character: start.character,
      preview: name,
      matchLength,
    });
    if (hits.length >= maxHits) {
      break;
    }
  }
  return { hits };
}
