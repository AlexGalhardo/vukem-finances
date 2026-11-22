import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import type * as React from "react";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "group/badge inline-flex h-5 w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-md border border-transparent px-2 text-xs font-medium whitespace-nowrap transition-colors duration-(--vf-duration-fast) [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      variant: {
        neutral: "bg-muted text-foreground",
        outline: "border-border text-muted-foreground [a]:hover:bg-muted",
        success: "bg-brand-subtle text-brand-text",
        warning: "bg-warning-subtle text-warning",
        danger: "bg-expense-subtle text-expense",
        info: "bg-info-subtle text-info",
        // Names from before the design system, kept so existing call sites compile unchanged.
        default: "bg-primary text-primary-foreground [a]:hover:bg-primary/88",
        secondary: "bg-muted text-foreground [a]:hover:bg-border",
        destructive: "bg-expense-subtle text-expense",
        ghost: "text-muted-foreground hover:bg-muted",
        link: "text-brand-text underline underline-offset-4 hover:decoration-2",
      },
    },
    defaultVariants: {
      variant: "neutral",
    },
  },
);

function Badge({
  className,
  variant = "neutral",
  asChild = false,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span";

  return (
    <Comp data-slot="badge" data-variant={variant} className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
