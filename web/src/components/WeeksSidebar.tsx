import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { listWeeks, type WeekSummary } from "../lib/api";

type Props = {
  selectedISO: string | null;
  onSelect: (iso: string) => void;
  refreshKey: number;
  onPlanNew?: () => void;
  planNewDisabled?: boolean;
};

export function WeeksSidebar({
  selectedISO,
  onSelect,
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
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-stone-200 bg-stone-100/50 dark:border-stone-800 dark:bg-stone-900/50">
      <header className="flex items-center justify-between border-b border-stone-200 px-4 py-3 dark:border-stone-800">
        <h2 className="font-serif text-base text-stone-900 dark:text-stone-100">
          {t("sidebar.history")}
        </h2>
        {onPlanNew && (
          <button
            type="button"
            onClick={onPlanNew}
            disabled={planNewDisabled}
            title={t("sidebar.new_week_title")}
            aria-label={t("sidebar.new_week_title")}
            className="flex items-center gap-1 rounded-md border border-stone-300 bg-white px-2 py-1 text-xs font-medium text-stone-700 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          >
            <svg width="12" height="12" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M10 3a1 1 0 0 1 1 1v5h5a1 1 0 1 1 0 2h-5v5a1 1 0 1 1-2 0v-5H4a1 1 0 1 1 0-2h5V4a1 1 0 0 1 1-1Z" />
            </svg>
            {t("sidebar.new_week")}
          </button>
        )}
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
            <li key={w.iso_week}>
              <button
                type="button"
                onClick={() => onSelect(w.iso_week)}
                className={`w-full rounded-md px-3 py-2 text-left text-sm transition-colors ${
                  selectedISO === w.iso_week
                    ? "bg-stone-200 text-stone-900 dark:bg-stone-800 dark:text-stone-50"
                    : "text-stone-700 hover:bg-stone-200/60 dark:text-stone-300 dark:hover:bg-stone-800/60"
                }`}
              >
                <div className="font-mono text-xs tabular-nums text-stone-700 dark:text-stone-300">
                  {w.iso_week}
                </div>
                <div className="mt-0.5 text-[11px] text-stone-500 tabular-nums dark:text-stone-400">
                  {w.start_date} · {w.dinner_count} {t("sidebar.dinners_short")} ·{" "}
                  <span className={statusColor(w.status)}>{t(`status.${w.status}`)}</span>
                </div>
              </button>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
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
    case "archived":
      return "text-stone-400 dark:text-stone-500";
  }
}
