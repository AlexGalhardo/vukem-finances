import type { ReactNode } from "react";
import { Text, TextInput, View } from "react-native";
import { parseAmount } from "../domain/money";
import { TOUCH_MIN, useUi } from "./ui";

/**
 * The amount, first thing on the Add form: a numeric keypad, a visible label
 * and an error that a screen reader announces. The text stays as typed; the
 * caller turns it into cents with `parseAmount` (the one parser shared with the server).
 */
export function AmountField(props: {
  value: string;
  onChangeText: (text: string) => void;
  label?: string;
  /** Shows the "enter an amount" error when the text is not empty and does not parse. */
  showError?: boolean;
  autoFocus?: boolean;
}): ReactNode {
  const ui = useUi();
  const { value, onChangeText, autoFocus = false } = props;
  const label = props.label ?? ui.t("amount");
  const invalid = (props.showError ?? true) && value.trim() !== "" && parseAmount(value) === null;

  return (
    <View>
      <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        accessibilityHint={invalid ? ui.t("invalidAmount") : undefined}
        keyboardType="decimal-pad"
        inputMode="decimal"
        value={value}
        onChangeText={onChangeText}
        autoFocus={autoFocus}
        placeholder="0.00"
        placeholderTextColor={ui.colors.textMuted}
        style={{
          minHeight: TOUCH_MIN,
          fontSize: ui.fontSize.xl,
          color: ui.colors.text,
          borderWidth: 1,
          borderColor: invalid ? ui.colors.expense : ui.colors.controlBorder,
          borderRadius: ui.radius,
          paddingHorizontal: ui.spacing[3],
        }}
      />
      {invalid ? (
        <Text accessibilityLiveRegion="polite" style={{ color: ui.colors.expense, fontSize: ui.fontSize.sm }}>
          {ui.t("invalidAmount")}
        </Text>
      ) : null}
    </View>
  );
}
