import type { ReactNode } from "react";
import { Text, type TextStyle } from "react-native";
import { useUi } from "./ui";

export type MoneyKind = "income" | "expense" | "neutral";

/**
 * Integer cents as text. Income and expense never rely on color alone: they
 * carry a sign in the text and a spoken label ("Expense minus 12 dollars 50
 * cents"). A neutral amount (a balance) shows its own sign.
 */
export function MoneyText(props: { cents: number; kind?: MoneyKind; style?: TextStyle }): ReactNode {
  const { cents, kind = "neutral", style } = props;
  const ui = useUi();
  const magnitude = Math.abs(cents);

  let shown: string;
  let spoken: string;
  let color: string = ui.colors.text;
  if (kind === "income") {
    shown = `+${ui.money(magnitude)}`;
    spoken = ui.t("incomeSign", { amount: ui.spoken(magnitude) });
    color = ui.colors.income;
  } else if (kind === "expense") {
    shown = `-${ui.money(magnitude)}`;
    spoken = ui.t("expenseSign", { amount: ui.spoken(-magnitude) });
    color = ui.colors.expense;
  } else {
    shown = ui.money(cents);
    spoken = ui.spoken(cents);
  }

  return (
    <Text
      accessibilityRole="text"
      accessibilityLabel={spoken}
      style={[{ color, fontSize: ui.fontSize.base, fontVariant: ["tabular-nums"] }, style]}
    >
      {shown}
    </Text>
  );
}
