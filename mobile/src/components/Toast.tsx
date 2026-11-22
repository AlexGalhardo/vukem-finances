import type { ReactNode } from "react";
import { Pressable, Text, View } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/**
 * A short message announced politely, with an optional action (Undo) and a
 * close button. It does not animate, so reduced motion needs no special case,
 * and it does not time out by itself: the screen decides when to remove it.
 */
export function Toast(props: {
  message: string;
  action?: { label: string; onPress: () => void };
  onDismiss?: () => void;
}): ReactNode {
  const ui = useUi();
  const { message, action, onDismiss } = props;

  const button = (label: string, onPress: () => void): ReactNode => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={{ minHeight: TOUCH_MIN, minWidth: TOUCH_MIN, justifyContent: "center", paddingHorizontal: ui.spacing[3] }}
    >
      <Text style={{ color: ui.colors.brand, fontSize: ui.fontSize.base, fontWeight: "600" }}>{label}</Text>
    </Pressable>
  );

  return (
    <View
      style={{
        flexDirection: "row",
        alignItems: "center",
        flexWrap: "wrap",
        gap: ui.spacing[2],
        padding: ui.spacing[3],
        borderRadius: ui.radius,
        borderWidth: 1,
        borderColor: ui.colors.border,
        backgroundColor: ui.colors.surface,
      }}
    >
      <Text
        accessibilityRole="alert"
        accessibilityLiveRegion="polite"
        style={{ flex: 1, flexShrink: 1, color: ui.colors.text, fontSize: ui.fontSize.base }}
      >
        {message}
      </Text>
      {action === undefined ? null : button(action.label, action.onPress)}
      {onDismiss === undefined ? null : button(ui.t("close"), onDismiss)}
    </View>
  );
}
