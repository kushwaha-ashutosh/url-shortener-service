// A small rotating palette used anywhere the UI cycles through colors
// by index — link list items, chart/table bars — so color carries
// meaning (this item vs. that item) rather than everything being one
// flat accent color. Values match the CSS custom properties defined
// in index.css so light/dark mode stay in sync automatically.
export const PALETTE = [
  "var(--coral)",
  "var(--teal)",
  "var(--violet)",
  "var(--amber)",
  "var(--pink)",
] as const;

export function colorForIndex(index: number): string {
  return PALETTE[index % PALETTE.length];
}
