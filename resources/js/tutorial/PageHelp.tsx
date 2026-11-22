import { usePage } from "@inertiajs/react";
import type { ReactElement } from "react";
import { Button } from "../components/ui/button";
import { HelpSheet } from "../help/HelpSheet";
import { screenIdForPath } from "../help/screens";
import { useFinanceI18n } from "../lib/i18n";
import { tutorialScreens } from "./screens";
import TutorialCenter from "./TutorialCenter";
import { useTutorial } from "./useTutorial";

const SHORTCUTS = ["open-palette", "open-shortcut-help", "open-page-help", "open-tutorial"] as const;

/** The help of the current screen: why it exists and its steps in words, then the tutorial center. */
export default function PageHelp({
  open,
  onOpenChange,
  fallbackTitle,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fallbackTitle: string;
}): ReactElement {
  const page = usePage();
  const { t } = useFinanceI18n();
  const tutorial = useTutorial();
  const screenId = screenIdForPath(page.url);
  const screen = tutorialScreens.find((candidate) => candidate.id === screenId);
  const teachable = tutorial.screens.some((candidate) => candidate.id === screenId);
  const title = screen?.nameKey ? t(screen.nameKey) : fallbackTitle;
  return (
    <HelpSheet
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      why={screen ? t(screen.whyKey) : undefined}
      steps={screen?.steps.map((step) => ({ title: t(step.titleKey), body: t(step.bodyKey) }))}
      shortcutActions={SHORTCUTS}
      footer={
        <>
          {teachable && screenId && (
            <Button
              type="button"
              variant="brand"
              className="self-start"
              onClick={() => {
                onOpenChange(false);
                tutorial.start(screenId);
              }}
            >
              {t("tutorialShowScreen")}
            </Button>
          )}
          <TutorialCenter onStart={() => onOpenChange(false)} />
        </>
      }
    />
  );
}
