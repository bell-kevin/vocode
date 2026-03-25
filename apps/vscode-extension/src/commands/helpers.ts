import * as vscode from "vscode";

import type { ExtensionServices } from "./services";
import type { CommandDefinition } from "./types";

export function registerCommands(
  services: ExtensionServices,
  definitions: CommandDefinition[],
): vscode.Disposable[] {
  return definitions.map((definition) =>
    vscode.commands.registerCommand(definition.id, async () => {
      try {
        if (definition.requiresDaemon) {
          if (!services.client) {
            void vscode.window.showErrorMessage(
              "Vocode daemon is not running.",
            );
            return;
          }

          await definition.run(services.client, services);
          return;
        }

        await definition.run(services);
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Unknown command error";

        console.error(`[vocode] command ${definition.id} failed:`, error);
        void vscode.window.showErrorMessage(
          `${definition.id} failed: ${message}`,
        );
      }
    }),
  );
}
