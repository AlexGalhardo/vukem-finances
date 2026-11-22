import { Head, Link, usePage } from "@inertiajs/react";
import { CircleHelp } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import Mark from "../components/Mark";
import Preferences from "../components/Preferences";
import ThemeToggle from "../components/ThemeToggle";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Button } from "../components/ui/button";
import { TooltipProvider } from "../components/ui/tooltip";
import { screenIdForPath } from "../help/screens";
import { type DashboardData, getJSON, type NavItem } from "../lib/adminApi";
import { useFinanceI18n } from "../lib/i18n";
import { helpAction } from "../tutorial/state";
import TutorialHost from "../tutorial/TutorialHost";
import { TutorialProvider, useTutorial } from "../tutorial/useTutorial";

const navLink =
  "flex min-h-9 items-center rounded-md px-3 text-body whitespace-nowrap text-muted-foreground transition-colors duration-(--vf-duration-fast) hover:bg-muted hover:text-foreground aria-[current=page]:bg-muted aria-[current=page]:text-foreground pointer-coarse:min-h-11";

/** Starts or resumes the tutorial of the panel screen the person is on; the `h t` key does the same. */
function TutorialButton() {
  const { t } = useFinanceI18n();
  const tutorial = useTutorial();
  const url = usePage().url;
  const action = helpAction(tutorial.state);
  if (action === "none") return null;
  return (
    <Button
      type="button"
      variant="ghost"
      shortcutAction="open-tutorial"
      onClick={() => (action === "resume" ? tutorial.resume() : tutorial.start(screenIdForPath(url)))}
    >
      <CircleHelp aria-hidden="true" data-icon="inline-start" />
      {action === "resume" ? t("tutorialResume") : t("tutorialStart")}
    </Button>
  );
}

/** Sidebar dashboard shell: the resource list comes from the panel's JSON dashboard. */
export default function AdminLayout({ children, title }: { children: ReactNode; title: string }) {
  const page = usePage<{ dataUrl?: string; edition?: string }>();
  // Revenue is not a panel page and gets no `dataUrl`; the panel is always mounted at /admin-api (app/admin.DataPrefix).
  const dataUrl = page.props.dataUrl ?? "/admin-api";
  const { t } = useFinanceI18n();
  const [nav, setNav] = useState<NavItem[]>([]);
  const [name, setName] = useState(t("admPanel"));
  const [extra, setExtra] = useState<NavItem[]>([]);
  const path = page.url.split("?")[0] ?? "";
  const success = page.flash?.success ?? page.flash?.status;
  const failure = page.flash?.error;

  useEffect(() => {
    const ctl = new AbortController();
    getJSON<DashboardData>(`${dataUrl}?counts=0`, ctl.signal)
      .then((d) => {
        setNav((d.panel.nav ?? []).filter((item) => item.kind === "resource"));
        // Custom pages, the audit log and the flags page are HTML pages of the panel: plain links, no React screen.
        const html: NavItem[] = (d.panel.nav ?? []).filter((item) => item.kind === "page");
        if (d.panel.auditUrl)
          html.push({ label: t("admAuditLog"), url: d.panel.auditUrl, kind: "page", active: false });
        if (d.panel.flagsUrl)
          html.push({ label: t("admFeatureFlags"), url: d.panel.flagsUrl, kind: "page", active: false });
        setExtra(html);
        setName(d.panel.title);
      })
      .catch(() => {});
    return () => ctl.abort();
  }, [dataUrl, t]);

  const hrefOf = (item: NavItem) => `/admin/${item.url.replace(`${dataUrl}/`, "")}`;
  const links = (
    <>
      <Link href="/admin" aria-current={path === "/admin" ? "page" : undefined} className={navLink}>
        {t("admDashboard")}
      </Link>
      {nav.map((item) => (
        <Link
          key={item.url}
          href={hrefOf(item)}
          aria-current={path === hrefOf(item) || path.startsWith(`${hrefOf(item)}/`) ? "page" : undefined}
          className={navLink}
        >
          {item.label}
        </Link>
      ))}
      {page.props.edition !== "personal" && (
        <Link href="/admin/revenue" aria-current={path === "/admin/revenue" ? "page" : undefined} className={navLink}>
          {t("revenueTitle")}
        </Link>
      )}
      {extra.map((item) => (
        <a key={item.url} href={item.url} className={navLink}>
          {item.label}
        </a>
      ))}
    </>
  );

  return (
    <TooltipProvider>
      <TutorialProvider>
        <div className="min-h-svh bg-background text-foreground">
          <Head title={`${title} · ${name}`} />
          <aside className="flex flex-col gap-6 overflow-x-auto border-b border-border px-4 py-2 lg:fixed lg:inset-y-0 lg:start-0 lg:w-60 lg:overflow-x-visible lg:border-e lg:border-b-0 lg:p-4">
            <Link href="/admin" className="hidden items-center gap-2 text-h2 lg:flex">
              <Mark />
              <span className="truncate">{name}</span>
            </Link>
            <nav aria-label={t("admNavLabel")} data-tour="admin-nav" className="flex gap-1 lg:flex-col">
              {links}
            </nav>
          </aside>
          <div className="min-w-0 lg:ps-60">
            <header className="flex min-h-14 flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border px-4 py-2 lg:px-8">
              <h1 tabIndex={-1} className="min-w-0 text-h2 outline-none">
                {title}
              </h1>
              <div className="flex flex-wrap items-center gap-2">
                <TutorialButton />
                <Button asChild variant="ghost">
                  <Link href="/dashboard">{t("admBackToApp")}</Link>
                </Button>
                <Preferences />
                <ThemeToggle />
              </div>
            </header>
            <main className="flex min-w-0 flex-col gap-6 p-4 lg:p-8">
              {typeof success === "string" && (
                <Alert variant="success">
                  <AlertDescription>{success}</AlertDescription>
                </Alert>
              )}
              {typeof failure === "string" && (
                <Alert variant="error">
                  <AlertDescription>{failure}</AlertDescription>
                </Alert>
              )}
              {children}
            </main>
          </div>
          <TutorialHost />
        </div>
      </TutorialProvider>
    </TooltipProvider>
  );
}
