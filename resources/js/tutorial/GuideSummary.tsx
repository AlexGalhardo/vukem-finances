import type { ReactElement } from "react";
import { useFinanceI18n } from "../lib/i18n";
import { tutorialScreens } from "./screens";

/**
 * The "How this works" block of a screen shown before sign-in: a collapsed details element with the same steps as
 * the tour, because there is no signed-in person and no stored progress to spotlight. Pages place it under the form.
 */
export default function GuideSummary({ screenId }: { screenId: string }): ReactElement | null {
  const { t } = useFinanceI18n();
  const screen = tutorialScreens.find((candidate) => candidate.id === screenId);
  if (!screen) return null;
  return (
    <details data-slot="tutorial-guide" className="rounded-md border border-border p-3 text-body">
      <summary className="cursor-pointer text-label">{t("tutorialGuideSummary")}</summary>
      <p className="mt-2 text-pretty text-muted-foreground">{t(screen.whyKey)}</p>
      <ol className="mt-2 flex list-decimal flex-col gap-2 ps-5">
        {screen.steps.map((step) => (
          <li key={step.target}>
            <span className="text-label">{t(step.titleKey)}</span>{" "}
            <span className="text-muted-foreground">{t(step.bodyKey)}</span>
          </li>
        ))}
      </ol>
    </details>
  );
}
