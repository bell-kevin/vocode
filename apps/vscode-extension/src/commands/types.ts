import type { DaemonClient } from "../client/daemon-client";
import type { ExtensionServices } from "./services";

export type CommandDefinition =
  | {
      id: string;
      requiresDaemon: false;
      run: (services: ExtensionServices) => void | Promise<void>;
    }
  | {
      id: string;
      requiresDaemon: true;
      run: (
        client: DaemonClient,
        services: ExtensionServices,
      ) => void | Promise<void>;
    };
