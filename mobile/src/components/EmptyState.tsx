import type { ReactNode } from "react";
import { Pressable, Text, View } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/** What a list shows when it has nothing: a sentence and, when something can be done, one button. */
export function EmptyState(props: { message: string; action?: { label: string; onPress: () => void } }): ReactNode {
	const ui = useUi();
	const { message, action } = props;

	return (
		<View style={{ alignItems: "center", padding: ui.spacing[6], gap: ui.spacing[4] }}>
			<Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.base, textAlign: "center" }}>{message}</Text>
			{action === undefined ? null : (
				<Pressable
					accessibilityRole="button"
					accessibilityLabel={action.label}
					onPress={action.onPress}
					style={{
						minHeight: TOUCH_MIN,
						justifyContent: "center",
						paddingHorizontal: ui.spacing[6],
						borderRadius: ui.radius,
						backgroundColor: ui.colors.brand,
					}}
				>
					<Text style={{ color: ui.colors.onBrand, fontSize: ui.fontSize.base }}>{action.label}</Text>
				</Pressable>
			)}
		</View>
	);
}
