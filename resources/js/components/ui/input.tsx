import { cva, type VariantProps } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../../lib/utils";

/** The box shared by Input, Textarea, NativeSelect and the Select trigger. */
const controlBox =
  "w-full min-w-0 rounded-md border border-control bg-background px-3 text-foreground transition-colors duration-(--vf-duration-fast) placeholder:text-subtle hover:border-muted-foreground focus-visible:border-ring aria-invalid:border-expense disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-muted disabled:text-disabled read-only:bg-muted";

const inputVariants = cva(
  `${controlBox} pointer-coarse:min-h-11 pointer-coarse:text-base file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground`,
  {
    variants: {
      inputSize: {
        sm: "h-8 text-sm",
        default: "h-9 text-sm",
        lg: "h-11 text-base",
      },
    },
    defaultVariants: { inputSize: "default" },
  },
);

type InputProps = Omit<React.ComponentProps<"input">, "size"> & VariantProps<typeof inputVariants>;

function Input({ className, type, inputSize, ...props }: InputProps) {
  return (
    <input
      type={type}
      data-slot="input"
      data-size={inputSize ?? "default"}
      className={cn(inputVariants({ inputSize }), className)}
      {...props}
    />
  );
}

export { controlBox, Input, inputVariants };
