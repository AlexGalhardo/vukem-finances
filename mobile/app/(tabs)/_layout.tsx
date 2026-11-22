import { Tabs } from "expo-router";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useUi } from "../../src/components/ui";

export default function TabsLayout(): React.JSX.Element {
  const ui = useUi();
  const insets = useSafeAreaInsets();
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        sceneStyle: { paddingTop: insets.top, backgroundColor: ui.colors.bg },
        tabBarActiveTintColor: ui.colors.brandText,
        tabBarInactiveTintColor: ui.colors.textMuted,
        tabBarStyle: { backgroundColor: ui.colors.surface },
      }}
    >
      <Tabs.Screen name="index" options={{ title: ui.t("tabHome") }} />
      <Tabs.Screen name="transactions" options={{ title: ui.t("tabTransactions") }} />
      <Tabs.Screen name="add" options={{ title: ui.t("tabAdd") }} />
      <Tabs.Screen name="accounts" options={{ title: ui.t("tabAccounts") }} />
      <Tabs.Screen name="more" options={{ title: ui.t("tabMore") }} />
    </Tabs>
  );
}
