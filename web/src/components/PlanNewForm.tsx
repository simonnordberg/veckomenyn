import { type FormEvent, useEffect, useMemo, useState } from "react";
import { t, useLang } from "../i18n";
import { getSettings } from "../lib/api";
import { EditableDate } from "./Editable";

type Props = {
  onSubmit: (prompt: string) => void;
  busy: boolean;
};

const DEFAULT_DINNERS = 7;

export function PlanNewForm({ onSubmit, busy }: Props) {
  useLang();
  const [startDate, setStartDate] = useState(() => addDays(today(), 1));
  const [dinners, setDinners] = useState(DEFAULT_DINNERS);
  const [notes, setNotes] = useState("");

  useEffect(() => {
    getSettings()
      .then((s) => setDinners(s.default_dinners))
      .catch(() => {
        /* keep fallback */
      });
  }, []);

  const endDate = useMemo(() => addDays(startDate, dinners - 1), [startDate, dinners]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    const parts = [t("plan.prompt", { dinners, start: startDate, end: endDate })];
    if (notes.trim()) parts.push(notes.trim());
    onSubmit(parts.join(" "));
  };

  return (
    <div className="mx-auto flex max-w-xl flex-col px-6 py-12">
      <h1 className="font-serif text-3xl tracking-tight text-stone-900 dark:text-stone-100">
        {t("plan.title")}
      </h1>
      <p className="mt-2 text-sm text-stone-600 dark:text-stone-400">{t("plan.subtitle")}</p>
      <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
        <Field label={t("plan.start_date")}>
          <div className="rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100">
            <EditableDate
              value={startDate}
              label={t("plan.start_date")}
              onCommit={(v) => {
                if (v) setStartDate(v);
              }}
            />
          </div>
          <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">
            {t("plan.menu_runs_through", { end: endDate })}
          </p>
        </Field>
        <Field label={t("plan.num_dinners")}>
          <input
            type="number"
            min={1}
            max={14}
            value={dinners}
            onChange={(e) => setDinners(Number.parseInt(e.target.value, 10) || 7)}
            className="w-32 rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
          />
        </Field>
        <Field label={t("plan.notes")} hint={t("plan.notes_hint")}>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={3}
            placeholder={t("plan.notes_placeholder")}
            className="w-full resize-none rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
          />
        </Field>
        <button
          type="submit"
          disabled={busy}
          className="mt-2 self-start rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
        >
          {busy ? t("plan.submitting") : t("plan.submit")}
        </button>
      </form>
    </div>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-sm font-medium text-stone-700 dark:text-stone-300">{label}</span>
      {children}
      {hint && <span className="text-xs text-stone-500 dark:text-stone-400">{hint}</span>}
    </label>
  );
}

function today(): string {
  return iso(new Date());
}

function addDays(base: string, offset: number): string {
  const d = new Date(`${base}T00:00:00`);
  d.setDate(d.getDate() + offset);
  return iso(d);
}

function iso(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}
