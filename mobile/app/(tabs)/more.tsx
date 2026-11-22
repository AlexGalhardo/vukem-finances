import { useRouter } from "expo-router";
import { MoreScreen } from "../../src/screens/MoreScreen";

export default function More(): React.JSX.Element {
	const router = useRouter();
	return <MoreScreen onOpen={(target) => router.push(`/${target}`)} />;
}
