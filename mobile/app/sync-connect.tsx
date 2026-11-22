import { useRouter } from "expo-router";
import { SyncConnectScreen } from "../src/screens/SyncConnectScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <SyncConnectScreen onBack={() => router.back()} onConnected={() => router.replace("/sync")} />;
}
