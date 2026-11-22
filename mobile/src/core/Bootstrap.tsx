import { getCalendars, getLocales } from "expo-localization";
import { type ReactNode, useEffect, useState } from "react";
import { ActivityIndicator, Platform, useColorScheme, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import type { TransactionalDatabase } from "../db/database";
import { openLocalDatabase } from "../db/expo";
import { migrate } from "../db/migrations";
import { Repository } from "../db/repository";
import { Ledger } from "../domain/ledger";
import { resolveLocale } from "../i18n";
import { LockGate } from "../lock/LockGate";
import { devicePlatform, secureLockStore } from "../lock/platform";
import { FirstRunScreen } from "../screens/FirstRunScreen";
import { SyncProvider } from "../sync/context";
import { SyncController } from "../sync/controller";
import { secureTokenStore } from "../sync/secure-token";
import { HttpTransport } from "../sync/transport";
import { systemClock, systemRandom } from "../sync/types";
import { defaultSettings, loadSettings, type Settings } from "./settings";
import { AppProvider, useApp } from "./state";

interface Ready {
  db: TransactionalDatabase;
  repo: Repository;
  ledger: Ledger;
  settings: Settings;
  sync: SyncController;
}

/** Opens the local database, applies the migrations, loads the settings and shows the first run until it is done. */
export function Bootstrap(props: { children: ReactNode }): ReactNode {
  const [ready, setReady] = useState<Ready | null>(null);
  const [failure, setFailure] = useState<unknown>(null);
  const scheme = useColorScheme() === "dark" ? "dark" : "light";

  useEffect(() => {
    let live = true;
    (async (): Promise<void> => {
      const db = await openLocalDatabase();
      await migrate(db);
      const device = defaultSettings(resolveLocale(getLocales()[0]?.languageTag), getCalendars()[0]?.timeZone ?? "UTC");
      const settings = await loadSettings(db, device);
      const repo = new Repository(db);
      repo.setKeepOutbox(settings.keepSyncQueue);
      const sync = new SyncController({
        db,
        repo,
        tokens: secureTokenStore,
        makeTransport: (baseUrl, allowInsecure, deviceId) =>
          new HttpTransport({ baseUrl, allowInsecure, tokens: secureTokenStore, deviceId }),
        clock: systemClock,
        random: systemRandom,
        defaultDeviceName: Platform.OS === "ios" ? "iPhone" : "Android phone",
        platform: Platform.OS === "ios" ? "ios" : "android",
      });
      await sync.engine.start();
      if (live) setReady({ db, repo, ledger: new Ledger(db, repo), settings, sync });
    })().catch((error: unknown) => {
      if (live) setFailure(error);
    });
    return () => {
      live = false;
    };
  }, []);

  if (failure !== null) {
    throw failure;
  }
  if (ready === null) {
    return (
      <View style={{ flex: 1, justifyContent: "center" }}>
        <ActivityIndicator accessibilityLabel="Loading" />
      </View>
    );
  }

  return (
    <AppProvider
      db={ready.db}
      repo={ready.repo}
      ledger={ready.ledger}
      initialSettings={ready.settings}
      systemScheme={scheme}
      onChanged={() => ready.sync.engine.onLocalWrite()}
    >
      <SyncProvider controller={ready.sync}>
        <Gate>{props.children}</Gate>
      </SyncProvider>
    </AppProvider>
  );
}

function Gate(props: { children: ReactNode }): ReactNode {
  const app = useApp();
  return app.settings.onboarded ? (
    <LockGate store={secureLockStore} platform={devicePlatform}>
      {props.children}
    </LockGate>
  ) : (
    <SafeAreaView style={{ flex: 1 }}>
      <FirstRunScreen />
    </SafeAreaView>
  );
}
