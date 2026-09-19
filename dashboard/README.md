# dashboard

A small React + TypeScript dashboard for creating short links and
watching their stats update live.

## What it does

- Create a link (with an optional custom code) via the API
- See your created links in a list, kept in `localStorage` — the API
  has no "list all links" endpoint by design, so this list is a
  per-browser convenience, not backend state
- Select a link to see its stats: a 30-day clicks-per-day bar chart,
  top referrers, and browser breakdown, polling the API every 5s so
  new clicks (which the backend batches asynchronously — see the root
  README) show up without a manual refresh

No charting library — the bar chart is ~30 lines of hand-rolled SVG.
Worth revisiting once this needs more than one chart type, axes, or
tooltips beyond a native `<title>`; not worth it for a fixed set of
bars today.

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
