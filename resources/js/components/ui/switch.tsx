import { Switch as SwitchPrimitive } from "radix-ui";
import type * as React from "react";
import { cn } from "../../lib/utils";

type SwitchProps = React.ComponentProps<typeof SwitchPrimitive.Root> & {
  /** The state in words, shown beside the switch so it never relies on color alone. */
  onLabel?: string;
  offLabel?: string;
};

/** 36 by 20 px. Use it only for settings that apply immediately. */
function Switch({ className, onLabel, offLabel, ...props }: SwitchProps) {
  const control = (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "peer relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border border-control bg-muted transition-colors duration-(--vf-duration-fast) pointer-coarse:after:absolute pointer-coarse:after:-inset-3 pointer-coarse:after:content-[''] disabled:pointer-events-none disabled:border-border disabled:opacity-60 data-[state=checked]:border-brand data-[state=checked]:bg-brand",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="pointer-events-none block size-4 translate-x-0.5 rounded-full border border-control bg-background transition-transform duration-(--vf-duration-fast) data-[state=checked]:translate-x-4.5 rtl:-translate-x-0.5 rtl:data-[state=checked]:-translate-x-4.5"
      />
    </SwitchPrimitive.Root>
  );
  if (onLabel === undefined && offLabel === undefined) return control;
  return (
    <span className="inline-flex items-center gap-2">
      {control}
      <span aria-hidden="true" className="hidden text-caption text-muted-foreground peer-data-[state=checked]:inline">
        {onLabel}
      </span>
      <span aria-hidden="true" className="text-caption text-muted-foreground peer-data-[state=checked]:hidden">
        {offLabel}
      </span>
    </span>
  );
}

export { Switch };
