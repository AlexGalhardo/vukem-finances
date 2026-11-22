import type { ReactNode } from "react";
import { Pressable, Text } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/** A full-width action button; `destructive` uses the expense color and keeps its text label. */
export function Button(props: {
	label: string;
	onPress: () => void;
	variant?: "primary" | "secondary" | "destructive";
	disabled?: boolean;
}): ReactNode {
	const ui = useUi();
	const { label, onPress, variant = "primary", disabled = false } = props;
	const filled = variant === "primary";
	const accent = variant === "destructive" ? ui.colors.expense : ui.colors.brand;

	return (
		<Pressable
			accessibilityRole="button"
			accessibilityLabel={label}
			accessibilityState={{ disabled }}
			disabled={disabled}
			onPress={onPress}
			style={{
				minHeight: TOUCH_MIN,
				justifyContent: "center",
				alignItems: "center",
				paddingHorizontal: ui.spacing[6],
				borderRadius: ui.radius,
				borderWidth: 1,
				borderColor: accent,
				backgroundColor: filled ? accent : ui.colors.surface,
				opacity: disabled ? 0.5 : 1,
			}}
		>
			<Text style={{ color: filled ? ui.colors.onBrand : accent, fontSize: ui.fontSize.base, fontWeight: "600" }}>
				{label}
			</Text>
		</Pressable>
	);
}
