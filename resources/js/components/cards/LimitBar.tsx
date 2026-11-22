import { HelpHint } from "../../help/HelpHint";
import { MoneyText } from "../MoneyText";
import { useFinanceI18n } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Progress } from "../ui/progress";
import type { CardOverview } from "../../page-props";

export interface LimitBarProps {
  card: CardOverview;
  currency: string;
  /** Shows the "Limit used" hint beside the figures; one per screen is enough. */
  help?: boolean;
}

/** The limit used, as a bar plus the same numbers as text: the bar alone never carries the meaning. */
export default function LimitBar({ card, currency, help = false }: LimitBarProps) {
  const { t, locale } = useFinanceI18n();
  const labels = { minus: t("moneyMinus"), plus: t("moneyPlus") };
  const percent = card.limit_cents > 0 ? Math.min(100, Math.round((card.usedCents / card.limit_cents) * 100)) : 0;
  const money = (cents: number): string =>
    new Intl.NumberFormat(locale, { style: "currency", currency }).format(cents / 100);
  return (
    <div className="flex flex-col gap-1">
      <Progress
        value={card.usedCents}
        max={Math.max(card.limit_cents, 1)}
        aria-label={`${t("cardLimitBar")}: ${card.name}`}
        aria-valuetext={`${money(card.usedCents)} / ${money(card.limit_cents)} (${percent}%)`}
      />
      <p className="flex flex-wrap justify-between gap-x-3 text-caption text-muted-foreground">
        <span>
          {t("cardUsed")}
          {help && <HelpHint id="hintCardLimitUsed" />}{" "}
          <MoneyText
            cents={card.usedCents}
            currency={currency}
            locale={locale}
            labels={labels}
            className={cn(card.overLimit && "text-(--vf-expense)")}
          />{" "}
          / <MoneyText cents={card.limit_cents} currency={currency} locale={locale} labels={labels} />
        </span>
        <span>
          {card.overLimit ? (
            <strong className="text-(--vf-expense)">{t("cardOverLimit")}</strong>
          ) : (
            <>
              {t("cardAvailable")}{" "}
              <MoneyText cents={card.availableCents} currency={currency} locale={locale} labels={labels} />
            </>
          )}
        </span>
      </p>
    </div>
  );
}
