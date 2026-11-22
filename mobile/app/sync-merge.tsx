import { useRouter } from "expo-router";
import { systemSharer } from "../src/backup/share";
import { SyncMergeScreen } from "../src/screens/SyncMergeScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <SyncMergeScreen sharer={systemSharer} onBack={() => router.back()} onDone={() => router.replace("/sync")} />;
}
