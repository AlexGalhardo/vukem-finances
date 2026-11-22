import { CheckIcon, MinusIcon } from "lucide-react";
import { Checkbox as CheckboxPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "../../lib/utils";

/** 16 px box; on a coarse pointer the pseudo-element stretches the hit area to 44 px. */
function Checkbox({ className, checked, ...props }: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      checked={checked}
      className={cn(
        "peer relative inline-flex size-4 shrink-0 items-center justify-center rounded-sm border border-control bg-background text-brand-foreground transition-colors duration-(--vf-duration-fast) pointer-coarse:after:absolute pointer-coarse:after:-inset-3.5 pointer-coarse:after:content-[''] hover:border-muted-foreground disabled:pointer-events-none disabled:border-border disabled:bg-muted data-[state=checked]:border-brand data-[state=checked]:bg-brand data-[state=indeterminate]:border-brand data-[state=indeterminate]:bg-brand aria-invalid:border-expense",
        className,
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator data-slot="checkbox-indicator" className="flex items-center justify-center">
        {checked === "indeterminate" ? (
          <MinusIcon aria-hidden="true" className="size-3.5" />
        ) : (
          <CheckIcon aria-hidden="true" className="size-3.5" />
        )}
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

export { Checkbox };
