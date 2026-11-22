import { useRouter } from "expo-router";
import { SyncStatusScreen } from "../src/screens/SyncStatusScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return (
    <SyncStatusScreen
      onBack={() => router.back()}
      onConnect={() => router.push("/sync-connect")}
      onChoose={() => router.push("/sync-merge")}
      onConflicts={() => router.push("/sync-conflicts")}
      onProblems={() => router.push("/sync-problems")}
    />
  );
}
