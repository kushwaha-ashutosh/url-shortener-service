import type { NamedCount } from "../api";

interface Props {
  title: string;
  rows: NamedCount[];
}

export function NamedCountTable({ title, rows }: Props) {
  return (
    <div className="named-count-table">
      <h3>{title}</h3>
      {rows.length === 0 ? (
        <p className="empty-state">No data yet.</p>
      ) : (
        <table>
          <tbody>
            {rows.map((row) => (
              <tr key={row.name}>
                <td className="name-cell" title={row.name}>
                  {row.name}
                </td>
                <td className="count-cell">{row.clicks}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
