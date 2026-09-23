import type { Link } from "../api";

interface Props {
  links: Link[];
  selectedCode: string | null;
  onSelect: (code: string) => void;
  onRemove: (code: string) => void;
  onCopy: (link: Link) => void;
}

export function LinksList({ links, selectedCode, onSelect, onRemove, onCopy }: Props) {
  if (links.length === 0) {
    return (
      <div className="empty-panel">
        <p className="empty-state">No links yet — create one above to get started.</p>
      </div>
    );
  }

  return (
    <ul className="links-list">
      {links.map((link) => (
        <li
          key={link.code}
          className={link.code === selectedCode ? "selected" : ""}
          onClick={() => onSelect(link.code)}
        >
          <div className="link-info">
            <span className="short-url">{link.short_url.replace(/^https?:\/\//, "")}</span>
            <span className="long-url" title={link.long_url}>
              {link.long_url}
            </span>
          </div>
          <div className="link-actions">
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
