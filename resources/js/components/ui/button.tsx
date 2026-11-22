import { cva, type VariantProps } from "class-variance-authority";
import { LoaderCircleIcon } from "lucide-react";
import { Slot } from "radix-ui";
import type * as React from "react";
import { cn } from "../../lib/utils";
import { type ActionHint, useActionHint } from "../../shortcuts/useActionHint";
import { ShortcutHint } from "../Kbd";

const buttonVariants = cva(
  "group/button inline-flex shrink-0 pointer-coarse:min-h-11 pointer-coarse:min-w-11 items-center justify-center gap-2 rounded-md border border-transparent text-sm font-medium whitespace-nowrap transition-colors duration-(--vf-duration-fast) select-none disabled:pointer-events-none disabled:text-disabled aria-disabled:pointer-events-none aria-disabled:text-disabled aria-invalid:border-expense [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground hover:bg-primary/88 active:bg-primary/80 disabled:bg-muted aria-disabled:bg-muted",
        brand:
          "bg-brand text-brand-foreground hover:bg-brand-hover active:bg-brand-active disabled:bg-muted aria-disabled:bg-muted",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-[color-mix(in_srgb,var(--secondary),var(--foreground)_6%)] active:bg-[color-mix(in_srgb,var(--secondary),var(--foreground)_10%)] aria-expanded:bg-[color-mix(in_srgb,var(--secondary),var(--foreground)_10%)] disabled:bg-muted aria-disabled:bg-muted",
        outline:
          "border-control bg-background text-foreground hover:bg-muted active:bg-muted aria-expanded:bg-muted disabled:bg-muted aria-disabled:bg-muted",
        ghost: "text-foreground hover:bg-muted active:bg-muted aria-expanded:bg-muted",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90 active:bg-destructive/80 disabled:bg-muted aria-disabled:bg-muted",
        "destructive-outline":
          "border-expense bg-transparent text-expense hover:bg-expense-subtle active:bg-expense-subtle disabled:border-border aria-disabled:border-border",
        link: "text-brand-text underline underline-offset-4 decoration-1 hover:decoration-2",
      },
      size: {
        default: "h-9 px-3.5 text-sm",
        xs: "h-6 px-2 text-xs [&_svg:not([class*='size-'])]:size-3",
        sm: "h-8 px-3 text-caption [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-11 px-4.5 text-base [&_svg:not([class*='size-'])]:size-4.5",
        icon: "size-9 px-0",
        "icon-xs": "size-6 px-0 [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-8 px-0 [&_svg:not([class*='size-'])]:size-3.5",
        "icon-lg": "size-11 px-0 [&_svg:not([class*='size-'])]:size-4.5",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

type ButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    /** Keeps the width, sets aria-busy, disables the button and shows a spinner beside the label. */
    loading?: boolean;
    /** The id of a registered action: its shortcut shows at the end of the label and in aria-keyshortcuts. */
    shortcutAction?: string;
  };

function PlainButton({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  loading = false,
  disabled,
  children,
  hint,
  ...props
}: Omit<ButtonProps, "shortcutAction"> & { hint?: ActionHint | null }) {
  const Comp = asChild ? Slot.Root : "button";

  if (asChild) {
    return (
      <Comp
        data-slot="button"
        data-variant={variant}
        data-size={size}
        className={cn(buttonVariants({ variant, size }), className)}
        aria-keyshortcuts={hint?.ariaKeyShortcuts}
        {...props}
      >
        {children}
      </Comp>
    );
  }

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      aria-busy={loading || undefined}
      disabled={disabled || loading}
      className={cn(buttonVariants({ variant, size }), className)}
      aria-keyshortcuts={hint?.ariaKeyShortcuts}
      {...props}
    >
      {loading ? <LoaderCircleIcon aria-hidden="true" className="motion-safe:animate-spin" /> : null}
      {children}
      {hint && !asChild ? <ShortcutHint parts={hint.parts} /> : null}
    </Comp>
  );
}

function HintedButton({ shortcutAction, ...props }: ButtonProps & { shortcutAction: string }) {
  const hint = useActionHint(shortcutAction);
  return <PlainButton {...props} hint={hint} />;
}

function Button({ shortcutAction, ...props }: ButtonProps) {
  // Only a button that names an action reads the registry, so the many plain buttons never re-render with it.
  if (shortcutAction !== undefined) return <HintedButton shortcutAction={shortcutAction} {...props} />;
  return <PlainButton {...props} />;
}

export { Button, buttonVariants };
