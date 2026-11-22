import type * as React from "react";
import { cn } from "../../lib/utils";
import { controlBox } from "./input";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(controlBox, "field-sizing-content min-h-20 py-2 text-sm pointer-coarse:text-base", className)}
      {...props}
    />
  );
}

export { Textarea };
