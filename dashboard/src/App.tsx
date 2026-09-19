import { useState } from "react";
import "./App.css";
import { CreateLinkForm } from "./components/CreateLinkForm";
import { LinksList } from "./components/LinksList";
import { StatsPanel } from "./components/StatsPanel";
import { useLocalLinks } from "./useLocalLinks";

function App() {
  const { links, addLink, removeLink } = useLocalLinks();
  const [selectedCode, setSelectedCode] = useState<string | null>(null);

  return (
    <div className="app">
      <header>
        <h1>url-shortener dashboard</h1>
        <p className="subtitle">
          Links created here are remembered in this browser only — the API has no "list all
          links" endpoint by design.
        </p>
      </header>

      <CreateLinkForm
        onCreated={(link) => {
          addLink(link);
          setSelectedCode(link.code);
        }}
      />

      <main className="layout">
        <section className="sidebar">
          <h2>Your links</h2>
          <LinksList
            links={links}
            selectedCode={selectedCode}
            onSelect={setSelectedCode}
            onRemove={(code) => {
              removeLink(code);
              if (selectedCode === code) setSelectedCode(null);
            }}
          />
        </section>

        <section className="content">
          {selectedCode ? (
            <StatsPanel code={selectedCode} />
          ) : (
            <p className="empty-state">Select a link to see its stats.</p>
          )}
        </section>
      </main>
    </div>
  );
}

export default App;
