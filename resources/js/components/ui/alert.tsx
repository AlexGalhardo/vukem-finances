import { cva, type VariantProps } from "class-variance-authority";
import { CircleAlertIcon, CircleCheckIcon, InfoIcon, TriangleAlertIcon } from "lucide-react";
import type * as React from "react";
import { cn } from "../../lib/utils";

const alertVariants = cva(
  "relative grid w-full grid-cols-[auto_1fr] items-start gap-x-3 gap-y-1 rounded-lg border border-transparent px-4 py-3 text-body has-data-[slot=alert-action]:grid-cols-[auto_1fr_auto]",
  {
    variants: {
      variant: {
        info: "bg-info-subtle text-info",
        success: "bg-brand-subtle text-brand-text",
        warning: "bg-warning-subtle text-warning",
        error: "bg-expense-subtle text-expense",
      },
    },
    defaultVariants: { variant: "info" },
  },
);

const icons = {
  info: InfoIcon,
  success: CircleCheckIcon,
  warning: TriangleAlertIcon,
  error: CircleAlertIcon,
} as const;

/** An inline banner. Errors are announced at once (role alert), the rest politely (role status). */
function Alert({
  className,
  variant = "info",
  children,
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof alertVariants>) {
  const kind = variant ?? "info";
  const Icon = icons[kind];
  return (
    <div
      data-slot="alert"
      data-variant={kind}
      role={kind === "error" ? "alert" : "status"}
      className={cn(alertVariants({ variant: kind }), className)}
      {...props}
    >
      <Icon aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <div className="col-start-2 grid gap-1">{children}</div>
    </div>
  );
}

function AlertTitle({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="alert-title" className={cn("text-label", className)} {...props} />;
}

function AlertDescription({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="alert-description" className={cn("text-foreground", className)} {...props} />;
}

function AlertAction({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="alert-action" className={cn("col-start-3 row-start-1 self-center", className)} {...props} />;
}

export { Alert, AlertAction, AlertDescription, AlertTitle };
