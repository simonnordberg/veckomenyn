import { useSyncExternalStore } from "react";

// Route is the set of canonical URLs the app understands. Everything the user
// can navigate to should appear here so the URL bar, back/forward, and
// deep-linking all work the same way.
export type Route =
  | { kind: "current" } //          /
  | { kind: "week"; iso: string } //         /weeks/:iso
  | { kind: "new" } //              /weeks/new
  | { kind: "print"; iso: string } //        /weeks/:iso/print
  | { kind: "settings" } //         /settings
  | { kind: "preferences" }; //     /preferences

const ISO_RE = /^\d{4}-W\d{2}$/;

export function parseRoute(pathname: string): Route {
  if (pathname === "/" || pathname === "") return { kind: "current" };
  if (pathname === "/weeks/new") return { kind: "new" };
  if (pathname === "/settings") return { kind: "settings" };
  if (pathname === "/preferences") return { kind: "preferences" };

  const parts = pathname.split("/").filter(Boolean);
  if (parts.length >= 2 && parts[0] === "weeks" && ISO_RE.test(parts[1])) {
    if (parts.length === 2) return { kind: "week", iso: parts[1] };
    if (parts.length === 3 && parts[2] === "print") return { kind: "print", iso: parts[1] };
  }
  return { kind: "current" };
}

export function routeToPath(route: Route): string {
  switch (route.kind) {
    case "current":
      return "/";
    case "week":
      return `/weeks/${route.iso}`;
    case "new":
      return "/weeks/new";
    case "print":
      return `/weeks/${route.iso}/print`;
    case "settings":
      return "/settings";
    case "preferences":
      return "/preferences";
  }
}

const listeners = new Set<() => void>();

function notify() {
  for (const l of listeners) l();
}

function subscribe(cb: () => void): () => void {
  if (listeners.size === 0) window.addEventListener("popstate", notify);
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
    if (listeners.size === 0) window.removeEventListener("popstate", notify);
  };
}

function getSnapshot(): string {
  return window.location.pathname;
}

export function navigate(route: Route | string, options?: { replace?: boolean }): void {
  const path = typeof route === "string" ? route : routeToPath(route);
  if (path === window.location.pathname) return;
  if (options?.replace) window.history.replaceState(null, "", path);
  else window.history.pushState(null, "", path);
  notify();
}

// goBack returns to the previous in-app URL if possible, otherwise navigates
// to the fallback (useful for modals opened directly from a bookmark).
export function goBack(fallback: Route = { kind: "current" }): void {
  if (window.history.length > 1) window.history.back();
  else navigate(fallback, { replace: true });
}

export function useRoute(): Route {
  const pathname = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
  return parseRoute(pathname);
}
