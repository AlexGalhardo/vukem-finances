import { useRouter } from "expo-router";
import { SyncConflictsScreen } from "../src/screens/SyncConflictsScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <SyncConflictsScreen onBack={() => router.back()} />;
}
