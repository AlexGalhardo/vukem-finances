import type { ReactElement, ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface SettingsSectionProps {
  title: string;
  /** Why the section exists, in one or two lines. */
  description?: string;
  /** A HelpHint shown beside the title. */
  help?: ReactNode;
  /** The last section of a page for destructive actions: an expense-colored border. */
  danger?: boolean;
  /** The `data-tour` target of the tutorial step that points at this section. */
  tour?: string;
  children: ReactNode;
}

/**
 * One row of a settings page (section 7.7): the title and the reason on the left, the controls on the right, and a
 * hairline below. On a narrow screen the two stack. Each section saves on its own. It is deliberately not a named
 * region: a landmark labelled by its heading would also answer `getByLabel` for words the heading shares with a field.
 */
export default function SettingsSection({
  title,
  description,
  help,
  danger = false,
  tour,
  children,
}: SettingsSectionProps): ReactElement {
  return (
    <section
      data-tour={tour}
      className={cn(
        "grid gap-4 py-8 lg:grid-cols-[280px_minmax(0,1fr)] lg:gap-8",
        danger ? "rounded-lg border border-expense px-4 lg:px-6" : "border-b border-border last:border-b-0",
      )}
    >
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex items-center gap-1">
          <h2 className="text-h3">{title}</h2>
          {help}
        </div>
        {description && <p className="text-body text-pretty text-muted-foreground">{description}</p>}
      </div>
      <div className="flex min-w-0 flex-col gap-4">{children}</div>
    </section>
  );
}
