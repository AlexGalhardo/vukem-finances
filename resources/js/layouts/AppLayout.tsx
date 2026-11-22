import { Link, usePage } from "@inertiajs/react";
import { ArrowUpRight } from "lucide-react";
import type { ReactNode } from "react";
import Preferences from "../components/Preferences";
import ThemeToggle from "../components/ThemeToggle";
import { Button } from "../components/ui/button";
import { TooltipProvider } from "../components/ui/tooltip";
import { useFinanceI18n } from "../lib/i18n";
import type { SharedProps } from "../page-props";
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
        <p role="status" className="rounded-lg border bg-muted p-4 text-sm">
          {success}
        </p>
      )}
      {typeof error === "string" && (
        <p role="alert" className="rounded-lg border border-destructive p-4 text-sm text-destructive">
          {error}
        </p>
      )}
    </>
  );

  // The panel's own screens bring AdminLayout (sidebar, title, sign-out): wrapping them here would stack a second header.
  if (page.component.startsWith("Admin/") && page.component !== "Admin/Revenue") return <>{children}</>;

  if (auth?.user && workspace) return <Shell notices={notices}>{children}</Shell>;

  const brand = (
    <Link href="/" className="flex items-center gap-2 text-lg font-semibold tracking-tight">
      <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <ArrowUpRight aria-hidden="true" className="size-5" />
      </span>
      {page.props.appName}
    </Link>
  );
  const landing = page.url.split("?")[0] === "/";
  return (
    <TooltipProvider>
      <header className="flex h-20 items-center justify-between border-b px-5 md:px-12">
        {brand}
        <div className="flex items-center gap-3">
          <Preferences />
          <Button asChild variant="ghost">
            <Link href="/login">{t("login")}</Link>
          </Button>
          <ThemeToggle />
        </div>
      </header>
      {landing ? (
        <main>
          {notices}
          {children}
        </main>
      ) : (
        <main className="mx-auto flex min-h-[calc(100svh-5rem)] max-w-md flex-col justify-center gap-6 px-5 py-12">
          {notices}
          {children}
        </main>
      )}
    </TooltipProvider>
  );
}
