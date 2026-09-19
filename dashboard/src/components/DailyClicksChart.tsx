import type { DailyClicks } from "../api";

interface Props {
  data: DailyClicks[];
}

// A hand-rolled bar chart rather than pulling in a charting library —
// this is ~30 lines of SVG for a fixed number of bars, which is a
// worse trade than a library once you need axes, tooltips, zoom, or
// more than one chart type, but not before that.
export function DailyClicksChart({ data }: Props) {
  if (data.length === 0) {
    return <p className="empty-state">No clicks in the last 30 days yet.</p>;
  }

  const width = 560;
  const height = 160;
  const barGap = 4;
  const barWidth = Math.max(4, width / data.length - barGap);
  const maxClicks = Math.max(...data.map((d) => d.clicks), 1);

  return (
    <svg viewBox={`0 0 ${width} ${height + 20}`} className="daily-chart" role="img" aria-label="Clicks per day">
      {data.map((d, i) => {
        const barHeight = (d.clicks / maxClicks) * height;
        const x = i * (barWidth + barGap);
        const y = height - barHeight;
        return (
          <g key={d.day}>
            <rect x={x} y={y} width={barWidth} height={barHeight} rx={2} className="bar">
              <title>{`${d.day}: ${d.clicks} click${d.clicks === 1 ? "" : "s"}`}</title>
            </rect>
          </g>
        );
      })}
      <text x={0} y={height + 15} className="axis-label">
        {data[0].day}
      </text>
      <text x={width} y={height + 15} textAnchor="end" className="axis-label">
        {data[data.length - 1].day}
      </text>
    </svg>
  );
}
