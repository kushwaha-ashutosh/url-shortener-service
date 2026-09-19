import { useEffect, useState } from "react";
import type { Link } from "./api";

const STORAGE_KEY = "url-shortener-dashboard:links";

// The API has no "list all links" endpoint, so the dashboard is the
// only source of truth for "which links did I create" — kept in
// localStorage so a page reload doesn't lose them. This is a per-
// browser convenience list, not a backend feature: two people, or the
// same person in two browsers, see different lists.
function loadLinks(): Link[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Link[]) : [];
  } catch {
    return [];
  }
}

export function useLocalLinks() {
  const [links, setLinks] = useState<Link[]>(loadLinks);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(links));
    } catch {
      // best-effort persistence only
    }
  }, [links]);

  function addLink(link: Link) {
    setLinks((prev) => [link, ...prev.filter((l) => l.code !== link.code)]);
  }

  function removeLink(code: string) {
    setLinks((prev) => prev.filter((l) => l.code !== code));
  }

  return { links, addLink, removeLink };
}
