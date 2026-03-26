import type * as vscode from "vscode";

import { registerCommands } from "./helpers";
import { pingCommand } from "./ping";
import { sendTranscriptCommand } from "./send-transcript";
import type { ExtensionServices } from "./services";
import { startVoiceCommand } from "./start-voice";
import { stopVoiceCommand } from "./stop-voice";
import type { CommandDefinition } from "./types";

const definitions: CommandDefinition[] = [
  pingCommand,
  startVoiceCommand,
  stopVoiceCommand,
  sendTranscriptCommand,
];

export function registerAllCommands(
  services: ExtensionServices,
): vscode.Disposable[] {
  return registerCommands(services, definitions);
}
