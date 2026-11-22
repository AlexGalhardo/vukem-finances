import { type ReactNode, useState } from "react";
import { Platform, Text } from "react-native";
import { Button } from "../components/Button";
import { Page } from "../components/Page";
import { TextField } from "../components/TextField";
import { useUi } from "../components/ui";
import type { MessageKey } from "../i18n";
import { useSync } from "../sync/context";

const PROBLEMS: Record<string, MessageKey> = {
  empty: "syncErrEmpty",
  scheme: "syncErrScheme",
  malformed: "syncErrMalformed",
  public_http: "syncErrPublicHttp",
  pairing: "syncErrPairing",
  network: "syncErrNetwork",
  rate: "syncErrRate",
};

/** Server address, pairing code and device name. Plain http to a private address needs a confirmation that says it is not encrypted. */
export function SyncConnectScreen(props: { onBack?: () => void; onConnected: () => void }): ReactNode {
  const ui = useUi();
  const sync = useSync();
  const [url, setUrl] = useState("");
  const [code, setCode] = useState("");
  const [name, setName] = useState(sync.defaultDeviceName);
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  async function connect(confirmedInsecure: boolean): Promise<void> {
    if (code.trim() === "") {
      setError(ui.t("syncErrCode"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const result = await sync.connect({
        serverUrl: url,
        confirmedInsecure,
        code,
        deviceName: name.trim() === "" ? sync.defaultDeviceName : name,
        platform: Platform.OS === "ios" ? "ios" : "android",
      });
      if (result.ok) {
        // The code did its job: it is not kept.
        setCode("");
        props.onConnected();
        return;
      }
      if (result.reason === "needs_insecure_confirmation") {
        setConfirming(true);
        return;
      }
      setConfirming(false);
      setError(ui.t(PROBLEMS[result.reason] ?? "errorGeneric"));
    } catch {
      setError(ui.t("errorGeneric"));
    } finally {
      setBusy(false);
    }
  }

  if (confirming) {
    return (
      <Page title={ui.t("syncInsecureTitle")} onBack={() => setConfirming(false)}>
        <Text style={{ color: ui.colors.text }}>{ui.t("syncInsecureBody", { url: url.trim() })}</Text>
        <Button
          label={ui.t("syncInsecureConfirm")}
          variant="destructive"
          disabled={busy}
          onPress={() => void connect(true)}
        />
        <Button label={ui.t("cancel")} variant="secondary" onPress={() => setConfirming(false)} />
      </Page>
    );
  }

  return (
    <Page title={ui.t("titleSyncConnect")} onBack={props.onBack}>
      <TextField
        label={ui.t("syncServerAddress")}
        value={url}
        onChangeText={setUrl}
        keyboardType="url"
        autoCapitalize="none"
        placeholder="https://"
      />
      <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("syncAddressHelp")}</Text>
      <TextField label={ui.t("syncPairingCode")} value={code} onChangeText={setCode} autoCapitalize="characters" />
      <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{ui.t("syncPairingHelp")}</Text>
      <TextField label={ui.t("syncDeviceName")} value={name} onChangeText={setName} />
      {error === null ? null : (
        <Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
          {error}
        </Text>
      )}
      <Button label={ui.t("syncConnectButton")} disabled={busy} onPress={() => void connect(false)} />
    </Page>
  );
}
