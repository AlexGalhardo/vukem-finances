import { getCalendars, getLocales } from "expo-localization";
import { type ReactNode, useEffect, useState } from "react";
import { ActivityIndicator, useColorScheme, View } from "react-native";
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
import { defaultSettings, loadSettings, type Settings } from "./settings";
import { AppProvider, useApp } from "./state";

interface Ready {
  db: TransactionalDatabase;
  repo: Repository;
  ledger: Ledger;
  settings: Settings;
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
      if (live) setReady({ db, repo, ledger: new Ledger(db, repo), settings });
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
    >
      <Gate>{props.children}</Gate>
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
