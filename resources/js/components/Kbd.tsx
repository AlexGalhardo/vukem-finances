import { Fragment } from "react";
import { cn } from "../lib/utils";
import { formatBindingParts } from "../shortcuts/format";

export function Kbd({ className, ...props }: React.ComponentProps<"kbd">) {
  return (
    <kbd
      className={cn(
        "inline-flex min-w-6 items-center justify-center rounded-md border border-border bg-muted px-1.5 py-0.5 font-sans text-xs font-medium text-foreground",
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
    <span className="inline-flex items-center gap-1">
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
