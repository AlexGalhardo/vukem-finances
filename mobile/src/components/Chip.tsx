import type { ReactNode } from "react";
import { Pressable, Text } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/**
 * A choice chip. `selected` is exposed to assistive technology and also drawn
 * with a check mark, so the state is never only a color.
 */
export function Chip(props: {
	label: string;
	onPress: () => void;
	selected?: boolean;
	/** "radio" for one choice among several, "checkbox" for filters that can all be on. */
	role?: "button" | "radio" | "checkbox";
}): ReactNode {
	const ui = useUi();
	const { label, onPress, selected = false, role = "button" } = props;

	return (
		<Pressable
			accessibilityRole={role}
			accessibilityLabel={label}
			accessibilityState={role === "button" ? undefined : { selected, checked: selected }}
			onPress={onPress}
			style={{
				minHeight: TOUCH_MIN,
				minWidth: TOUCH_MIN,
				justifyContent: "center",
				paddingHorizontal: ui.spacing[4],
				borderRadius: ui.radius,
				borderWidth: 1,
				borderColor: selected ? ui.colors.brand : ui.colors.controlBorder,
				backgroundColor: selected ? ui.colors.brand : ui.colors.surface,
			}}
		>
			<Text style={{ color: selected ? ui.colors.onBrand : ui.colors.text, fontSize: ui.fontSize.base }}>
				{selected ? `✓ ${label}` : label}
			</Text>
		</Pressable>
	);
}
