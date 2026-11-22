import { useRouter } from "expo-router";
import { HomeScreen } from "../../src/screens/HomeScreen";

export default function Home(): React.JSX.Element {
  const router = useRouter();
  return <HomeScreen onAdd={() => router.push("/add")} />;
}
