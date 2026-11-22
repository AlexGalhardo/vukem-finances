import { useRouter } from "expo-router";
import { ReportsScreen } from "../src/screens/ReportsScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <ReportsScreen onBack={() => router.back()} />;
}
