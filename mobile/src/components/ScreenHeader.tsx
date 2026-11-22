import type { ReactNode } from "react";
import { Pressable, Text, View } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/** The one heading of a screen, with an optional back button and one action button (never swipe-only). */
export function ScreenHeader(props: {
	title: string;
	onBack?: () => void;
	action?: { label: string; onPress: () => void };
}): ReactNode {
	const ui = useUi();
	const { title, onBack, action } = props;

	const button = (label: string, onPress: () => void): ReactNode => (
		<Pressable
			accessibilityRole="button"
			accessibilityLabel={label}
			onPress={onPress}
			style={{ minHeight: TOUCH_MIN, minWidth: TOUCH_MIN, justifyContent: "center", paddingHorizontal: ui.spacing[2] }}
		>
			<Text style={{ color: ui.colors.brand, fontSize: ui.fontSize.base }}>{label}</Text>
		</Pressable>
	);

	return (
		<View style={{ flexDirection: "row", alignItems: "center", gap: ui.spacing[2], paddingHorizontal: ui.spacing[2] }}>
			{onBack === undefined ? null : button(ui.t("back"), onBack)}
			<Text
				accessibilityRole="header"
				style={{ flex: 1, flexShrink: 1, color: ui.colors.text, fontSize: ui.fontSize.xl, fontWeight: "600" }}
			>
				{title}
			</Text>
			{action === undefined ? null : button(action.label, action.onPress)}
		</View>
	);
}
