import { useFinanceI18n } from "../../lib/i18n";

type Point = { month: string; balanceCents: number };

// LineChart draws a series of balances (the net worth at the end of each month) as an SVG polyline with a dot and a
// title per point. Values are positioned with float maths for pixels only; the money itself stays in cents.
export default function LineChart({ points, currency }: { points: Point[]; currency: string }) {
  const { t, money, month: monthLabel } = useFinanceI18n();
  const width = 480;
  const height = 180;
  const low = Math.min(0, ...points.map((p) => p.balanceCents));
  const high = Math.max(1, ...points.map((p) => p.balanceCents));
  const span = high - low;
  const step = points.length > 1 ? width / (points.length - 1) : 0;
  const at = (i: number, cents: number): [number, number] => [
    points.length > 1 ? i * step : width / 2,
    height - ((cents - low) / span) * height,
  ];
  const zero = height - ((0 - low) / span) * height;
  const line = points.map((p, i) => at(i, p.balanceCents).join(",")).join(" ");

  return (
    <svg viewBox={`0 0 ${width} ${height + 24}`} role="img" aria-label={t("rptTabNetWorth")} className="w-full">
      <line x1="0" x2={width} y1={zero} y2={zero} className="stroke-border" strokeDasharray="4 3" />
      <polyline points={line} fill="none" strokeWidth="2" className="stroke-chart-1" />
      {points.map((p, i) => {
        const [x, y] = at(i, p.balanceCents);
        return (
          <g key={p.month}>
            <rect x={x - 4} y={y - 4} width="8" height="8" className="fill-chart-1" transform={`rotate(45 ${x} ${y})`}>
              <title>{`${monthLabel(p.month)}: ${money(p.balanceCents, currency)}`}</title>
            </rect>
            {(points.length <= 12 || i % Math.ceil(points.length / 12) === 0) && (
              <text x={x} y={height + 16} textAnchor="middle" className="fill-muted-foreground text-[11px]">
                {monthLabel(p.month)}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
