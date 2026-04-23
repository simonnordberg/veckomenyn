import { useEffect, useState } from "react";
import { formatWeekday, t, useLang } from "../i18n";
import { getWeek, type WeekDetail } from "../lib/api";
import { Markdown } from "./Markdown";

type Props = {
  iso: string;
};

export function PrintableWeek({ iso }: Props) {
  useLang();
  const [week, setWeek] = useState<WeekDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Paper is always light. Suspend any dark-mode preference while this
  // view is mounted so prose-invert stops flipping recipe text to
  // light-on-white.
  useEffect(() => {
    const html = document.documentElement;
    const hadDark = html.classList.contains("dark");
    html.classList.remove("dark");
    const prevScheme = html.style.colorScheme;
    html.style.colorScheme = "light";
    return () => {
      if (hadDark) html.classList.add("dark");
      html.style.colorScheme = prevScheme;
    };
  }, []);

  useEffect(() => {
    getWeek(iso)
      .then(setWeek)
      .catch((e: Error) => setError(e.message));
  }, [iso]);

  if (error) {
    return <p className="m-8 text-sm text-red-700">{error}</p>;
  }
  if (!week) {
    return <p className="m-8 text-sm text-stone-500">{t("print.loading")}</p>;
  }

  return (
    <div className="mx-auto max-w-3xl bg-white px-8 py-10 text-stone-900 print:px-0 print:py-0">
      {/* Print controls, hidden when printing */}
      <div className="mb-6 flex items-center justify-between border-b border-stone-200 pb-4 print:hidden">
        <div className="text-xs text-stone-500">{t("print.preview_hint")}</div>
        <button
          type="button"
          onClick={() => window.print()}
          className="rounded-md bg-stone-900 px-3 py-1.5 text-sm font-medium text-stone-50 hover:bg-stone-800"
        >
          {t("print.print_button")}
        </button>
      </div>

      <header className="mb-6">
        <h1 className="font-serif text-4xl tracking-tight text-stone-900">{week.iso_week}</h1>
        <p className="mt-1 text-sm text-stone-600 tabular-nums">
          {week.start_date} → {week.end_date}
        </p>
        {week.notes_md && (
          <p className="mt-3 whitespace-pre-wrap text-sm italic text-stone-700">{week.notes_md}</p>
        )}
      </header>

      {week.exceptions.length > 0 && (
        <section className="mb-6 rounded border border-stone-300 px-4 py-2 text-sm">
          <h2 className="text-xs font-medium uppercase tracking-wide text-stone-500">
            {t("week.this_week")}
          </h2>
          <ul className="mt-1 list-disc pl-5">
            {week.exceptions.map((e) => (
              <li key={e.id}>
                <span className="font-mono text-xs uppercase tracking-wide text-stone-600">
                  {e.kind}
                </span>{" "}
                {e.description}
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="mb-8">
        <h2 className="mb-3 font-serif text-xl text-stone-900">{t("print.overview")}</h2>
        <table className="w-full border-collapse text-sm">
          <colgroup>
            <col className="w-[22%]" />
            <col />
            <col className="w-[6%]" />
            <col className="w-[22%]" />
          </colgroup>
          <thead>
            <tr className="border-b border-stone-300">
              <th className="py-1.5 pr-3 text-left font-medium">{t("print.day")}</th>
              <th className="py-1.5 pr-3 text-left font-medium">{t("print.dinner")}</th>
              <th className="py-1.5 pr-3 text-right font-medium">{t("print.pers")}</th>
              <th className="py-1.5 text-left font-medium">{t("print.source")}</th>
            </tr>
          </thead>
          <tbody>
            {week.dinners.map((d) => (
              <tr key={d.id} className="border-b border-stone-200">
                <td className="py-1.5 pr-3 font-mono text-xs tabular-nums">
                  {d.day_date} · {formatWeekday(d.day_date)}
                </td>
                <td className="py-1.5 pr-3">{d.dish_name || "-"}</td>
                <td className="py-1.5 pr-3 text-right tabular-nums">{d.servings}</td>
                <td className="py-1.5 text-xs text-stone-600">
                  {Object.entries(d.sourcing || {})
                    .map(([k, v]) => `${k}: ${v}`)
                    .join("; ") || "-"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="flex flex-col gap-8">
        {week.dinners.map((d) => (
          <article key={d.id} className="break-inside-avoid">
            <header className="mb-2 border-b border-stone-300 pb-1">
              <div className="font-mono text-xs uppercase tracking-wide text-stone-500 tabular-nums">
                {d.day_date} · {formatWeekday(d.day_date)}
                {d.cuisine && <> · {d.cuisine}</>} · {d.servings} {t("dinner.servings")}
              </div>
              <h3 className="mt-0.5 font-serif text-2xl tracking-tight text-stone-900">
                {d.dish_name || t("dinner.untitled")}
              </h3>
            </header>
            {Object.keys(d.sourcing || {}).length > 0 && (
              <p className="mb-2 text-xs text-stone-600">
                <strong>{t("print.source")}:</strong>{" "}
                {Object.entries(d.sourcing)
                  .map(([k, v]) => `${k}: ${v}`)
                  .join("; ")}
              </p>
            )}
            {d.notes && <p className="mb-2 text-sm italic text-stone-700">{d.notes}</p>}
            {d.recipe_md && (
              <Markdown source={d.recipe_md} stripLeadingHeading={d.dish_name} headingShift={2} />
            )}
          </article>
        ))}
      </section>

      {week.retrospectives.length > 0 && (
        <section className="mt-8 border-t border-stone-300 pt-4">
          <h2 className="mb-2 font-serif text-xl text-stone-900">{t("week.retrospective")}</h2>
          {week.retrospectives.map((r) => (
            <Markdown key={r.id} source={r.notes_md} />
          ))}
        </section>
      )}
    </div>
  );
}
