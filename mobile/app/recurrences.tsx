import { useRouter } from "expo-router";
import { RecurrencesScreen } from "../src/screens/RecurrencesScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <RecurrencesScreen onBack={() => router.back()} />;
}
