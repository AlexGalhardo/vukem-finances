import type { ComponentProps, ReactNode } from "react";
import { cn } from "../lib/utils";
import { Input } from "./ui/input";

type AmountInputProps = Omit<ComponentProps<typeof Input>, "type" | "inputMode"> & {
  /** The currency symbol or code shown inside the field, before the figures. */
  prefix?: ReactNode;
};

/**
 * A text input for an amount: right-aligned, tabular figures. It hands the raw string to the server, which owns
 * parsing (locale separators, rounding to cents); the browser never turns it into a float.
 */
export default function AmountInput({ prefix, className, ...props }: AmountInputProps) {
  const input = (
    <Input
      {...props}
      type="text"
      inputMode="decimal"
      enterKeyHint="next"
      autoComplete="off"
      spellCheck={false}
      className={cn("text-right tabular-nums", prefix !== undefined && "ps-12", className)}
    />
  );
  if (prefix === undefined) return input;
  return (
    <div className="relative">
      <span
        aria-hidden="true"
        className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-body text-muted-foreground"
      >
        {prefix}
      </span>
      {input}
    </div>
  );
}
