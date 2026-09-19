import { useEffect, useState } from "react";
import { ApiError, getStats, type Stats } from "../api";
import { DailyClicksChart } from "./DailyClicksChart";
import { NamedCountTable } from "./NamedCountTable";

interface Props {
  code: string;
}

const POLL_INTERVAL_MS = 5000;

export function StatsPanel({ code }: Props) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const s = await getStats(code);
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
  }, [code]);

  if (error) {
    return <p className="error-text">{error}</p>;
  }
  if (!stats) {
    return <p className="empty-state">Loading stats…</p>;
  }

  return (
    <div className="stats-panel">
      <div className="stats-header">
        <h2>{stats.code}</h2>
        <span className="total-clicks">{stats.total_clicks} total clicks</span>
      </div>

      <DailyClicksChart data={stats.clicks_by_day ?? []} />

      <div className="stats-tables">
        <NamedCountTable title="Top referrers" rows={stats.top_referrers ?? []} />
        <NamedCountTable title="Browsers" rows={stats.top_browsers ?? []} />
      </div>
    </div>
  );
}
