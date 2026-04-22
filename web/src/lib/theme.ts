import { useSyncExternalStore } from "react";

export type Theme = "system" | "light" | "dark";

const STORAGE_KEY = "wf:theme";

function read(): Theme {
  if (typeof window === "undefined") return "system";
  const v = window.localStorage.getItem(STORAGE_KEY);
  return v === "light" || v === "dark" ? v : "system";
}

// Resolves "system" against prefers-color-scheme. Used to decide whether to
// apply the .dark class to <html>.
export function resolveTheme(t: Theme): "light" | "dark" {
  if (t !== "system") return t;
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function apply(t: Theme) {
  if (typeof document === "undefined") return;
  const resolved = resolveTheme(t);
  document.documentElement.classList.toggle("dark", resolved === "dark");
  document.documentElement.style.colorScheme = resolved;
}

const listeners = new Set<() => void>();

export function setTheme(t: Theme) {
  if (typeof window !== "undefined") {
    if (t === "system") window.localStorage.removeItem(STORAGE_KEY);
    else window.localStorage.setItem(STORAGE_KEY, t);
  }
  apply(t);
  for (const fn of listeners) fn();
}

// Call once at boot (from main.tsx) so the initial paint is correct.
export function initTheme() {
  apply(read());
  if (typeof window !== "undefined" && window.matchMedia) {
    // If the user picks "system", re-apply whenever the OS flips.
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
      if (read() === "system") apply("system");
      for (const fn of listeners) fn();
    });
  }
}

function subscribe(fn: () => void) {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export function useTheme() {
  const t = useSyncExternalStore<Theme>(subscribe, read, () => "system");
  return { theme: t, resolved: resolveTheme(t) };
}
