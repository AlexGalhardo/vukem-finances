import type { ReactNode } from "react";
import { Text, TextInput, type TextInputProps, View } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/** A labelled text input at least 48 dp tall; the label is also its accessibility label. */
export function TextField(props: {
	label: string;
	value: string;
	onChangeText: (text: string) => void;
	error?: string | null;
	placeholder?: string;
	keyboardType?: TextInputProps["keyboardType"];
	autoCapitalize?: TextInputProps["autoCapitalize"];
	multiline?: boolean;
}): ReactNode {
	const ui = useUi();
	const { label, value, onChangeText, error = null, placeholder, keyboardType, autoCapitalize, multiline = false } = props;

	return (
		<View>
			<Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{label}</Text>
			<TextInput
				accessibilityLabel={label}
				value={value}
				onChangeText={onChangeText}
				placeholder={placeholder}
				placeholderTextColor={ui.colors.textMuted}
				keyboardType={keyboardType}
				autoCapitalize={autoCapitalize}
				multiline={multiline}
				style={{
					minHeight: TOUCH_MIN,
					fontSize: ui.fontSize.base,
					color: ui.colors.text,
					borderWidth: 1,
					borderColor: error === null ? ui.colors.controlBorder : ui.colors.expense,
					borderRadius: ui.radius,
					paddingHorizontal: ui.spacing[3],
				}}
			/>
			{error === null ? null : (
				<Text accessibilityLiveRegion="polite" style={{ color: ui.colors.expense, fontSize: ui.fontSize.sm }}>
					{error}
				</Text>
			)}
		</View>
	);
}
