import { useState } from "react";
import type { DailyClicks } from "../api";

interface Props {
  data: DailyClicks[];
}

const WIDTH = 600;
const HEIGHT = 180;
const PADDING_BOTTOM = 22;
const GRID_LINES = 4;

// A hand-rolled bar chart rather than pulling in a charting library —
// this is a fixed set of bars with a tooltip, which is a reasonable
// amount of SVG by hand. Worth revisiting once this needs zoom, pan,
// multiple series, or more chart types than one.
export function DailyClicksChart({ data }: Props) {
  const [hovered, setHovered] = useState<number | null>(null);

  if (data.length === 0) {
    return (
      <div className="empty-panel chart-empty">
        <p className="empty-state">No clicks in the last 30 days yet.</p>
      </div>
    );
  }

  const plotHeight = HEIGHT - PADDING_BOTTOM;
  const barGap = 4;
  const barWidth = Math.max(3, WIDTH / data.length - barGap);
  const maxClicks = Math.max(...data.map((d) => d.clicks), 1);

  return (
    <div className="chart-wrap">
      <svg
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        className="daily-chart"
        role="img"
        aria-label="Clicks per day"
        onMouseLeave={() => setHovered(null)}
      >
        {Array.from({ length: GRID_LINES + 1 }).map((_, i) => {
          const y = (plotHeight / GRID_LINES) * i;
          return <line key={i} x1={0} x2={WIDTH} y1={y} y2={y} className="grid-line" />;
        })}

        {data.map((d, i) => {
          const barHeight = (d.clicks / maxClicks) * plotHeight;
          const x = i * (barWidth + barGap);
          const y = plotHeight - barHeight;
          const isHovered = hovered === i;
          return (
            <g
              key={d.day}
              onMouseEnter={() => setHovered(i)}
              onMouseMove={() => setHovered(i)}
            >
              <rect x={x} y={0} width={barWidth + barGap} height={plotHeight} fill="transparent" />
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={Math.max(barHeight, d.clicks > 0 ? 2 : 0)}
                rx={2}
                className={`bar ${isHovered ? "bar-hovered" : ""}`}
              />
            </g>
          );
        })}

        {hovered !== null && (
          <ChartTooltip
            point={data[hovered]}
            x={hovered * (barWidth + barGap) + barWidth / 2}
            barTop={plotHeight - (data[hovered].clicks / maxClicks) * plotHeight}
          />
        )}
      </svg>
      <div className="chart-axis">
        <span>{data[0].day}</span>
        <span>{data[data.length - 1].day}</span>
      </div>
    </div>
  );
}

function ChartTooltip({
  point,
  x,
  barTop,
}: {
  point: DailyClicks;
  x: number;
  barTop: number;
}) {
  const clampedX = Math.min(Math.max(x, 40), WIDTH - 40);
  return (
    <g transform={`translate(${clampedX}, ${Math.max(barTop - 10, 14)})`}>
      <rect x={-34} y={-16} width={68} height={22} rx={5} className="tooltip-bg" />
      <text textAnchor="middle" y={0} className="tooltip-text">
        {point.clicks} click{point.clicks === 1 ? "" : "s"}
      </text>
    </g>
  );
}
