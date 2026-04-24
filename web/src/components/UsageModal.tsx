import { useEffect, useMemo, useState } from "react";
import { locale, t, useLang } from "../i18n";
import { getUsageSummary, type UsageSummary } from "../lib/api";

type Props = {
  open: boolean;
  onClose: () => void;
};

export function UsageModal({ open, onClose }: Props) {
  useLang();
  const [summary, setSummary] = useState<UsageSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setSummary(null);
    setError(null);
    let cancelled = false;
    getUsageSummary()
      .then((s) => {
        if (!cancelled) setSummary(s);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message);
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-30 flex items-center justify-center bg-stone-900/40 p-3 backdrop-blur-[1px] sm:p-6 dark:bg-black/60"
      onClick={onClose}
      onKeyDown={(e) => e.key === "Escape" && onClose()}
      role="button"
      tabIndex={-1}
      aria-label={t("usage.close")}
    >
      <div
        className="flex max-h-[90vh] w-full max-w-3xl flex-col rounded-lg border border-stone-200 bg-white shadow-xl dark:border-stone-800 dark:bg-stone-900"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={t("usage.title")}
      >
        <header className="flex shrink-0 items-center justify-between gap-3 border-b border-stone-200 px-5 py-3 dark:border-stone-800">
          <div className="flex items-baseline gap-3">
            <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
              {t("usage.title")}
            </h2>
            <span className="text-xs text-stone-500 dark:text-stone-400">
              {t("usage.subtitle")}
            </span>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-stone-500 hover:bg-stone-100 dark:hover:bg-stone-800"
            aria-label={t("usage.close")}
          >
            <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
            </svg>
          </button>
        </header>
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {error && (
            <div className="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
              {error}
            </div>
          )}
          {!summary && !error && <p className="text-sm text-stone-500">{t("usage.loading")}</p>}
          {summary && <UsageContent s={summary} />}
        </div>
      </div>
    </div>
  );
}

function UsageContent({ s }: { s: UsageSummary }) {
  const empty = s.total.calls === 0;
  const cacheHitRate = useMemo(() => {
    const served = s.total.cache_read_input_tokens;
    const allInput =
      s.total.input_tokens + s.total.cache_creation_input_tokens + s.total.cache_read_input_tokens;
    return allInput === 0 ? 0 : served / allInput;
  }, [s]);

  if (empty) {
    return <p className="text-sm text-stone-500">{t("usage.empty")}</p>;
  }

  return (
    <div className="flex flex-col gap-6">
      <section className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Stat label={t("usage.total_cost")} value={formatUSD(s.total.cost_usd)} emphasis />
        <Stat label={t("usage.total_calls")} value={formatCount(s.total.calls)} />
        <Stat
          label={t("usage.cache_hit_rate")}
          value={formatPercent(cacheHitRate)}
          hint={t("usage.cache_hit_hint")}
        />
        <Stat label={t("usage.total_input")} value={formatTokens(s.total.input_tokens)} />
        <Stat label={t("usage.total_output")} value={formatTokens(s.total.output_tokens)} />
        <Stat
          label={t("usage.total_cache_read")}
          value={formatTokens(s.total.cache_read_input_tokens)}
        />
      </section>

      <Section title={t("usage.by_model")}>
        <Table
          head={[
            t("usage.col.model"),
            t("usage.col.calls"),
            t("usage.col.input"),
            t("usage.col.cache_write"),
            t("usage.col.cache_read"),
            t("usage.col.output"),
            t("usage.col.cost"),
          ]}
          rows={s.by_model.map((m) => [
            { text: m.model, mono: true },
            { text: formatCount(m.calls), align: "right" },
            { text: formatTokens(m.input_tokens), align: "right" },
            { text: formatTokens(m.cache_creation_input_tokens), align: "right" },
            { text: formatTokens(m.cache_read_input_tokens), align: "right" },
            { text: formatTokens(m.output_tokens), align: "right" },
            { text: formatUSD(m.cost_usd), align: "right", emphasis: true },
          ])}
        />
      </Section>

      {s.by_week.length > 0 && (
        <Section title={t("usage.by_week")}>
          <Table
            head={[t("usage.col.plan"), t("usage.col.calls"), t("usage.col.cost")]}
            rows={s.by_week.map((v) => [
              { text: v.iso_week || `#${v.week_id}`, mono: true },
              { text: formatCount(v.calls), align: "right" },
              { text: formatUSD(v.cost_usd), align: "right", emphasis: true },
            ])}
          />
        </Section>
      )}

      {s.recent_conversations.length > 0 && (
        <Section title={t("usage.recent_conversations")}>
          <Table
            head={[
              t("usage.col.conversation"),
              t("usage.col.plan"),
              t("usage.col.calls"),
              t("usage.col.cost"),
            ]}
            rows={s.recent_conversations.map((c) => [
              { text: c.title || `#${c.conversation_id}`, truncate: true },
              { text: c.iso_week || "", mono: true },
              { text: formatCount(c.calls), align: "right" },
              { text: formatUSD(c.cost_usd), align: "right", emphasis: true },
            ])}
          />
        </Section>
      )}

      <Section title={t("usage.by_day")}>
        <DailyBars days={s.by_day} />
      </Section>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-2 font-serif text-sm text-stone-700 dark:text-stone-300">{title}</h3>
      {children}
    </section>
  );
}

function Stat({
  label,
  value,
  hint,
  emphasis,
}: {
  label: string;
  value: string;
  hint?: string;
  emphasis?: boolean;
}) {
  return (
    <div className="rounded-md border border-stone-200 bg-stone-50 px-3 py-2 dark:border-stone-800 dark:bg-stone-900/60">
      <div className="text-xs text-stone-500 dark:text-stone-400">{label}</div>
      <div
        className={`font-serif ${emphasis ? "text-lg text-stone-900 dark:text-stone-100" : "text-base text-stone-800 dark:text-stone-200"}`}
      >
        {value}
      </div>
      {hint && <div className="mt-0.5 text-[11px] text-stone-500 dark:text-stone-500">{hint}</div>}
    </div>
  );
}

type Cell = {
  text: string;
  align?: "right";
  mono?: boolean;
  emphasis?: boolean;
  truncate?: boolean;
};

function Table({ head, rows }: { head: string[]; rows: Cell[][] }) {
  return (
    <div className="overflow-x-auto rounded-md border border-stone-200 dark:border-stone-800">
      <table className="w-full text-sm">
        <thead className="bg-stone-50 text-stone-600 dark:bg-stone-900/60 dark:text-stone-400">
          <tr>
            {head.map((h, i) => (
              <th
                key={h}
                className={`px-3 py-2 text-left font-medium ${i > 0 ? "text-right" : ""}`}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, ri) => (
            <tr
              key={ri}
              className="border-t border-stone-200 text-stone-800 dark:border-stone-800 dark:text-stone-200"
            >
              {row.map((c, ci) => (
                <td
                  key={ci}
                  className={[
                    "px-3 py-2",
                    c.align === "right" ? "text-right tabular-nums" : "",
                    c.mono ? "font-mono text-xs" : "",
                    c.emphasis ? "font-medium" : "",
                    c.truncate ? "max-w-[14rem] truncate" : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  title={c.truncate ? c.text : undefined}
                >
                  {c.text}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DailyBars({ days }: { days: { date: string; cost_usd: number; calls: number }[] }) {
  const max = days.reduce((m, d) => Math.max(m, d.cost_usd), 0);
  if (max === 0) {
    return <p className="text-xs text-stone-500">{t("usage.empty")}</p>;
  }
  return (
    <ul className="flex flex-col gap-1">
      {days.map((d) => {
        const pct = (d.cost_usd / max) * 100;
        return (
          <li key={d.date} className="grid grid-cols-[6rem_1fr_5rem] items-center gap-2 text-xs">
            <span className="text-stone-500 dark:text-stone-400">{formatDate(d.date)}</span>
            <span
              className="flex h-3 items-center rounded-sm bg-stone-100 dark:bg-stone-800"
              aria-hidden
            >
              <span
                className="h-full rounded-sm bg-stone-800 dark:bg-stone-200"
                style={{ width: `${pct}%` }}
              />
            </span>
            <span className="text-right tabular-nums text-stone-700 dark:text-stone-300">
              {formatUSD(d.cost_usd)}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

function formatUSD(v: number): string {
  return new Intl.NumberFormat(locale(), {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: v < 1 ? 4 : 2,
  }).format(v);
}

function formatTokens(n: number): string {
  return new Intl.NumberFormat(locale()).format(n);
}

function formatCount(n: number): string {
  return new Intl.NumberFormat(locale()).format(n);
}

function formatPercent(v: number): string {
  return new Intl.NumberFormat(locale(), {
    style: "percent",
    maximumFractionDigits: 1,
  }).format(v);
}

function formatDate(isoDate: string): string {
  const d = new Date(`${isoDate}T00:00:00`);
  return d.toLocaleDateString(locale(), { month: "short", day: "numeric" });
}
