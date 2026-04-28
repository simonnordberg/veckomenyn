import { useEffect, useState } from "react";
import { formatPeriod, t, useLang } from "../i18n";
import { listWeeks, type WeekSummary } from "../lib/api";
import { toast } from "../lib/toast";

type Props = {
  refreshKey: number;
  onPlanNew: () => void;
  onOpenWeek: (id: number) => void;
};

export function HomeView({ refreshKey, onPlanNew, onOpenWeek }: Props) {
  useLang();
  const [weeks, setWeeks] = useState<WeekSummary[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    listWeeks()
      .then((rows) => {
        if (!cancelled) setWeeks(rows);
      })
      .catch((e: Error) => {
        if (!cancelled) toast.error(e.message);
      });
    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  const empty = weeks !== null && weeks.length === 0;

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-8 sm:px-6 sm:py-12">
      <header className="flex flex-col items-start gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="font-serif text-3xl tracking-tight text-stone-900 dark:text-stone-100">
            {t("home.title")}
          </h1>
          <p className="mt-1 text-sm text-stone-600 dark:text-stone-400">{t("home.subtitle")}</p>
        </div>
        <button
          type="button"
          onClick={onPlanNew}
          className="shrink-0 rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 shadow-sm hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
        >
          {t("home.plan_new")}
        </button>
      </header>

      {empty && <EmptyHero onPlanNew={onPlanNew} />}

      {weeks && weeks.length > 0 && (
        <section>
          <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-stone-500 dark:text-stone-400">
            {t("home.your_weeks")}
          </h2>
          <ul className="grid gap-2 sm:grid-cols-2">
            {weeks.map((w) => (
              <li key={w.id}>
                <button
                  type="button"
                  onClick={() => onOpenWeek(w.id)}
                  className="flex w-full flex-col items-start gap-1 rounded-md border border-stone-200 bg-white px-4 py-3 text-left transition-colors hover:border-stone-300 hover:bg-stone-50 dark:border-stone-800 dark:bg-stone-900 dark:hover:border-stone-700 dark:hover:bg-stone-800"
                >
                  <span className="font-mono text-xs tabular-nums text-stone-500 dark:text-stone-400">
                    {formatPeriod(w.start_date, w.end_date)}
                  </span>
                  <span className="text-sm text-stone-800 dark:text-stone-200">
                    {w.dinner_count} {t("sidebar.dinners_short")}
                  </span>
                  <span
                    className={`mt-0.5 text-[11px] font-medium uppercase tracking-wide ${statusColor(w.status)}`}
                  >
                    {t(`status.${w.status}`)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

function EmptyHero({ onPlanNew }: { onPlanNew: () => void }) {
  return (
    <section className="rounded-lg border border-dashed border-stone-300 bg-stone-50 px-6 py-10 text-stone-700 dark:border-stone-700 dark:bg-stone-900/50 dark:text-stone-300">
      <h2 className="font-serif text-xl text-stone-900 dark:text-stone-100">
        {t("home.empty_title")}
      </h2>
      <p className="mt-2 max-w-prose text-sm leading-relaxed">{t("home.empty_body")}</p>
      <ol className="mt-4 list-decimal space-y-1 pl-5 text-sm">
        <li>{t("home.loop_step_plan")}</li>
        <li>{t("home.loop_step_cart")}</li>
        <li>{t("home.loop_step_order")}</li>
        <li>{t("home.loop_step_retro")}</li>
      </ol>
      <button
        type="button"
        onClick={onPlanNew}
        className="mt-5 rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
      >
        {t("home.plan_first")}
      </button>
    </section>
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
