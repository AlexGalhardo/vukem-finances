import { Head, Link } from "@inertiajs/react";
import { ArrowUpRight, PiggyBank, Users, Wallet } from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { type Key, useFinanceI18n } from "../lib/i18n";
import GuideSummary from "../tutorial/GuideSummary";

// Twelve months of an example cash flow, as the share of the tallest bar.
const bars = [38, 52, 44, 61, 49, 70, 58, 66, 74, 63, 81, 92];

interface SampleRow {
  label: Key;
  category: Key;
  cents: number;
}

const sample: readonly SampleRow[] = [
  { label: "landRowSalary", category: "landCatIncome", cents: 625000 },
  { label: "landRowRent", category: "landCatHome", cents: -180000 },
  { label: "landRowGroceries", category: "landCatFood", cents: -33400 },
  { label: "landRowSavings", category: "landCatTransfer", cents: -100000 },
];

const features: readonly { icon: typeof Wallet; title: Key; body: Key }[] = [
  { icon: Wallet, title: "homeFeature1", body: "landFeature1Body" },
  { icon: PiggyBank, title: "homeFeature2", body: "landFeature2Body" },
  { icon: Users, title: "homeFeature3", body: "landFeature3Body" },
];

const plans: readonly { title: Key; items: readonly Key[] }[] = [
  { title: "billingFree", items: ["billingFreeAccounts", "billingFreeTransactions", "billingFreeMembers"] },
  { title: "billingPro", items: ["billingProUnlimited", "billingProImport", "billingProBudgets"] },
];

/**
 * The public landing page. The composition is ours: a left-aligned statement and one call to action beside an
 * example ledger, a quiet three-column list of what the product does, and nothing else. It uses the palette tokens
 * only, so it follows the theme, and it loads no outside font, image or script.
 */
export default function Home() {
  const { t, money } = useFinanceI18n();
  return (
    <>
      <Head title={t("personalFinance")} />
      <div className="mx-auto flex w-full max-w-[1200px] flex-col gap-20 px-4 py-12 md:px-8 md:py-20">
        <section className="grid items-center gap-12 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)] lg:gap-16">
          <div className="flex flex-col items-start gap-6">
            <p className="flex items-center gap-2 text-label text-brand-text">
              <span aria-hidden="true" className="size-2 rounded-full bg-brand" />
              {t("homeEyebrow")}
            </p>
            <h1 className="max-w-[16ch] text-[clamp(2.5rem,7vw,4.75rem)] leading-[1.02] font-semibold tracking-[-0.035em] text-balance">
              {t("homeTitle")}
            </h1>
            <p className="max-w-[48ch] text-lg text-pretty text-muted-foreground">{t("homeDescription")}</p>
            <div className="flex flex-wrap items-center gap-x-6 gap-y-3">
              <Button asChild variant="brand" size="lg" data-tour="home-cta">
                <Link href="/register">
                  {t("homeCta")}
                  <ArrowUpRight data-icon="inline-end" />
                </Link>
              </Button>
              <Link
                href="/login"
                className="text-body text-brand-text underline underline-offset-4 decoration-1 hover:decoration-2 pointer-coarse:py-3"
              >
                {t("homeSecondary")} {t("login")}
              </Link>
            </div>
          </div>

          <figure className="flex flex-col gap-6 rounded-xl border border-border bg-card p-5 text-card-foreground md:p-8">
            <figcaption className="flex items-center justify-between gap-3">
              <span className="text-label">{t("landMonth")}</span>
              <Badge variant="outline">{t("landExample")}</Badge>
            </figcaption>
            <div className="flex flex-col gap-1">
              <span className="text-caption text-muted-foreground">{t("previewBalance")}</span>
              <span className="text-[clamp(2rem,5vw,3rem)] leading-none font-semibold tracking-[-0.03em] tabular-nums">
                {money(2485000, "USD")}
              </span>
            </div>
            <div role="img" aria-label={t("landBars")} className="flex h-24 items-end gap-1.5">
              {bars.map((height, index) => (
                <span
                  // The bars never reorder and carry no data of their own.
                  // biome-ignore lint/suspicious/noArrayIndexKey: static decoration
                  key={index}
                  style={{ height: `${height}%` }}
                  className={index === bars.length - 1 ? "flex-1 rounded-sm bg-brand" : "flex-1 rounded-sm bg-border"}
                />
              ))}
            </div>
            <ul className="flex flex-col divide-y divide-border border-t border-border">
              {sample.map((row) => (
                <li key={row.label} className="flex items-center justify-between gap-3 py-3">
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate text-label">{t(row.label)}</span>
                    <span className="text-caption text-muted-foreground">{t(row.category)}</span>
                  </div>
                  <span className={row.cents > 0 ? "text-income tabular-nums" : "text-expense tabular-nums"}>
                    {row.cents > 0 ? "+" : "-"}
                    {money(Math.abs(row.cents), "USD")}
                  </span>
                </li>
              ))}
            </ul>
          </figure>
        </section>

        <section
          aria-label={t("landFeaturesLabel")}
          className="grid gap-10 border-t border-border pt-12 md:grid-cols-3"
        >
          {features.map(({ icon: Icon, title, body }) => (
            <div key={title} className="flex flex-col gap-3">
              <span className="flex size-10 items-center justify-center rounded-md bg-muted">
                <Icon aria-hidden="true" className="size-5" />
              </span>
              <h2 className="text-h3">{t(title)}</h2>
              <p className="text-body text-pretty text-muted-foreground">{t(body)}</p>
            </div>
          ))}
        </section>

        <section aria-label={t("landPlansLabel")} className="grid gap-4 border-t border-border pt-12 md:grid-cols-2">
          {plans.map((plan) => (
            <div key={plan.title} className="flex flex-col gap-3 rounded-xl border border-border p-5 md:p-8">
              <h2 className="text-h2">{t(plan.title)}</h2>
              <ul className="flex list-disc flex-col gap-1.5 ps-5 text-body text-muted-foreground">
                {plan.items.map((item) => (
                  <li key={item}>{t(item)}</li>
                ))}
              </ul>
            </div>
          ))}
        </section>

        <div className="flex flex-col gap-6 border-t border-border pt-8">
          <p className="text-h2 text-balance">{t("homeFooter")}</p>
          <div className="max-w-xl">
            <GuideSummary screenId="Home" />
          </div>
        </div>
      </div>
    </>
  );
}
