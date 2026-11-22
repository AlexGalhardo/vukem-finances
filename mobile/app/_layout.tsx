import { Slot } from "expo-router";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { Bootstrap } from "../src/app/Bootstrap";

export default function RootLayout(): React.JSX.Element {
	return (
		<SafeAreaProvider>
			<Bootstrap>
				<Slot />
			</Bootstrap>
		</SafeAreaProvider>
	);
}
