import type { ReactElement, SVGProps } from "react";
import { cn } from "../lib/utils";

/**
 * The product mark: a brand-green square with a "V". It replaces the arrow, which already means an expense.
 * Decorative: the product name is always written next to it.
 */
export default function Mark({ className, ...props }: SVGProps<SVGSVGElement>): ReactElement {
  return (
    <svg aria-hidden="true" viewBox="0 0 32 32" className={cn("size-8 shrink-0", className)} fill="none" {...props}>
      <rect width="32" height="32" rx="8" className="fill-brand" />
      <path
        d="M9 11l7 11 7-11"
        className="stroke-brand-foreground"
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
