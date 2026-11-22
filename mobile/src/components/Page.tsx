import type { ReactNode } from "react";
import { ScrollView } from "react-native";
import { ScreenHeader } from "./ScreenHeader";
import { useUi } from "./ui";

/** A scrolling screen with its one heading; the content flows with flex and wraps, so large text never clips. */
export function Page(props: {
	title: string;
	onBack?: () => void;
	action?: { label: string; onPress: () => void };
	children: ReactNode;
}): ReactNode {
	const ui = useUi();
	return (
		<ScrollView contentContainerStyle={{ padding: ui.spacing[4], gap: ui.spacing[4] }} keyboardShouldPersistTaps="handled">
			<ScreenHeader title={props.title} onBack={props.onBack} action={props.action} />
			{props.children}
		</ScrollView>
	);
}
