import type { ReactNode } from "react";

export interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  /** A HelpHint for the page, shown beside the title. */
  help?: ReactNode;
}

/** The page's one h1. The shell moves focus here after a navigation, so it must stay focusable by script. */
export default function PageHeader({ title, description, actions, help }: PageHeaderProps) {
  return (
    <header className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex items-center gap-2">
          <h1 tabIndex={-1} className="text-h1 outline-none">
            {title}
          </h1>
          {help}
        </div>
        {description && <p className="max-w-[65ch] text-body text-pretty text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}
