import { useEffect, useState } from "react";
import { formatDayShort, formatPeriod, t, useLang } from "../i18n";
import type { WeekSummary } from "../lib/api";
import { EditableDate } from "./Editable";

type Props = {
  source: WeekSummary | null;
  onCancel: () => void;
  onConfirm: (startDate: string) => Promise<void> | void;
};

// DuplicatePlanDialog asks for the new plan's start date. Duration matches
// the source so no dinners drop when the period shifts. Default is
// source.start_date + 7 days, which covers the common "next week" case.
export function DuplicatePlanDialog({ source, onCancel, onConfirm }: Props) {
  useLang();
  const [startDate, setStartDate] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!source) return;
    setStartDate(addDays(source.start_date, 7));
    setBusy(false);
  }, [source]);

  if (!source) return null;

  const duration = daysBetween(source.start_date, source.end_date);
  const endDate = addDays(startDate, duration);
  const canConfirm = startDate !== "" && !busy;

  const confirm = async () => {
    if (!canConfirm) return;
    setBusy(true);
    try {
      await onConfirm(startDate);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-40 flex items-center justify-center bg-stone-900/40 backdrop-blur-[1px] dark:bg-black/60"
      onClick={onCancel}
      onKeyDown={(e) => e.key === "Escape" && onCancel()}
      role="button"
      tabIndex={-1}
      aria-label={t("duplicate.close")}
    >
      <div
        className="flex w-full max-w-sm flex-col rounded-lg border border-stone-200 bg-white shadow-xl dark:border-stone-800 dark:bg-stone-900"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={t("duplicate.title")}
      >
        <header className="border-b border-stone-200 px-5 py-3 dark:border-stone-800">
          <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
            {t("duplicate.title")}
          </h2>
          <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">
            {t("duplicate.source_prefix")} {formatPeriod(source.start_date, source.end_date)}
          </p>
        </header>
        <form
          className="flex flex-col gap-3 px-5 py-4"
          onSubmit={(e) => {
            e.preventDefault();
            void confirm();
          }}
        >
          <div className="flex flex-col gap-1">
            <span className="text-sm font-medium text-stone-700 dark:text-stone-300">
              {t("duplicate.start_date")}
            </span>
            <div className="rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100">
              <EditableDate
                value={startDate}
                label={t("duplicate.start_date")}
                onCommit={(v) => {
                  if (v) setStartDate(v);
                }}
              />
            </div>
          </div>
          <p className="text-xs text-stone-500 dark:text-stone-400">
            {startDate
              ? `${t("duplicate.new_period_prefix")} ${formatDayShort(startDate)} → ${formatDayShort(endDate)}`
              : ""}
          </p>
          <div className="mt-2 flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={onCancel}
              disabled={busy}
              className="rounded-md px-3 py-2 text-sm text-stone-600 hover:text-stone-900 disabled:cursor-not-allowed disabled:opacity-50 dark:text-stone-400 dark:hover:text-stone-100"
            >
              {t("duplicate.cancel")}
            </button>
            <button
              type="submit"
              disabled={!canConfirm}
              className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
            >
              {busy ? t("duplicate.submitting") : t("duplicate.confirm")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function addDays(base: string, offset: number): string {
  const d = new Date(`${base}T00:00:00`);
  d.setDate(d.getDate() + offset);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function daysBetween(startISO: string, endISO: string): number {
  const start = new Date(`${startISO}T00:00:00`);
  const end = new Date(`${endISO}T00:00:00`);
  return Math.round((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24));
}
