import { Link, usePage } from "@inertiajs/react";
import type { ReactElement, ReactNode } from "react";
import Mark from "../components/Mark";
import Preferences from "../components/Preferences";
import ThemeToggle from "../components/ThemeToggle";
import type { SharedProps } from "../page-props";

/**
 * The frame of every signed-out screen: the product mark and the language and theme controls on one line, then a
 * single 400 px column on the page background. No card, border or illustration around the form. The page brings its
 * own h1 and form; the theme toggle stays the last control of the header.
 */
export default function AuthLayout({ children, notices }: { children: ReactNode; notices?: ReactNode }): ReactElement {
  const { appName } = usePage<SharedProps>().props;
  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="flex h-16 items-center justify-between gap-3 px-4 md:px-8">
        <Link href="/" className="flex min-w-0 items-center gap-2 text-h2">
          <Mark />
          <span className="truncate max-[359px]:sr-only">{appName}</span>
        </Link>
        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          <Preferences />
          <ThemeToggle />
        </div>
      </header>
      <main className="mx-auto flex w-full max-w-[400px] flex-1 flex-col gap-6 px-4 pt-8 pb-12 md:pt-16">
        {notices}
        {children}
      </main>
    </div>
  );
}
