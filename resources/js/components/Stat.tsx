import { ArrowDownRight, ArrowUpRight } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../lib/utils";

export interface StatProps {
  label: string;
  /** Usually a MoneyText. */
  value: ReactNode;
  /** A HelpHint for the figure, placed beside the label. */
  help?: ReactNode;
  /** The change against the previous period: the text already carries its own sign. */
  delta?: { text: string; direction: "up" | "down" | "flat"; favorable?: boolean };
  size?: "md" | "hero";
  className?: string;
}

/** A labelled figure: the caption on top, the number below, an optional change with a sign and an arrow. */
export default function Stat({ label, value, help, delta, size = "md", className }: StatProps) {
  const Arrow = delta?.direction === "down" ? ArrowDownRight : ArrowUpRight;
  return (
    <div data-slot="stat" className={cn("flex min-w-0 flex-col gap-1", className)}>
      <div className="flex items-center gap-1.5 text-caption text-muted-foreground">
        <span>{label}</span>
        {help}
      </div>
      <div className={cn("tabular-nums", size === "hero" ? "text-hero-number" : "text-h2")}>{value}</div>
      {delta && (
        <div
          className={cn(
            "flex items-center gap-1 text-caption tabular-nums",
            delta.favorable === undefined ? "text-muted-foreground" : delta.favorable ? "text-income" : "text-expense",
          )}
        >
          {delta.direction !== "flat" && <Arrow aria-hidden="true" className="size-3.5" />}
          {delta.text}
        </div>
      )}
    </div>
  );
}
