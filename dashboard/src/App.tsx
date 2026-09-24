import { useMemo, useState } from "react";
import "./App.css";
import { AggregateStatsPanel } from "./components/AggregateStatsPanel";
import { CreateLinkForm } from "./components/CreateLinkForm";
import { LinksList } from "./components/LinksList";
import { QRModal } from "./components/QRModal";
import { StatsPanel } from "./components/StatsPanel";
import { ToastStack } from "./components/Toast";
import { useLocalLinks, type StoredLink } from "./useLocalLinks";
import { useToasts } from "./useToasts";

// The content pane shows one of: the "All links" aggregate, the
// "Favourites" aggregate, or one specific link's stats. The two
// aggregate views aren't real link codes, so sentinel strings pick
// them out from a real `selection` that's otherwise always a code.
const VIEW_ALL = "__all__";
const VIEW_FAVORITES = "__favorites__";

function App() {
  const { links, addLink, removeLink, toggleFavorite } = useLocalLinks();
  // Defaults to the aggregate view rather than an empty "select a
  // link" placeholder, so there's always something to look at.
  const [selection, setSelection] = useState<string>(VIEW_ALL);
  const [qrLink, setQrLink] = useState<StoredLink | null>(null);
  const { toasts, push } = useToasts();

  const favoriteLinks = useMemo(() => links.filter((l) => l.favorite), [links]);
  const selectedLink = useMemo(() => links.find((l) => l.code === selection) ?? null, [links, selection]);

  async function copyLink(link: StoredLink) {
    try {
      await navigator.clipboard.writeText(link.short_url);
      push("Copied to clipboard");
    } catch {
      push("Couldn't copy — copy it manually", "error");
    }
  }

  return (
    <div className="app">
      <header>
        <div className="brand">
          <span className="brand-mark">🔗</span>
          <h1>url-shortener</h1>
        </div>
        <p className="subtitle">
          Shorten any link, watch clicks roll in live, and share a scannable QR code — all in
          one place.
        </p>
      </header>

      <CreateLinkForm
        onCreated={(link) => {
          addLink(link);
          setSelection(link.code);
          push("Short link created");
        }}
        onError={(message) => push(message, "error")}
      />

      <main className="layout">
        <section className="sidebar">
          <div className="view-tabs">
            <button
              type="button"
              className={`view-tab ${selection === VIEW_ALL ? "active" : ""}`}
              onClick={() => setSelection(VIEW_ALL)}
            >
              🔗 All links
            </button>
            <button
              type="button"
              className={`view-tab ${selection === VIEW_FAVORITES ? "active" : ""}`}
              onClick={() => setSelection(VIEW_FAVORITES)}
            >
              ⭐ Favourites
            </button>
          </div>

          <h2>Your links ({links.length})</h2>
          <LinksList
            links={links}
            selectedCode={selection}
            onSelect={setSelection}
            onCopy={copyLink}
            onShowQR={setQrLink}
            onToggleFavorite={toggleFavorite}
            onRemove={(code) => {
              removeLink(code);
              if (selection === code) setSelection(VIEW_ALL);
            }}
          />
        </section>

        <section className="content">
          {selection === VIEW_ALL && (
            <AggregateStatsPanel
              title="All links"
              icon="🔗"
              links={links}
              emptyMessage="No links yet — create one above to get started."
            />
          )}
          {selection === VIEW_FAVORITES && (
            <AggregateStatsPanel
              title="Favourites"
              icon="⭐"
              links={favoriteLinks}
              emptyMessage="No favourites yet — star a link to see it here."
            />
          )}
          {selectedLink && (
            <StatsPanel key={selectedLink.code} link={selectedLink} onCopy={copyLink} onShowQR={setQrLink} />
          )}
        </section>
      </main>

      {qrLink && (
        <QRModal
          link={qrLink}
          onClose={() => setQrLink(null)}
          onError={(message) => push(message, "error")}
        />
      )}

      <ToastStack toasts={toasts} />
    </div>
  );
}

export default App;
