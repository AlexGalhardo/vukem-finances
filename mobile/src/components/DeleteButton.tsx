import { type ReactNode, useState } from "react";
import { Text, View } from "react-native";
import { Button } from "./Button";
import { useUi } from "./ui";

/** Delete in two presses: the first asks "Delete this item?", the second confirms. Both are visible buttons. */
export function DeleteButton(props: { onConfirm: () => void }): ReactNode {
	const ui = useUi();
	const [asking, setAsking] = useState(false);

	if (!asking) {
		return <Button variant="destructive" label={ui.t("delete")} onPress={() => setAsking(true)} />;
	}
	return (
		<View style={{ gap: ui.spacing[2] }}>
			<Text accessibilityRole="alert" style={{ color: ui.colors.text, fontSize: ui.fontSize.base }}>
				{ui.t("confirmDelete")}
			</Text>
			<Button
				variant="destructive"
				label={ui.t("delete")}
				onPress={() => {
					setAsking(false);
					props.onConfirm();
				}}
			/>
			<Button variant="secondary" label={ui.t("cancel")} onPress={() => setAsking(false)} />
		</View>
	);
}
