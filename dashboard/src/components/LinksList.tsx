import type { Link } from "../api";

interface Props {
  links: Link[];
  selectedCode: string | null;
  onSelect: (code: string) => void;
  onRemove: (code: string) => void;
}

export function LinksList({ links, selectedCode, onSelect, onRemove }: Props) {
  if (links.length === 0) {
    return <p className="empty-state">No links yet — create one above.</p>;
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
            <a href={link.short_url} target="_blank" rel="noreferrer" onClick={(e) => e.stopPropagation()}>
              {link.short_url}
            </a>
            <span className="long-url" title={link.long_url}>
              {link.long_url}
            </span>
          </div>
          <button
            type="button"
            className="remove-btn"
            title="Remove from this list (does not delete the link)"
            onClick={(e) => {
              e.stopPropagation();
              onRemove(link.code);
            }}
          >
            ×
          </button>
        </li>
      ))}
    </ul>
  );
}
