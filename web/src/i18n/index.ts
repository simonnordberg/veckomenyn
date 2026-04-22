import { useSyncExternalStore } from "react";
import { en } from "./en";
import { sv } from "./sv";

export type Lang = "sv" | "en";

const dictionaries: Record<Lang, Record<string, string>> = { sv, en };

let currentLang: Lang = "sv";
const listeners = new Set<() => void>();

export function getLang(): Lang {
  return currentLang;
}

export function setLang(l: Lang): void {
  if (l === currentLang) return;
  currentLang = l;
  for (const f of listeners) f();
}

function subscribe(f: () => void): () => void {
  listeners.add(f);
  return () => {
    listeners.delete(f);
  };
}

/**
 * t translates a key to the current language. If the key is missing, falls
 * back to English, then to the key itself. `params` interpolates {name} tokens.
 */
export function t(key: string, params?: Record<string, string | number>): string {
  const raw = dictionaries[currentLang][key] ?? dictionaries.en[key] ?? key;
  if (!params) return raw;
  return raw.replace(/\{(\w+)\}/g, (_, name) =>
    name in params ? String(params[name]) : `{${name}}`,
  );
}

/** useLang re-renders the component when the language changes. */
export function useLang(): Lang {
  return useSyncExternalStore(
    subscribe,
    () => currentLang,
    () => currentLang,
  );
}

// BCP 47 locale for the current language, used with Intl APIs.
export function locale(): string {
  switch (currentLang) {
    case "sv":
      return "sv-SE";
    case "en":
      return "en-US";
  }
}

// Short weekday name ("mån", "Thu") for a yyyy-mm-dd date in the current language.
export function formatWeekday(isoDate: string): string {
  const d = new Date(`${isoDate}T00:00:00`);
  return d.toLocaleDateString(locale(), { weekday: "short" });
}

// Seven short weekday labels starting on Monday, localised. Used by the
// calendar popup header strip.
export function shortWeekdaysMondayFirst(): string[] {
  // 2024-01-01 was a Monday.
  const base = new Date(2024, 0, 1);
  return Array.from({ length: 7 }, (_, i) => {
    const d = new Date(base);
    d.setDate(base.getDate() + i);
    return d.toLocaleDateString(locale(), { weekday: "short" });
  });
}

// "April 2026" / "april 2026" — month label for the calendar popup.
export function formatMonthYear(year: number, month0: number): string {
  return new Date(year, month0, 1).toLocaleDateString(locale(), {
    month: "long",
    year: "numeric",
  });
}
