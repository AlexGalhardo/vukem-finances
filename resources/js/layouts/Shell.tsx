import { Link, router, usePage } from "@inertiajs/react";
import {
  BookOpen,
  ChevronRight,
  CircleHelp,
  CircleUser,
  GraduationCap,
  Keyboard,
  LogOut,
  Plus,
  Search,
  Settings,
  ShieldCheck,
} from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import AssistantHost from "../components/assistant/AssistantHost";
import CommandPalette from "../components/CommandPalette";
import { Kbd, ShortcutHint } from "../components/Kbd";
import Mark from "../components/Mark";
import NotificationBell from "../components/NotificationBell";
import Preferences from "../components/Preferences";
import ShortcutHelp from "../components/ShortcutHelp";
import ThemeToggle from "../components/ThemeToggle";
import QuickAdd, { useQuickAdd } from "../components/transactions/QuickAdd";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../components/ui/tooltip";
import { useFinanceI18n } from "../lib/i18n";
import { isActivePath, type NavItem, navGroups, navHintActionId, navItemsFor } from "../nav";
import type { SharedProps } from "../page-props";
import ShellKeyboard from "../shortcuts/ShellKeyboard";
import { useActionHint } from "../shortcuts/useActionHint";
import PageHelp from "../tutorial/PageHelp";
import { helpAction } from "../tutorial/state";
import TutorialHost from "../tutorial/TutorialHost";
import { TutorialProvider, useTutorial } from "../tutorial/useTutorial";
import WelcomeCard from "../tutorial/WelcomeCard";

const DOCS_URL = "https://finances.vukem.dev/docs/";

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

/** The key caps of an action inside a menu item, read from the registry. */
function ActionKeys({ actionId }: { actionId: string }) {
  const hint = useActionHint(actionId);
  if (!hint) return null;
  return (
    <DropdownMenuShortcut aria-hidden="true" className="inline-flex gap-0.5">
      {hint.parts.map((part, index) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: the same cap can repeat in a sequence
        <Kbd key={index}>{part}</Kbd>
      ))}
    </DropdownMenuShortcut>
  );
}

/** An icon-only header button: its name and shortcut show in a tooltip, never in a native title. */
function HeaderIconButton({
  label,
  actionId,
  onClick,
  children,
}: {
  label: string;
  actionId?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  const hint = useActionHint(actionId ?? "");
  const { t } = useFinanceI18n();
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-haspopup="dialog"
          aria-label={label}
          aria-keyshortcuts={hint?.ariaKeyShortcuts}
          onClick={onClick}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {label}
        {hint?.parts.map((part, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: the same cap can repeat in a sequence
          <span key={index} className="inline-flex items-center gap-1">
            {index > 0 && <span>{t("shortcutThen")}</span>}
            <Kbd>{part}</Kbd>
          </span>
        ))}
      </TooltipContent>
    </Tooltip>
  );
}

/** One sidebar link: active bar and aria-current, and in the collapsed rail a tooltip with its shortcut. */
function NavEntry({ item, active }: { item: NavItem; active: boolean }) {
  const { t } = useFinanceI18n();
  const hint = useActionHint(navHintActionId(item));
  const label = t(item.labelKey);
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        isActive={active}
        tooltip={{
          children: (
            <>
              {label}
              {hint?.parts.map((part, index) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: the same cap can repeat in a sequence
                <Kbd key={index}>{part}</Kbd>
              ))}
            </>
          ),
        }}
        className="relative data-[active=true]:font-medium data-[active=true]:before:absolute data-[active=true]:before:inset-y-1.5 data-[active=true]:before:start-0 data-[active=true]:before:w-0.5 data-[active=true]:before:rounded-full data-[active=true]:before:bg-brand"
      >
        <Link href={item.href} aria-current={active ? "page" : undefined}>
          <item.icon aria-hidden="true" />
          <span>{label}</span>
          {hint && (
            <ShortcutHint
              parts={hint.parts}
              className="ms-auto opacity-0 group-focus-within/menu-item:opacity-100 group-hover/menu-item:opacity-100 group-data-[collapsible=icon]:hidden!"
            />
          )}
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

/** Header button of the new transaction dialog; viewers cannot record, so they do not get it. */
function NewTransactionButton() {
  const { t } = useFinanceI18n();
  const { openTransaction, canRecord } = useQuickAdd();
  if (!canRecord) return null;
  return (
    <Button
      type="button"
      variant="brand"
      aria-haspopup="dialog"
      shortcutAction="new-expense"
      onClick={() => openTransaction("expense")}
    >
      <Plus aria-hidden="true" />
      <span className="hidden sm:inline">{t("newTransaction")}</span>
      <span className="sm:hidden sr-only">{t("newTransaction")}</span>
    </Button>
  );
}

function ShellFrame({ children, notices }: { children: ReactNode; notices: ReactNode }) {
  const page = usePage<SharedProps>();
  const { auth, workspace } = page.props;
  const { t } = useFinanceI18n();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [pageHelpOpen, setPageHelpOpen] = useState(false);
  const announcement = useRouteFocus();
  const tutorial = useTutorial();
  const tutorialAction = helpAction(tutorial.state);

  // A dialog left open would cover the page the person just asked for.
  useEffect(
    () =>
      router.on("navigate", () => {
        setPaletteOpen(false);
        setHelpOpen(false);
        setPageHelpOpen(false);
      }),
    [],
  );

  if (!auth?.user || !workspace) return null;
  const saas = workspace.edition !== "personal";
  const items = navItemsFor(workspace.edition);
  const current = items.find((item) => isActivePath(page.url, item.href));

  return (
    <TooltipProvider>
      <a
        data-print="hide"
        href="#content"
        className="fixed top-2 left-2 z-(--vf-z-skip) -translate-y-20 rounded-md bg-primary px-4 py-2 text-label text-primary-foreground focus:translate-y-0"
      >
        {t("skipToContent")}
      </a>
      <SidebarProvider>
        <QuickAdd>
          <AssistantHost />
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
            onOpenPageHelp={() => {
              setPaletteOpen(false);
              setPageHelpOpen(true);
            }}
          />
          <Sidebar>
            <SidebarHeader className="gap-4 p-4">
              <Link href="/dashboard" className="flex items-center gap-2 text-h2">
                <Mark />
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
              <nav aria-label={t("mainNavigation")} className="flex flex-col gap-2">
                <SidebarGroup>
                  <SidebarGroupContent>
                    <SidebarMenu>
                      {items
                        .filter((item) => item.group === undefined)
                        .map((item) => (
                          <NavEntry key={item.id} item={item} active={isActivePath(page.url, item.href)} />
                        ))}
                    </SidebarMenu>
                  </SidebarGroupContent>
                </SidebarGroup>
                {navGroups.map((group) => {
                  const members = items.filter((item) => item.group === group.id);
                  if (members.length === 0) return null;
                  return (
                    <SidebarGroup key={group.id}>
                      <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
                      <SidebarGroupContent>
                        <SidebarMenu>
                          {members.map((item) => (
                            <NavEntry key={item.id} item={item} active={isActivePath(page.url, item.href)} />
                          ))}
                        </SidebarMenu>
                      </SidebarGroupContent>
                    </SidebarGroup>
                  );
                })}
              </nav>
            </SidebarContent>
            <SidebarFooter className="p-4">
              <SidebarMenu>
                <SidebarMenuItem>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <SidebarMenuButton tooltip={t("myAccount")}>
                        <CircleUser aria-hidden="true" />
                        <span>{t("myAccount")}</span>
                      </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent side="top" align="start">
                      <p className="truncate px-2 py-1.5 text-xs text-muted-foreground">{auth.user.name}</p>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem asChild>
                        <Link href="/user/two-factor">
                          <Settings aria-hidden="true" />
                          {t("accountSettings")}
                        </Link>
                      </DropdownMenuItem>
                      {workspace.admin && (
                        <DropdownMenuItem asChild>
                          <Link href="/admin">
                            <ShieldCheck aria-hidden="true" />
                            {t("admin")}
                          </Link>
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onSelect={() => router.post("/logout")}>
                        <LogOut aria-hidden="true" />
                        {t("logout")}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild tooltip={t("documentation")}>
                    <a href={DOCS_URL} target="_blank" rel="noreferrer">
                      <BookOpen aria-hidden="true" />
                      <span>{t("documentation")}</span>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarFooter>
          </Sidebar>
          <div className="relative flex min-w-0 flex-1 flex-col bg-background">
            <header
              data-print="hide"
              className="sticky top-0 z-(--vf-z-header) flex min-h-14 flex-wrap items-center gap-2 border-b border-border bg-background px-4 py-2 lg:px-6"
            >
              <SidebarTrigger aria-label={t("toggleSidebar")} />
              <nav aria-label={t("breadcrumb")} className="min-w-0">
                <ol className="flex min-w-0 items-center gap-1.5 text-body text-muted-foreground">
                  <li className="truncate">{workspace.current.name}</li>
                  {current && (
                    <li className="flex items-center gap-1.5 text-foreground">
                      <ChevronRight aria-hidden="true" className="size-3.5 text-subtle" />
                      <span aria-current="page">{t(current.labelKey)}</span>
                    </li>
                  )}
                </ol>
              </nav>
              <div className="ml-auto flex flex-wrap items-center gap-2">
                <NewTransactionButton />
                <Button
                  type="button"
                  variant="outline"
                  aria-haspopup="dialog"
                  aria-label={t("openPalette")}
                  shortcutAction="open-palette"
                  onClick={() => setPaletteOpen(true)}
                >
                  <Search aria-hidden="true" />
                  <span className="hidden sm:inline">{t("openPalette")}</span>
                </Button>
                <HeaderIconButton
                  label={t("openShortcutHelp")}
                  actionId="open-shortcut-help"
                  onClick={() => setHelpOpen(true)}
                >
                  <Keyboard aria-hidden="true" />
                </HeaderIconButton>
                <DropdownMenu>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <DropdownMenuTrigger asChild>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t("helpMenu")}
                          data-tour="help-button"
                        >
                          <CircleHelp aria-hidden="true" />
                        </Button>
                      </DropdownMenuTrigger>
                    </TooltipTrigger>
                    <TooltipContent>{t("helpMenu")}</TooltipContent>
                  </Tooltip>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={() => setPageHelpOpen(true)}>
                      <CircleHelp aria-hidden="true" />
                      {t("helpThisPage")}
                      <ActionKeys actionId="open-page-help" />
                    </DropdownMenuItem>
                    {tutorialAction !== "none" && (
                      <DropdownMenuItem
                        onSelect={() => (tutorialAction === "resume" ? tutorial.resume() : tutorial.start())}
                      >
                        <GraduationCap aria-hidden="true" />
                        {tutorialAction === "resume" ? t("tutorialResume") : t("tutorialStart")}
                        <ActionKeys actionId="open-tutorial" />
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
                <NotificationBell />
                <Preferences withShortcuts />
                <ThemeToggle />
              </div>
            </header>
            <main id="content" tabIndex={-1} className="outline-none">
              <div className="mx-auto flex w-full max-w-[1200px] flex-col gap-8 px-4 py-6 md:px-8 md:py-8">
                {notices}
                <WelcomeCard />
                {children}
              </div>
            </main>
          </div>
          <div role="status" aria-live="polite" className="sr-only">
            {announcement}
          </div>
          <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
          <ShortcutHelp open={helpOpen} onOpenChange={setHelpOpen} />
          <PageHelp
            open={pageHelpOpen}
            onOpenChange={setPageHelpOpen}
            fallbackTitle={current ? t(current.labelKey) : workspace.current.name}
          />
          <TutorialHost />
        </QuickAdd>
      </SidebarProvider>
    </TooltipProvider>
  );
}

/** The signed-in frame, with the tutorial's state around it so the Help menu and the step card share it. */
export default function Shell(props: { children: ReactNode; notices: ReactNode }) {
  return (
    <TutorialProvider>
      <ShellFrame {...props} />
    </TutorialProvider>
  );
}
