import { useRouter } from "expo-router";
import { CardsScreen } from "../src/screens/CardsScreen";

export default function Cards(): React.JSX.Element {
  const router = useRouter();
  return <CardsScreen onBack={() => router.back()} />;
}
