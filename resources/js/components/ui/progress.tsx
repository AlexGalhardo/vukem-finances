import { Progress as ProgressPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "../../lib/utils";

type ProgressProps = Omit<React.ComponentProps<typeof ProgressPrimitive.Root>, "value" | "max"> & {
  value: number;
  max?: number;
};

/**
 * An 8 px track. The fill is brand green, turns to warning above 80% and to expense above 100% with a
 * stripe, so the state is never color alone. Always show the numbers in text next to it.
 */
function Progress({ className, value, max = 100, ...props }: ProgressProps) {
  const ratio = max > 0 ? value / max : 0;
  const tone = ratio > 1 ? "over" : ratio > 0.8 ? "warn" : "ok";
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      data-tone={tone}
      value={Math.min(Math.max(value, 0), max)}
      max={max}
      className={cn("relative h-2 w-full overflow-hidden rounded-full bg-muted", className)}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className={cn(
          "h-full rounded-full transition-[width] duration-(--vf-duration-slow)",
          tone === "ok" && "bg-brand",
          tone === "warn" && "bg-warning",
          tone === "over" &&
            "bg-expense bg-[repeating-linear-gradient(45deg,transparent_0_4px,color-mix(in_srgb,var(--background)_45%,transparent)_4px_8px)]",
        )}
        style={{ width: `${Math.min(Math.max(ratio, 0), 1) * 100}%` }}
      />
    </ProgressPrimitive.Root>
  );
}

export { Progress };
