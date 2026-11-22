import { createContext, type ReactNode, useContext, useEffect, useState } from "react";
import { AppState } from "react-native";
import { useApp } from "../core/state";
import type { SyncController } from "./controller";
import type { EngineSnapshot } from "./engine";
import type { Connection } from "./state";

const SyncContext = createContext<SyncController | null>(null);

export function useSync(): SyncController {
  const controller = useContext(SyncContext);
  if (controller === null) {
    throw new Error("useSync needs a SyncProvider");
  }
  return controller;
}

/** Gives the sync screens their controller, and syncs when the app comes back to the foreground. */
export function SyncProvider(props: { controller: SyncController; children: ReactNode }): ReactNode {
  const { controller } = props;
  const app = useApp();
  const { reload } = app;

  // What a sync wrote reaches the screens: reload when a run ends, without waking the engine again.
  useEffect(() => {
    let running = controller.engine.getStatus() === "syncing";
    return controller.engine.subscribe(() => {
      const now = controller.engine.getStatus() === "syncing";
      if (running && !now) reload();
      running = now;
    });
  }, [controller, reload]);

  useEffect(() => {
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        void controller.engine.trigger("foreground");
      }
    });
    return () => subscription.remove();
  }, [controller]);
  return <SyncContext.Provider value={controller}>{props.children}</SyncContext.Provider>;
}

export interface SyncView extends EngineSnapshot {
  connection: Connection | null;
}

/** The engine's state, kept current while the screen shows it; `undefined` until the first read. */
export function useSyncView(): SyncView | undefined {
  const controller = useSync();
  const [view, setView] = useState<SyncView | undefined>(undefined);

  useEffect(() => {
    let live = true;
    const read = async (): Promise<void> => {
      const snapshot = await controller.engine.refresh();
      const connection = await controller.connection();
      if (live) setView({ ...snapshot, connection });
    };
    const unsubscribe = controller.engine.subscribe(() => {
      const snapshot = controller.engine.view();
      void controller.connection().then((connection) => {
        if (live) setView({ ...snapshot, connection });
      });
    });
    void read();
    return () => {
      live = false;
      unsubscribe();
    };
  }, [controller]);

  return view;
}
