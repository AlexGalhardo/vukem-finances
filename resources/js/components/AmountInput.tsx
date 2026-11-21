import type { ComponentProps } from "react";
import { Input } from "./ui/input";

/**
 * A text input for an amount. It hands the raw string to the server, which owns parsing
 * (locale separators, rounding to cents); the browser never turns it into a float.
 */
export default function AmountInput(props: Omit<ComponentProps<typeof Input>, "type" | "inputMode">) {
  return <Input {...props} type="text" inputMode="decimal" autoComplete="off" spellCheck={false} />;
}
