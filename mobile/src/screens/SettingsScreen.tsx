import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { CURRENCIES, type ThemeSetting } from "../core/settings";
import { type App, useApp, useLoad } from "../core/state";
import { backupFileName, buildBackup } from "../backup/export";
import type { Sharer } from "../backup/share";
import { Button } from "../components/Button";
import { Chip } from "../components/Chip";
import { Page } from "../components/Page";
import { TextField } from "../components/TextField";
import { Toast } from "../components/Toast";
import { DeleteButton } from "../components/DeleteButton";
import { useUi } from "../components/ui";
import { isValidTimezone } from "../domain/time";
import { LOCALE_NAMES, LOCALES, type MessageKey } from "../i18n";
import type { LockPlatform, LockStore } from "../lock/lock";
import { OUTBOX_SOFT_CAP } from "../db/repository";

export interface SettingsServices {
  lockStore: LockStore;
  lockPlatform: LockPlatform;
  sharer: Sharer;
  version: string;
}

const THEMES: { value: ThemeSetting; label: MessageKey }[] = [
  { value: "system", label: "themeSystem" },
  { value: "light", label: "themeLight" },
  { value: "dark", label: "themeDark" },
];

/** Language, theme, time zone, currency, app lock, local backup, the server placeholder and about. */
export function SettingsScreen(props: { services: SettingsServices; onBack?: () => void }): ReactNode {
  const { services, onBack } = props;
  const app = useApp();
  const ui = useUi();
  const [timezone, setTimezone] = useState(app.settings.timezone);
  const [timezoneError, setTimezoneError] = useState<string | null>(null);
  const [lockOn, setLockOn] = useState<boolean | null>(null);
  const [lockNote, setLockNote] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [confirmingOff, setConfirmingOff] = useState(false);

  const lock = useLoad(async () => services.lockStore.isEnabled());
  const enabled = lockOn ?? lock ?? false;
  const queue = useLoad(async (a: App) => a.repo.outboxCount());

  async function toggleLock(): Promise<void> {
    if (enabled) {
      await services.lockStore.setEnabled(false);
      setLockOn(false);
      setLockNote(null);
      return;
    }
    // Turning it on needs a screen lock on the device and a successful check, so the person cannot lock themselves out.
    if (!(await services.lockPlatform.available())) {
      setLockNote(ui.t("settingsLockUnavailable"));
      return;
    }
    if (await services.lockPlatform.authenticate(ui.t("lockPrompt"))) {
      await services.lockStore.setEnabled(true);
      setLockOn(true);
      setLockNote(null);
    }
  }

  async function saveTimezone(): Promise<void> {
    if (!isValidTimezone(timezone)) {
      setTimezoneError(ui.t("errorGeneric"));
      return;
    }
    setTimezoneError(null);
    await app.updateSettings({ timezone });
  }

  async function exportBackup(): Promise<void> {
    try {
      const today = app.today();
      const backup = await buildBackup(
        app.db,
        { name: ui.t("appName"), currency: app.settings.currency, timezone: app.settings.timezone },
        new Date(),
      );
      await services.sharer.shareText(backupFileName(today), JSON.stringify(backup, null, 2));
      setMessage(null);
    } catch {
      setMessage(ui.t("settingsBackupFailed"));
    }
  }

  async function turnQueueOff(): Promise<void> {
    await app.repo.clearOutbox();
    await app.updateSettings({ keepSyncQueue: false });
    setConfirmingOff(false);
    app.changed();
  }

  const section = (title: string, children: ReactNode): ReactNode => (
    <View style={{ gap: ui.spacing[2] }}>
      <Text accessibilityRole="header" style={{ color: ui.colors.text, fontSize: ui.fontSize.lg }}>
        {title}
      </Text>
      {children}
    </View>
  );
  const row = { flexDirection: "row", flexWrap: "wrap", gap: ui.spacing[2] } as const;

  return (
    <Page title={ui.t("titleSettings")} onBack={onBack}>
      {section(
        ui.t("settingsLanguage"),
        <View style={row}>
          {LOCALES.map((l) => (
            <Chip
              key={l}
              role="radio"
              label={LOCALE_NAMES[l]}
              selected={app.settings.locale === l}
              onPress={() => void app.updateSettings({ locale: l })}
            />
          ))}
        </View>,
      )}
      {section(
        ui.t("settingsTheme"),
        <View style={row}>
          {THEMES.map((t) => (
            <Chip
              key={t.value}
              role="radio"
              label={ui.t(t.label)}
              selected={app.settings.theme === t.value}
              onPress={() => void app.updateSettings({ theme: t.value })}
            />
          ))}
        </View>,
      )}
      {section(
        ui.t("settingsCurrency"),
        <View style={row}>
          {CURRENCIES.map((c) => (
            <Chip
              key={c}
              role="radio"
              label={c}
              selected={app.settings.currency === c}
              onPress={() => void app.updateSettings({ currency: c })}
            />
          ))}
        </View>,
      )}
      {section(
        ui.t("settingsTimezone"),
        <>
          <TextField
            label={ui.t("settingsTimezone")}
            value={timezone}
            onChangeText={setTimezone}
            error={timezoneError}
            autoCapitalize="none"
          />
          <Button variant="secondary" label={ui.t("save")} onPress={() => void saveTimezone()} />
        </>,
      )}
      {section(
        ui.t("settingsLock"),
        <>
          <Chip role="checkbox" label={ui.t("settingsLock")} selected={enabled} onPress={() => void toggleLock()} />
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("settingsLockHelp")}</Text>
          {lockNote === null ? null : (
            <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
              {lockNote}
            </Text>
          )}
        </>,
      )}
      {section(
        ui.t("settingsBackup"),
        <>
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("settingsBackupHelp")}</Text>
          <Button label={ui.t("settingsBackupExport")} onPress={() => void exportBackup()} />
          {message === null ? null : <Toast message={message} onDismiss={() => setMessage(null)} />}
        </>,
      )}
      {section(
        ui.t("settingsServer"),
        <>
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("settingsServerHelp")}</Text>
          {queue === undefined ? null : (
            <Text style={{ color: ui.colors.text }}>{ui.t("settingsQueueCount", { count: queue })}</Text>
          )}
          <Chip
            role="checkbox"
            label={ui.t("settingsQueueKeep")}
            selected={app.settings.keepSyncQueue}
            onPress={() => {
              if (app.settings.keepSyncQueue) setConfirmingOff(true);
              else void app.updateSettings({ keepSyncQueue: true });
            }}
          />
          {confirmingOff ? (
            <View style={{ gap: ui.spacing[2] }}>
              <Text accessibilityRole="alert" style={{ color: ui.colors.text }}>
                {ui.t("settingsQueueOffConfirm", { count: queue ?? 0 })}
              </Text>
              <Button variant="destructive" label={ui.t("delete")} onPress={() => void turnQueueOff()} />
              <Button variant="secondary" label={ui.t("cancel")} onPress={() => setConfirmingOff(false)} />
            </View>
          ) : null}
          {queue !== undefined && queue > OUTBOX_SOFT_CAP ? (
            <View style={{ gap: ui.spacing[2] }}>
              <Text accessibilityRole="alert" style={{ color: ui.colors.text }}>
                {ui.t("settingsQueueFull", { count: OUTBOX_SOFT_CAP })}
              </Text>
              <DeleteButton
                onConfirm={() => {
                  void app.repo.dropOldestOutbox(true).then(() => app.changed());
                }}
              />
            </View>
          ) : null}
        </>,
      )}
      {section(
        ui.t("settingsAbout"),
        <>
          <Text style={{ color: ui.colors.text }}>{ui.t("settingsVersion", { version: services.version })}</Text>
          <Text style={{ color: ui.colors.text }}>{`${ui.t("settingsLicense")}: MIT`}</Text>
        </>,
      )}
    </Page>
  );
}
