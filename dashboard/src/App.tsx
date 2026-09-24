import { useMemo, useState } from "react";
import "./App.css";
import { CreateLinkForm } from "./components/CreateLinkForm";
import { LinksList } from "./components/LinksList";
import { QRModal } from "./components/QRModal";
import { StatsPanel } from "./components/StatsPanel";
import { ToastStack } from "./components/Toast";
import { useLocalLinks } from "./useLocalLinks";
import { useToasts } from "./useToasts";
import type { Link } from "./api";

function App() {
  const { links, addLink, removeLink } = useLocalLinks();
  const [selectedCode, setSelectedCode] = useState<string | null>(null);
  const [qrLink, setQrLink] = useState<Link | null>(null);
  const { toasts, push } = useToasts();

  const selectedLink = useMemo(
    () => links.find((l) => l.code === selectedCode) ?? null,
    [links, selectedCode]
  );

  async function copyLink(link: Link) {
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
          setSelectedCode(link.code);
          push("Short link created");
        }}
        onError={(message) => push(message, "error")}
      />

      <main className="layout">
        <section className="sidebar">
          <h2>Your links ({links.length})</h2>
          <LinksList
            links={links}
            selectedCode={selectedCode}
            onSelect={setSelectedCode}
            onCopy={copyLink}
            onShowQR={setQrLink}
            onRemove={(code) => {
              removeLink(code);
              if (selectedCode === code) setSelectedCode(null);
            }}
          />
        </section>

        <section className="content">
          {selectedLink ? (
            <StatsPanel
              key={selectedLink.code}
              link={selectedLink}
              onCopy={copyLink}
              onShowQR={setQrLink}
            />
          ) : (
            <div className="empty-panel content-empty">
              <span className="empty-panel-icon">📊</span>
              <p className="empty-state">Select a link to see its stats.</p>
            </div>
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
