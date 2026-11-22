import { useRouter } from "expo-router";
import { SyncProblemsScreen } from "../src/screens/SyncProblemsScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <SyncProblemsScreen onBack={() => router.back()} />;
}
