import { usePage } from "@inertiajs/react";
import type { ReactElement } from "react";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "../components/ui/card";
import { screenIdForPath } from "../help/screens";
import { useFinanceI18n } from "../lib/i18n";
import { useTutorial } from "./useTutorial";

/**
 * The dismissible welcome on the Dashboard, shown on the first signed-in visit. It never opens the tour by itself
 * and never takes focus: the person starts it, or says "Not now".
 */
export default function WelcomeCard(): ReactElement | null {
  const tutorial = useTutorial();
  const page = usePage();
  const { t } = useFinanceI18n();
  const { state } = tutorial;
  if (screenIdForPath(page.url) !== "Dashboard" || state.welcomeDismissed || state.status !== "idle") return null;
  if (tutorial.screens.length === 0) return null;
  return (
    <Card data-slot="tutorial-welcome" role="region" aria-labelledby="tutorial-welcome-title" data-print="hide">
      <CardContent className="flex flex-col gap-3">
        <CardTitle id="tutorial-welcome-title">{t("tutorialWelcomeTitle")}</CardTitle>
        <CardDescription className="max-w-[65ch] text-body text-pretty">{t("tutorialWelcomeBody")}</CardDescription>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="brand" onClick={() => tutorial.start()}>
            {t("tutorialTakeTour")}
          </Button>
          <Button type="button" variant="ghost" onClick={tutorial.dismissWelcome}>
            {t("tutorialNotNow")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
