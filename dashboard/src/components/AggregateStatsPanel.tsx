import { useEffect, useMemo, useState } from "react";
import { fetchAggregateStats } from "../aggregateStats";
import type { Stats } from "../api";
import type { StoredLink } from "../useLocalLinks";
import { DailyClicksChart } from "./DailyClicksChart";
import { NamedCountTable } from "./NamedCountTable";

interface Props {
  title: string;
  icon: string;
  links: StoredLink[];
  emptyMessage: string;
}

const POLL_INTERVAL_MS = 5000;

export function AggregateStatsPanel({ title, icon, links, emptyMessage }: Props) {
  const codes = useMemo(() => links.map((l) => l.code), [links]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const s = await fetchAggregateStats(codes);
        if (!cancelled) {
          setStats(s);
          setError(null);
        }
      } catch {
        if (!cancelled) setError("Could not reach the API");
      }
    }

    load();
    const interval = setInterval(load, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
    // codes is re-derived (new array) only when `links` itself changes,
    // via the useMemo above — safe to depend on directly.
  }, [codes]);

  if (links.length === 0) {
    return (
      <div className="empty-panel content-empty">
        <span className="empty-panel-icon">{icon}</span>
        <p className="empty-state">{emptyMessage}</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="empty-panel">
        <span className="empty-panel-icon">⚠️</span>
        <p className="error-text">{error}</p>
      </div>
    );
  }
  if (!stats) {
    return <AggregateSkeleton />;
  }

  const topReferrer = stats.top_referrers?.[0];
  const topBrowser = stats.top_browsers?.[0];

  return (
    <div className="stats-panel">
      <div className="stats-header">
        <div>
          <h2>
            {icon} {title}
          </h2>
          <p className="aggregate-count">
            {links.length} link{links.length === 1 ? "" : "s"}
          </p>
        </div>
      </div>

      <div className="stat-cards">
        <div className="stat-card stat-card-clicks">
          <span className="stat-card-icon">🖱️</span>
          <span className="stat-card-label">Total clicks</span>
          <span className="stat-card-value">{stats.total_clicks}</span>
        </div>
        <div className="stat-card stat-card-referrer">
          <span className="stat-card-icon">🔗</span>
          <span className="stat-card-label">Top referrer</span>
          <span className="stat-card-value stat-card-value-text">{topReferrer?.name ?? "—"}</span>
        </div>
        <div className="stat-card stat-card-browser">
          <span className="stat-card-icon">🌐</span>
          <span className="stat-card-label">Top browser</span>
          <span className="stat-card-value stat-card-value-text">{topBrowser?.name ?? "—"}</span>
        </div>
      </div>

      <DailyClicksChart data={stats.clicks_by_day ?? []} />

      <div className="stats-tables">
        <NamedCountTable title="Top referrers" rows={stats.top_referrers ?? []} />
        <NamedCountTable title="Browsers" rows={stats.top_browsers ?? []} />
      </div>
    </div>
  );
}

function AggregateSkeleton() {
  return (
    <div className="stats-panel">
      <div className="skeleton skeleton-title" />
      <div className="stat-cards">
        <div className="skeleton skeleton-card" />
        <div className="skeleton skeleton-card" />
        <div className="skeleton skeleton-card" />
      </div>
      <div className="skeleton skeleton-chart" />
    </div>
  );
}
