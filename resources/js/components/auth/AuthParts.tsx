import { CircleAlert } from "lucide-react";
import type { ReactElement, ReactNode } from "react";
import { useFinanceI18n } from "../../lib/i18n";
import { Button } from "../ui/button";

/** The one h1 of an auth screen with its one-line description. */
export function AuthHeading({ title, description }: { title: string; description?: string }): ReactElement {
  return (
    <div className="flex flex-col gap-2">
      <h1 className="text-h1-auth text-balance">{title}</h1>
      {description && <p className="text-body text-pretty text-muted-foreground">{description}</p>}
    </div>
  );
}

/**
 * The error summary above a form. Fields carry their own messages; this line is announced at once (role alert) and
 * can take focus when no field is to blame. Nothing renders while there are no errors.
 */
export function AuthErrorSummary({ errors }: { errors: Record<string, string | undefined> }): ReactElement | null {
  const { t } = useFinanceI18n();
  if (Object.keys(errors).length === 0) return null;
  return (
    <p
      role="alert"
      tabIndex={-1}
      data-slot="auth-error-summary"
      className="flex items-start gap-2 rounded-md bg-expense-subtle px-3 py-2 text-body text-expense outline-none"
    >
      <CircleAlert aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      {t("authErrorSummary")}
    </p>
  );
}

/** The sign-in with Google link: an outline button with the Google mark. */
export function GoogleButton({ label }: { label: string }): ReactElement {
  return (
    <Button variant="outline" size="lg" asChild>
      <a href="/auth/google/redirect">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            fill="currentColor"
            d="M21.6 12.2c0-.7-.1-1.4-.2-2.1H12v4h5.4a4.6 4.6 0 0 1-2 3v2.5h3.3c1.9-1.7 2.9-4.2 2.9-7.4ZM12 22c2.7 0 5-.9 6.7-2.4l-3.3-2.5c-.9.6-2 1-3.4 1-2.6 0-4.8-1.8-5.6-4.2H3v2.6A10 10 0 0 0 12 22ZM6.4 13.9a6 6 0 0 1 0-3.8V7.5H3a10 10 0 0 0 0 9l3.4-2.6ZM12 5.9c1.5 0 2.8.5 3.9 1.5l2.9-2.9A9.6 9.6 0 0 0 12 2a10 10 0 0 0-9 5.5l3.4 2.6A6 6 0 0 1 12 5.9Z"
          />
        </svg>
        {label}
      </a>
    </Button>
  );
}

/** A quiet text link under a form (for example "Forgot your password?"). */
export const authLinkClass = "text-brand-text underline underline-offset-4 decoration-1 hover:decoration-2";

/** Wraps a form and anything beside it so the focus helper can find the summary. */
export function AuthSection({ children }: { children: ReactNode }): ReactElement {
  return <div className="flex flex-col gap-6">{children}</div>;
}
