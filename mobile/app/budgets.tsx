import { useRouter } from "expo-router";
import { BudgetsScreen } from "../src/screens/BudgetsScreen";

export default function Route(): React.JSX.Element {
  const router = useRouter();
  return <BudgetsScreen onBack={() => router.back()} />;
}
