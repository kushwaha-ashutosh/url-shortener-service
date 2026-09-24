import { useEffect, useState } from "react";
import type { Link } from "./api";

const STORAGE_KEY = "url-shortener-dashboard:links";

export interface StoredLink extends Link {
  favorite: boolean;
}

// The API has no "list all links" endpoint, so the dashboard is the
// only source of truth for "which links did I create" — kept in
// localStorage so a page reload doesn't lose them. This is a per-
// browser convenience list, not a backend feature: two people, or the
// same person in two browsers, see different lists.
function loadLinks(): StoredLink[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as Partial<StoredLink>[];
    // favorite defaults to false for entries saved before this field
    // existed, rather than dropping or erroring on them.
    return parsed.map((l) => ({ ...(l as Link), favorite: l.favorite ?? false }));
  } catch {
    return [];
  }
}

export function useLocalLinks() {
  const [links, setLinks] = useState<StoredLink[]>(loadLinks);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(links));
    } catch {
      // best-effort persistence only
    }
  }, [links]);

  function addLink(link: Link) {
    setLinks((prev) => [{ ...link, favorite: false }, ...prev.filter((l) => l.code !== link.code)]);
  }

  function removeLink(code: string) {
    setLinks((prev) => prev.filter((l) => l.code !== code));
  }

  function toggleFavorite(code: string) {
    setLinks((prev) => prev.map((l) => (l.code === code ? { ...l, favorite: !l.favorite } : l)));
  }

  return { links, addLink, removeLink, toggleFavorite };
}
