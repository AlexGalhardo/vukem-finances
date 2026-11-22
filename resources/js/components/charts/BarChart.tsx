import { useFinanceI18n } from "../../lib/i18n";

type Point = { month: string; incomeCents: number; expenseCents: number };

// BarChart draws income and expense per month as plain SVG: no chart
// dependency, and the values stay readable in the title of each bar.
export default function BarChart({ points, currency }: { points: Point[]; currency: string }) {
  const { t, money: formatMoney, month: monthLabel } = useFinanceI18n();
  const max = Math.max(1, ...points.flatMap((p) => [p.incomeCents, p.expenseCents]));
  const width = 480;
  const height = 180;
  const group = width / Math.max(points.length, 1);
  const bar = Math.min(28, group / 3);

  return (
    <svg viewBox={`0 0 ${width} ${height + 44}`} role="img" aria-label={t("cashFlow")} className="w-full">
      <defs>
        {/* Expense bars are hatched, so income and expense differ by pattern and not by hue alone. */}
        <pattern id="bar-hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
          <rect width="6" height="6" className="fill-expense" />
          <line x1="0" y1="0" x2="0" y2="6" strokeWidth="2.5" className="stroke-background" />
        </pattern>
      </defs>
      {points.map((p, i) => {
        const x = i * group + group / 2;
        const hi = (p.incomeCents / max) * height;
        const he = (p.expenseCents / max) * height;
        return (
          <g key={p.month}>
            <rect x={x - bar - 1} y={height - hi} width={bar} height={hi} className="fill-income">
              <title>{`${monthLabel(p.month)}: ${t("income")} ${formatMoney(p.incomeCents, currency)}`}</title>
            </rect>
            <rect x={x + 1} y={height - he} width={bar} height={he} fill="url(#bar-hatch)">
              <title>{`${monthLabel(p.month)}: ${t("expenses")} ${formatMoney(p.expenseCents, currency)}`}</title>
            </rect>
            <text x={x} y={height + 16} textAnchor="middle" className="fill-muted-foreground text-[11px] tabular-nums">
              {monthLabel(p.month)}
            </text>
          </g>
        );
      })}
      <line x1="0" x2={width} y1={height} y2={height} className="stroke-border" />
      <rect x="0" y={height + 28} width="10" height="10" className="fill-income" />
      <text x="14" y={height + 37} className="fill-muted-foreground text-[11px] tabular-nums">
        {t("income")}
      </text>
      <rect x="90" y={height + 28} width="10" height="10" fill="url(#bar-hatch)" />
      <text x="104" y={height + 37} className="fill-muted-foreground text-[11px] tabular-nums">
        {t("expenses")}
      </text>
    </svg>
  );
}
