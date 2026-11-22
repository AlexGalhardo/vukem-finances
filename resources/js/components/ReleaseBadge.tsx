import { usePage } from "@inertiajs/react";
import type { ReactElement } from "react";
import type { SharedProps } from "../page-props";

/** The running release, in the bottom right corner. It never takes a click from what is under it. */
export default function ReleaseBadge(): ReactElement | null {
  const { release } = usePage<SharedProps>().props;
  if (!release) return null;
  return (
    <output
      data-print="hide"
      data-testid="release-version"
      className="pointer-events-none fixed right-[max(0.5rem,env(safe-area-inset-right))] bottom-[max(0.25rem,env(safe-area-inset-bottom))] z-40 rounded-sm bg-background/90 px-1.5 py-0.5 font-mono text-xs text-muted-foreground"
    >
      v{release}
    </output>
  );
}
