import * as vscode from "vscode";

import type {
  TranscriptEntry,
  TranscriptStore,
} from "../voice/transcript-store";

type TranscriptTreeNode =
  | {
      readonly kind: "header";
      readonly label: string;
      readonly description: string;
    }
  | { readonly kind: "entry"; readonly entry: TranscriptEntry };

export class TranscriptSidebarProvider
  implements vscode.TreeDataProvider<TranscriptTreeNode>, vscode.Disposable
{
  private readonly onDidChangeTreeDataEmitter = new vscode.EventEmitter<
    TranscriptTreeNode | undefined
  >();
  readonly onDidChangeTreeData = this.onDidChangeTreeDataEmitter.event;

  private readonly unsubscribe: () => void;

  constructor(private readonly transcriptStore: TranscriptStore) {
    this.unsubscribe = this.transcriptStore.onDidChange(() => {
      this.onDidChangeTreeDataEmitter.fire(undefined);
    });
  }

  getTreeItem(node: TranscriptTreeNode): vscode.TreeItem {
    if (node.kind === "header") {
      const item = new vscode.TreeItem(
        node.label,
        vscode.TreeItemCollapsibleState.None,
      );
      item.description = node.description;
      item.contextValue = "vocode-transcript-header";
      return item;
    }

    const item = new vscode.TreeItem(
      node.entry.text,
      vscode.TreeItemCollapsibleState.None,
    );
    item.description = `${node.entry.kind} • ${formatTimestamp(node.entry.receivedAt)}`;
    item.contextValue = `vocode-transcript-${node.entry.kind}`;
    item.tooltip = `${node.entry.text}\n(${node.entry.kind})`;
    item.iconPath =
      node.entry.kind === "final"
        ? new vscode.ThemeIcon("check")
        : new vscode.ThemeIcon("circle-large-outline");

    return item;
  }

  getChildren(): TranscriptTreeNode[] {
    const entries = this.transcriptStore.getEntries();

    const partialCount = entries.filter(
      (entry) => entry.kind === "partial",
    ).length;
    const finalCount = entries.filter((entry) => entry.kind === "final").length;

    const nodes: TranscriptTreeNode[] = [
      {
        kind: "header",
        label: "Partial",
        description: `${partialCount}`,
      },
      {
        kind: "header",
        label: "Final",
        description: `${finalCount}`,
      },
    ];

    for (const entry of entries) {
      nodes.push({ kind: "entry", entry });
    }

    return nodes;
  }

  dispose(): void {
    this.unsubscribe();
    this.onDidChangeTreeDataEmitter.dispose();
  }
}

function formatTimestamp(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}
