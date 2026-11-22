import { Stack } from "expo-router";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
import { Bootstrap } from "../src/core/Bootstrap";

function Screens(): React.JSX.Element {
	const insets = useSafeAreaInsets();
	return <Stack screenOptions={{ headerShown: false, contentStyle: { paddingTop: insets.top } }} />;
}

export default function RootLayout(): React.JSX.Element {
	return (
		<SafeAreaProvider>
			<Bootstrap>
				<Screens />
			</Bootstrap>
		</SafeAreaProvider>
	);
}
