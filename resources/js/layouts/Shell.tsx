import { Link, router, usePage } from "@inertiajs/react";
import { ArrowUpRight, Keyboard, LogOut, Plus, Search, Settings, ShieldCheck } from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import CommandPalette from "../components/CommandPalette";
import { Shortcut } from "../components/Kbd";
import NotificationBell from "../components/NotificationBell";
import Preferences from "../components/Preferences";
import ShortcutHelp from "../components/ShortcutHelp";
import QuickAdd, { useQuickAdd } from "../components/transactions/QuickAdd";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "../components/ui/sidebar";
import { TooltipProvider } from "../components/ui/tooltip";
import { useFinanceI18n } from "../lib/i18n";
import { isActivePath, navItemsFor } from "../nav";
import type { SharedProps } from "../page-props";
import ShellKeyboard from "../shortcuts/ShellKeyboard";
import { useIsMac } from "../shortcuts/useIsMac";

/** On a new path, moves focus to the page h1 (or main) and announces the title in a polite live region. */
function useRouteFocus(): string {
  const [announcement, setAnnouncement] = useState("");
  const last = useRef(typeof location === "undefined" ? "" : location.pathname);
  useEffect(
    () =>
      router.on("navigate", (event) => {
        const path = event.detail.page.url.split("?")[0] ?? "";
        if (path === last.current) return;
        last.current = path;
        // Wait for the new page to render before looking for its heading.
        requestAnimationFrame(() => {
          const main = document.getElementById("content");
          const heading = main?.querySelector<HTMLElement>("h1");
          if (heading && !heading.hasAttribute("tabindex")) heading.tabIndex = -1;
          const target = heading ?? main;
          target?.focus({ preventScroll: false });
          setAnnouncement(heading?.textContent ?? document.title);
        });
      }),
    [],
  );
  return announcement;
}

/** Header button of the new transaction dialog; viewers cannot record, so they do not get it. */
function NewTransactionButton() {
  const { t } = useFinanceI18n();
  const { openTransaction, canRecord } = useQuickAdd();
  if (!canRecord) return null;
  return (
    <Button type="button" aria-haspopup="dialog" onClick={() => openTransaction("expense")}>
      <Plus aria-hidden="true" />
      <span className="hidden sm:inline">{t("newTransaction")}</span>
      <span className="sm:hidden sr-only">{t("newTransaction")}</span>
    </Button>
  );
}

export default function Shell({ children, notices }: { children: ReactNode; notices: ReactNode }) {
  const page = usePage<SharedProps>();
  const { auth, workspace } = page.props;
  const { t } = useFinanceI18n();
  const isMac = useIsMac();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const announcement = useRouteFocus();

  // A dialog left open would cover the page the person just asked for.
  useEffect(
    () =>
      router.on("navigate", () => {
        setPaletteOpen(false);
        setHelpOpen(false);
      }),
    [],
  );

  if (!auth?.user || !workspace) return null;
  const saas = workspace.edition !== "personal";
  const items = navItemsFor(workspace.edition);

  return (
    <TooltipProvider>
      <a
        href="#content"
        className="fixed top-2 left-2 z-[70] -translate-y-20 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground focus:translate-y-0"
      >
        {t("skipToContent")}
      </a>
      <SidebarProvider>
        <QuickAdd>
          <ShellKeyboard
            items={items}
            onOpenPalette={() => {
              setHelpOpen(false);
              setPaletteOpen(true);
            }}
            onOpenHelp={() => {
              setPaletteOpen(false);
              setHelpOpen(true);
            }}
          />
          <Sidebar>
            <SidebarHeader className="gap-6 p-5">
              <Link href="/dashboard" className="flex items-center gap-2 text-lg font-semibold tracking-tight">
                <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <ArrowUpRight aria-hidden="true" className="size-5" />
                </span>
                {page.props.appName}
              </Link>
              <div className="flex flex-col gap-2">
                {saas && (
                  <NativeSelect
                    aria-label={t("workspace")}
                    value={workspace.current.id}
                    onChange={(e) => router.post(`/workspaces/${e.target.value}/switch`)}
                  >
                    {workspace.all.map((w) => (
                      <NativeSelectOption key={w.id} value={w.id}>
                        {w.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                )}
                <div>
                  <Badge variant="secondary">{workspace.plan}</Badge>
                </div>
              </div>
            </SidebarHeader>
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupLabel>{t("personalFinance")}</SidebarGroupLabel>
                <SidebarGroupContent>
                  <nav aria-label={t("mainNavigation")}>
                    <SidebarMenu>
                      {items.map((item) => {
                        const active = isActivePath(page.url, item.href);
                        return (
                          <SidebarMenuItem key={item.id}>
                            <SidebarMenuButton asChild isActive={active} tooltip={t(item.labelKey)}>
                              <Link href={item.href} aria-current={active ? "page" : undefined}>
                                <item.icon aria-hidden="true" />
                                <span>{t(item.labelKey)}</span>
                              </Link>
                            </SidebarMenuButton>
                          </SidebarMenuItem>
                        );
                      })}
                    </SidebarMenu>
                  </nav>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
            <SidebarFooter className="p-4">
              <SidebarMenu>
                {workspace.admin && (
                  <SidebarMenuItem>
                    <SidebarMenuButton asChild tooltip={t("admin")}>
                      <Link href="/admin">
                        <ShieldCheck aria-hidden="true" />
                        <span>{t("admin")}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )}
                <SidebarMenuItem>
                  <SidebarMenuButton tooltip={t("openShortcutHelp")} onClick={() => setHelpOpen(true)}>
                    <Keyboard aria-hidden="true" />
                    <span>{t("openShortcutHelp")}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild tooltip={auth.user.name}>
                    <Link href="/user/two-factor">
                      <Settings aria-hidden="true" />
                      <span>{auth.user.name}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild tooltip={t("logout")}>
                    <Link href="/logout" method="post" as="button">
                      <LogOut aria-hidden="true" />
                      <span>{t("logout")}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarFooter>
          </Sidebar>
          <div className="relative flex min-w-0 flex-1 flex-col bg-background">
            <header className="sticky top-0 z-10 flex min-h-14 flex-wrap items-center gap-2 border-b bg-background px-4 py-2 lg:px-6">
              <SidebarTrigger aria-label={t("toggleSidebar")} />
              <span className="truncate text-sm text-muted-foreground">{workspace.current.name}</span>
              <div className="ml-auto flex flex-wrap items-center gap-2">
                <NewTransactionButton />
                <Button
                  type="button"
                  variant="outline"
                  aria-haspopup="dialog"
                  aria-label={t("openPalette")}
                  onClick={() => setPaletteOpen(true)}
                >
                  <Search aria-hidden="true" />
                  <span className="hidden sm:inline">{t("openPalette")}</span>
                  <span className="hidden sm:inline" aria-hidden="true">
                    <Shortcut keys="mod+k" isMac={isMac} />
                  </span>
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-haspopup="dialog"
                  aria-label={t("openShortcutHelp")}
                  title={t("openShortcutHelp")}
                  onClick={() => setHelpOpen(true)}
                >
                  <Keyboard aria-hidden="true" />
                </Button>
                <NotificationBell />
                <Preferences withShortcuts />
              </div>
            </header>
            <main id="content" tabIndex={-1} className="outline-none">
              <div className="finance-page mx-auto flex w-full max-w-7xl flex-col gap-7 p-5 lg:p-8">
                {notices}
                {children}
              </div>
            </main>
          </div>
          <div role="status" aria-live="polite" className="sr-only">
            {announcement}
          </div>
          <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
          <ShortcutHelp open={helpOpen} onOpenChange={setHelpOpen} />
        </QuickAdd>
      </SidebarProvider>
    </TooltipProvider>
  );
}
