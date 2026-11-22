import { useRouter } from "expo-router";
import { CategoriesScreen } from "../src/screens/CategoriesScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <CategoriesScreen onBack={() => router.back()} />;
}
