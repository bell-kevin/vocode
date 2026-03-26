import type { CommandDefinition } from "../types";
import { runSendTranscript } from "./run";

export const sendTranscriptCommand: CommandDefinition = {
  id: "vocode.sendTranscript",
  requiresDaemon: true,
  run: (client, services) => runSendTranscript(client, services),
};
