import { Fragment } from "react";
import { cn } from "../lib/utils";
import { formatBindingParts } from "../shortcuts/format";

/**
 * A key cap. On a default or brand button it turns translucent so it follows the label color; inside a tooltip it
 * inverts through the tooltip's own `[data-slot=kbd]` rule.
 */
export function Kbd({ className, ...props }: React.ComponentProps<"kbd">) {
  return (
    <kbd
      data-slot="kbd"
      className={cn(
        "inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-sm border border-border bg-muted px-1 font-sans text-kbd text-muted-foreground group-data-[variant=brand]/button:border-current/20 group-data-[variant=brand]/button:bg-current/10 group-data-[variant=brand]/button:text-current group-data-[variant=default]/button:border-current/20 group-data-[variant=default]/button:bg-current/10 group-data-[variant=default]/button:text-current",
        className,
      )}
      {...props}
    />
  );
}

/** Renders a binding such as "g d" or "mod+k" as Kbd elements. */
export function Shortcut({ keys, isMac = false }: { keys: string; isMac?: boolean }) {
  const parts = formatBindingParts(keys, isMac);
  return (
    <span className="inline-flex items-center gap-0.5">
      {parts.map((part, index) => (
        // The binding is static, so the position is a stable key.
        // biome-ignore lint/suspicious/noArrayIndexKey: same label can repeat in a sequence ("g g")
        <Fragment key={index}>
          <Kbd>{part}</Kbd>
        </Fragment>
      ))}
    </span>
  );
}

/** The key caps of a hint, hidden by CSS on touch and on small screens (finance.css), so markup always matches. */
export function ShortcutHint({ parts, className }: { parts: readonly string[]; className?: string }) {
  return (
    <span data-slot="shortcut-hint" aria-hidden="true" className={cn("ms-1 items-center gap-0.5", className)}>
      {parts.map((part, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: same label can repeat in a sequence ("g g")
        <Kbd key={index}>{part}</Kbd>
      ))}
    </span>
  );
}
