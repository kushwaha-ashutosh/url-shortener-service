import { useEffect, useState } from "react";
import { ApiError, getStats, type Link, type Stats } from "../api";
import { DailyClicksChart } from "./DailyClicksChart";
import { NamedCountTable } from "./NamedCountTable";

interface Props {
  link: Link;
  onCopy: (link: Link) => void;
}

const POLL_INTERVAL_MS = 5000;

export function StatsPanel({ link, onCopy }: Props) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const s = await getStats(link.code);
        if (!cancelled) {
          setStats(s);
          setError(null);
        }
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof ApiError ? err.message : "Could not reach the API");
      }
    }

    load();
    // Clicks flush asynchronously on the backend (batched every ~2s),
    // so a newly-created click doesn't show up instantly — polling
    // here reflects that same eventual-consistency the API itself has.
    const interval = setInterval(load, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [link.code]);

  if (error) {
    return (
      <div className="empty-panel">
        <p className="error-text">{error}</p>
      </div>
    );
  }
  if (!stats) {
    return <StatsSkeleton />;
  }

  const topReferrer = stats.top_referrers?.[0];
  const topBrowser = stats.top_browsers?.[0];

  return (
    <div className="stats-panel">
      <div className="stats-header">
        <div>
          <h2>{stats.code}</h2>
          <button type="button" className="short-url-btn" onClick={() => onCopy(link)}>
            {link.short_url.replace(/^https?:\/\//, "")}
            <svg viewBox="0 0 16 16" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="1.4">
              <rect x="5" y="5" width="9" height="9" rx="1.5" />
              <path d="M3 10.5V3a1 1 0 0 1 1-1h7.5" />
            </svg>
          </button>
        </div>
      </div>

      <div className="stat-cards">
        <div className="stat-card">
          <span className="stat-card-label">Total clicks</span>
          <span className="stat-card-value">{stats.total_clicks}</span>
        </div>
        <div className="stat-card">
          <span className="stat-card-label">Top referrer</span>
          <span className="stat-card-value stat-card-value-text">{topReferrer?.name ?? "—"}</span>
        </div>
        <div className="stat-card">
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

function StatsSkeleton() {
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
