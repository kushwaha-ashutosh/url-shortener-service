const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8081";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export interface Link {
  code: string;
  short_url: string;
  long_url: string;
}

export interface DailyClicks {
  day: string;
  clicks: number;
}

export interface NamedCount {
  name: string;
  clicks: number;
}

export interface Stats {
  code: string;
  total_clicks: number;
  clicks_by_day: DailyClicks[] | null;
  top_referrers: NamedCount[] | null;
  top_browsers: NamedCount[] | null;
}

async function parseErrorBody(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    return body.error ?? res.statusText;
  } catch {
    return res.statusText;
  }
}

export async function createLink(url: string, customCode?: string): Promise<Link> {
  const res = await fetch(`${API_BASE}/api/links`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url, custom_code: customCode || undefined }),
  });
  if (!res.ok) {
    throw new ApiError(res.status, await parseErrorBody(res));
  }
  return res.json();
}

export async function getStats(code: string): Promise<Stats> {
  const res = await fetch(`${API_BASE}/api/links/${code}/stats`);
  if (!res.ok) {
    throw new ApiError(res.status, await parseErrorBody(res));
  }
  return res.json();
}
