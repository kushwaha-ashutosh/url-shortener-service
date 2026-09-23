import type { NamedCount } from "../api";

interface Props {
  title: string;
  rows: NamedCount[];
}

export function NamedCountTable({ title, rows }: Props) {
  const max = Math.max(...rows.map((r) => r.clicks), 1);

  return (
    <div className="named-count-table">
      <h3>{title}</h3>
      {rows.length === 0 ? (
        <p className="empty-state">No data yet.</p>
      ) : (
        <ul className="count-rows">
          {rows.map((row) => (
            <li key={row.name}>
              <div className="count-row-top">
                <span className="name-cell" title={row.name}>
                  {row.name}
                </span>
                <span className="count-cell">{row.clicks}</span>
              </div>
              <div className="count-bar-track">
                <div className="count-bar-fill" style={{ width: `${(row.clicks / max) * 100}%` }} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
