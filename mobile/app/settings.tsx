import { useRouter } from "expo-router";
import pkg from "../package.json";
import { systemSharer } from "../src/backup/share";
import { devicePlatform, secureLockStore } from "../src/lock/platform";
import { SettingsScreen } from "../src/screens/SettingsScreen";

const services = {
  lockStore: secureLockStore,
  lockPlatform: devicePlatform,
  sharer: systemSharer,
  version: pkg.version,
};

export default function Settings(): React.JSX.Element {
  const router = useRouter();
  return <SettingsScreen services={services} onBack={() => router.back()} />;
}
