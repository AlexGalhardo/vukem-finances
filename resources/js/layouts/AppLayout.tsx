import { Link, usePage } from "@inertiajs/react";
import type { ReactNode } from "react";
import Mark from "../components/Mark";
import Preferences from "../components/Preferences";
import ThemeToggle from "../components/ThemeToggle";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Button } from "../components/ui/button";
import { TooltipProvider } from "../components/ui/tooltip";
import { useFinanceI18n } from "../lib/i18n";
import type { SharedProps } from "../page-props";
import AuthLayout from "./AuthLayout";
import Shell from "./Shell";

/** Chooses the frame: the panel's own layout, the signed-in Shell, or the public header. */
export default function AppLayout({ children }: { children: ReactNode }) {
  const page = usePage<SharedProps>();
  const { auth, workspace } = page.props;
  const { t } = useFinanceI18n();
  const success = page.flash?.success ?? page.flash?.status;
  const error = page.flash?.error;
  const notices = (
    <>
      {typeof success === "string" && (
        <Alert variant="success">
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      )}
      {typeof error === "string" && (
        <Alert variant="error">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
    </>
  );

  // The panel's own screens, Revenue included, bring AdminLayout (sidebar, title, theme): wrapping them here would
  // stack a second header.
  if (page.component.startsWith("Admin/")) return <>{children}</>;

  if (auth?.user && workspace) return <Shell notices={notices}>{children}</Shell>;

  const brand = (
    <Link href="/" className="flex min-w-0 items-center gap-2 text-h2">
      <Mark />
      <span className="truncate max-[359px]:sr-only">{page.props.appName}</span>
    </Link>
  );
  const landing = page.url.split("?")[0] === "/";
  if (!landing) {
    return (
      <TooltipProvider>
        <AuthLayout notices={notices}>{children}</AuthLayout>
      </TooltipProvider>
    );
  }
  return (
    <TooltipProvider>
      <header className="flex h-20 items-center justify-between border-b border-border px-5 md:px-12">
        {brand}
        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          <Preferences />
          <Button asChild variant="ghost" className="max-sm:hidden">
            <Link href="/login">{t("login")}</Link>
          </Button>
          <ThemeToggle />
        </div>
      </header>
      <main>
        {notices}
        {children}
      </main>
    </TooltipProvider>
  );
}
