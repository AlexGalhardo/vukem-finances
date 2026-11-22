import { useRouter } from "expo-router";
import { TagsScreen } from "../src/screens/TagsScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <TagsScreen onBack={() => router.back()} />;
}
