# dashboard

A small React + TypeScript dashboard for creating short links and
watching their stats update live.

## What it does

- Create a link (with an optional custom code) via the API
- See your created links in a list, kept in `localStorage` — the API
  has no "list all links" endpoint by design, so this list is a
  per-browser convenience, not backend state
- Select a link to see its stats: a 30-day clicks-per-day bar chart
  (hover a bar for an exact count), top referrers, and browser
  breakdown, polling the API every 5s so new clicks (which the backend
  batches asynchronously — see the root README) show up without a
  manual refresh
- View or download a link's QR code (generated server-side by the
  backend, so it's a real, cacheable image, not something rendered
  only in this page)
- Click a short link directly (a real `<a href>`, opens in a new tab)
  to actually follow the redirect, separately from copying it
- Star any link as a favourite, and switch between three views: one
  link's stats, an "All links" aggregate, or a "Favourites" aggregate
  — the aggregates are computed client-side by fetching each link's
  stats and merging them (there's still no backend "list all links" or
  "aggregate stats" endpoint, so this is dashboard-only math, not a new
  API capability). Opens to "All links" by default instead of an empty
  "select a link" placeholder.

No charting library — the bar chart is ~30 lines of hand-rolled SVG
with a custom hover tooltip. Worth revisiting once this needs more
than one chart type or axes beyond two labels; not worth it for a
fixed set of bars today.

## Running it

The backend must be running first (see the root README) with CORS
configured to allow this dev server's origin — the default
`ALLOWED_ORIGIN=http://localhost:5173` on the backend already matches
Vite's default port.

```bash
cp .env.example .env
npm install
npm run dev
```

## Scripts

```bash
npm run dev      # dev server, :5173
npm run build    # tsc -b && vite build
npm run lint     # oxlint
```
