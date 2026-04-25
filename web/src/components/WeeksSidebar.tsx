import { useEffect, useState } from "react";
import { formatPeriod, t, useLang } from "../i18n";
import { getVersion, listWeeks, type VersionInfo, type WeekSummary } from "../lib/api";

type Props = {
  open: boolean;
  onClose: () => void;
  selectedID: number | null;
  onSelect: (id: number) => void;
  onDuplicate: (source: WeekSummary) => void;
  onDelete: (source: WeekSummary) => void | Promise<void>;
  refreshKey: number;
  onPlanNew?: () => void;
  planNewDisabled?: boolean;
};

export function WeeksSidebar({
  open,
  onClose,
  selectedID,
  onSelect,
  onDuplicate,
  onDelete,
  refreshKey,
  onPlanNew,
  planNewDisabled,
}: Props) {
  useLang();
  const [weeks, setWeeks] = useState<WeekSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listWeeks()
      .then((rows) => {
        if (!cancelled) setWeeks(rows);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message);
      });
    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  return (
    <aside
      aria-label={t("sidebar.history")}
      className={`fixed inset-y-0 left-0 z-30 flex w-72 max-w-[85vw] flex-col border-r border-stone-200 bg-stone-100 transition-transform md:static md:z-auto md:w-64 md:max-w-none md:shrink-0 md:translate-x-0 md:bg-stone-100/50 dark:border-stone-800 dark:bg-stone-900 md:dark:bg-stone-900/50 ${open ? "translate-x-0" : "-translate-x-full"}`}
    >
      <header className="flex items-center justify-between gap-2 border-b border-stone-200 px-4 py-3 dark:border-stone-800">
        <h2 className="font-serif text-base text-stone-900 dark:text-stone-100">
          {t("sidebar.history")}
        </h2>
        <div className="flex items-center gap-1">
          {onPlanNew && (
            <button
              type="button"
              onClick={onPlanNew}
              disabled={planNewDisabled}
              title={t("sidebar.new_week_title")}
              aria-label={t("sidebar.new_week_title")}
              className="flex items-center gap-1 rounded-md border border-stone-300 bg-white px-2 py-1 text-xs font-medium text-stone-700 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
            >
              <svg
                width="12"
                height="12"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path d="M10 3a1 1 0 0 1 1 1v5h5a1 1 0 1 1 0 2h-5v5a1 1 0 1 1-2 0v-5H4a1 1 0 1 1 0-2h5V4a1 1 0 0 1 1-1Z" />
              </svg>
              {t("sidebar.new_week")}
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            aria-label={t("topbar.close_chat")}
            className="flex h-7 w-7 items-center justify-center rounded-md text-stone-500 hover:bg-stone-200 md:hidden dark:text-stone-400 dark:hover:bg-stone-700"
          >
            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
            </svg>
          </button>
        </div>
      </header>
      <nav className="mt-2 flex-1 overflow-y-auto px-2 pb-3">
        {error && <div className="px-2 py-2 text-xs text-red-600 dark:text-red-400">{error}</div>}
        {weeks.length === 0 && !error && (
          <p className="px-2 py-4 text-xs text-stone-500 dark:text-stone-400">
            {t("sidebar.empty")}
          </p>
        )}
        <ul className="space-y-0.5">
          {weeks.map((w) => (
            <li key={w.id} className="group relative">
              <button
                type="button"
                onClick={() => onSelect(w.id)}
                className={`w-full rounded-md px-3 py-2 pr-9 text-left text-sm transition-colors ${
                  selectedID === w.id
                    ? "bg-stone-200 text-stone-900 dark:bg-stone-800 dark:text-stone-50"
                    : "text-stone-700 hover:bg-stone-200/60 dark:text-stone-300 dark:hover:bg-stone-800/60"
                }`}
              >
                <div className="text-xs tabular-nums text-stone-700 dark:text-stone-300">
                  {formatPeriod(w.start_date, w.end_date)}
                </div>
                <div className="mt-0.5 text-[11px] text-stone-500 tabular-nums dark:text-stone-400">
                  {w.dinner_count} {t("sidebar.dinners_short")} ·{" "}
                  <span className={statusColor(w.status)}>{t(`status.${w.status}`)}</span>
                </div>
              </button>
              <div className="absolute right-1.5 top-1.5 flex gap-0.5 opacity-100 transition-opacity focus-within:opacity-100 md:opacity-0 md:group-hover:opacity-100">
                <button
                  type="button"
                  onClick={() => onDuplicate(w)}
                  title={t("sidebar.duplicate")}
                  aria-label={t("sidebar.duplicate")}
                  className="flex h-6 w-6 items-center justify-center rounded text-stone-500 hover:bg-stone-200 hover:text-stone-900 dark:text-stone-400 dark:hover:bg-stone-700 dark:hover:text-stone-100"
                >
                  <svg
                    width="12"
                    height="12"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path d="M7 3a2 2 0 0 0-2 2v1H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2v-1h1a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2H7Zm6 13v1a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h1v7a2 2 0 0 0 2 2h6Zm3-3a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v8Z" />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => void onDelete(w)}
                  title={t("sidebar.delete")}
                  aria-label={t("sidebar.delete")}
                  className="flex h-6 w-6 items-center justify-center rounded text-stone-500 hover:bg-red-100 hover:text-red-700 dark:text-stone-400 dark:hover:bg-red-950/60 dark:hover:text-red-300"
                >
                  <svg
                    width="12"
                    height="12"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    aria-hidden="true"
                  >
                    <path d="M8 2a1 1 0 0 0-.894.553L6.382 4H4a1 1 0 0 0 0 2v10a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V6a1 1 0 1 0 0-2h-2.382l-.724-1.447A1 1 0 0 0 12 2H8Zm1 6a1 1 0 0 1 2 0v7a1 1 0 1 1-2 0V8ZM7 7a1 1 0 0 1 1 1v7a1 1 0 1 1-2 0V8a1 1 0 0 1 1-1Zm6 0a1 1 0 0 1 1 1v7a1 1 0 1 1-2 0V8a1 1 0 0 1 1-1Z" />
                  </svg>
                </button>
              </div>
            </li>
          ))}
        </ul>
      </nav>
      <VersionFooter />
    </aside>
  );
}

function VersionFooter() {
  const [info, setInfo] = useState<VersionInfo | null>(null);
  useEffect(() => {
    let cancelled = false;
    getVersion()
      .then((v) => {
        if (!cancelled) setInfo(v);
      })
      .catch(() => {
        /* silently hide footer on failure */
      });
    return () => {
      cancelled = true;
    };
  }, []);
  if (!info) return null;
  // Stable releases get a link to the GitHub release notes; dev builds
  // (version "dev" or a git-describe like "v0.1.0-3-gabc1234") just show
  // the string so users know they're not on a published tag.
  const isRelease = /^\d+\.\d+\.\d+$/.test(info.version);
  const label = isRelease ? `v${info.version}` : info.version;
  const className = "block px-4 py-2 text-xs tabular-nums text-stone-500 dark:text-stone-500";
  if (!isRelease) {
    return (
      <footer className={`border-t border-stone-200 dark:border-stone-800 ${className}`}>
        {label}
      </footer>
    );
  }
  return (
    <footer className="border-t border-stone-200 dark:border-stone-800">
      <a
        href={`https://github.com/simonnordberg/veckomenyn/releases/tag/v${info.version}`}
        target="_blank"
        rel="noreferrer"
        className={`${className} hover:text-stone-600 dark:hover:text-stone-400`}
      >
        {label}
      </a>
    </footer>
  );
}

function statusColor(s: WeekSummary["status"]): string {
  switch (s) {
    case "draft":
      return "text-stone-500 dark:text-stone-400";
    case "cart_built":
      return "text-amber-700 dark:text-amber-400";
    case "ordered":
      return "text-emerald-700 dark:text-emerald-400";
  }
}
