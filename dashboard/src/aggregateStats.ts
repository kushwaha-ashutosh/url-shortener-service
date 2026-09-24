import { getStats, type NamedCount, type Stats } from "./api";

// Merges several links' stats into one, for the "All links" and
// "Favourites" aggregate views — there's no backend endpoint for this
// (the API deliberately has no "list all links"), so it's done
// client-side from the per-link stats the dashboard already fetches.
export function mergeStats(list: Stats[]): Stats {
  const totalClicks = list.reduce((sum, s) => sum + s.total_clicks, 0);

  const dayTotals = new Map<string, number>();
  for (const s of list) {
    for (const d of s.clicks_by_day ?? []) {
      dayTotals.set(d.day, (dayTotals.get(d.day) ?? 0) + d.clicks);
    }
  }
  const clicksByDay = [...dayTotals.entries()]
    .map(([day, clicks]) => ({ day, clicks }))
    .sort((a, b) => a.day.localeCompare(b.day));

  function mergeNamed(pick: (s: Stats) => NamedCount[] | null): NamedCount[] {
    const totals = new Map<string, number>();
    for (const s of list) {
      for (const n of pick(s) ?? []) {
        totals.set(n.name, (totals.get(n.name) ?? 0) + n.clicks);
      }
    }
    return [...totals.entries()]
      .map(([name, clicks]) => ({ name, clicks }))
      .sort((a, b) => b.clicks - a.clicks)
      .slice(0, 10);
  }

  return {
    code: "",
    total_clicks: totalClicks,
    clicks_by_day: clicksByDay,
    top_referrers: mergeNamed((s) => s.top_referrers),
    top_browsers: mergeNamed((s) => s.top_browsers),
  };
}

// Fetches stats for each code and merges them, skipping any that fail
// (e.g. a stale localStorage entry for a link that no longer exists
// server-side) rather than failing the whole aggregate over one bad
// code.
export async function fetchAggregateStats(codes: string[]): Promise<Stats> {
  if (codes.length === 0) {
    return { code: "", total_clicks: 0, clicks_by_day: [], top_referrers: [], top_browsers: [] };
  }
  const results = await Promise.allSettled(codes.map(getStats));
  const fulfilled = results
    .filter((r): r is PromiseFulfilledResult<Stats> => r.status === "fulfilled")
    .map((r) => r.value);
  return mergeStats(fulfilled);
}
