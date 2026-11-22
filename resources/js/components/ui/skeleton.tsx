import { cn } from "../../lib/utils";

/** Static under reduced motion; the pulse is opted in by the vf-pulse rule of finance.css. */
function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="skeleton" className={cn("vf-pulse rounded-md bg-muted", className)} {...props} />;
}

export { Skeleton };
