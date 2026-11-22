import { Bell } from "lucide-react";

import { Button } from "./ui/button";

/** The bell button. Its accessible name carries the unread count; the visible badge is hidden from assistive tech so the count is not read twice. */
export function BellButton({ unread, label, ...rest }: { unread: number; label: string } & Record<string, unknown>) {
  return (
    <Button type="button" variant="ghost" size="icon" aria-label={label} className="relative" {...rest}>
      <Bell aria-hidden="true" />
      {unread > 0 && (
        <span
          aria-hidden="true"
          className="absolute -top-0.5 -right-0.5 flex min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground"
        >
          {unread > 99 ? "99+" : unread}
        </span>
      )}
    </Button>
  );
}
