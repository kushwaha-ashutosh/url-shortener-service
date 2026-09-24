import type { Link } from "../api";
import { colorForIndex } from "../palette";

interface Props {
  links: Link[];
  selectedCode: string | null;
  onSelect: (code: string) => void;
  onRemove: (code: string) => void;
  onCopy: (link: Link) => void;
  onShowQR: (link: Link) => void;
}

export function LinksList({ links, selectedCode, onSelect, onRemove, onCopy, onShowQR }: Props) {
  if (links.length === 0) {
    return (
      <div className="empty-panel">
        <span className="empty-panel-icon">🔗</span>
        <p className="empty-state">No links yet — create one above to get started.</p>
      </div>
    );
  }

  return (
    <ul className="links-list">
      {links.map((link, i) => (
        <li
          key={link.code}
          className={link.code === selectedCode ? "selected" : ""}
          style={{ ["--item-color" as string]: colorForIndex(i) }}
          onClick={() => onSelect(link.code)}
        >
          <div className="link-info">
            <a
              className="short-url"
              href={link.short_url}
              target="_blank"
              rel="noreferrer noopener"
              title="Open — follows the redirect to the destination"
              onClick={(e) => e.stopPropagation()}
            >
              {link.short_url.replace(/^https?:\/\//, "")}
            </a>
            <span className="long-url" title={link.long_url}>
              {link.long_url}
            </span>
          </div>
          <div className="link-actions">
            <button
              type="button"
              className="icon-btn"
              title="Show QR code"
              onClick={(e) => {
                e.stopPropagation();
                onShowQR(link);
              }}
            >
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="2" y="2" width="5" height="5" rx="0.5" />
                <rect x="9" y="2" width="5" height="5" rx="0.5" />
                <rect x="2" y="9" width="5" height="5" rx="0.5" />
                <path d="M9.5 9.5h2M12.5 9.5h1.5M9.5 12.5h1.5M9.5 14h1M12.5 12v2h1.5v-2z" strokeLinecap="round" />
              </svg>
            </button>
            <button
              type="button"
              className="icon-btn"
              title="Copy short URL"
              onClick={(e) => {
                e.stopPropagation();
                onCopy(link);
              }}
            >
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="5" y="5" width="9" height="9" rx="1.5" />
                <path d="M3 10.5V3a1 1 0 0 1 1-1h7.5" />
              </svg>
            </button>
            <button
              type="button"
              className="icon-btn danger"
              title="Remove from this list (does not delete the link)"
              onClick={(e) => {
                e.stopPropagation();
                onRemove(link.code);
              }}
            >
              ×
            </button>
          </div>
        </li>
      ))}
    </ul>
  );
}
