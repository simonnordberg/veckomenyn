import { useEffect, useMemo, useRef, useState } from "react";
import { formatWeekday, t, useLang } from "../i18n";
import {
  type CartItem,
  clearDinnerRating,
  type Dinner,
  type DinnerRating,
  type Exception,
  setDinnerRating,
  setWeekRetrospective,
  type WeekDetail,
  type WeekPatch,
} from "../lib/api";
import { EditableDate, EditableText } from "./Editable";
import { Markdown } from "./Markdown";

type Props = {
  week: WeekDetail;
  activeDayDate: string | null; // day the agent is currently writing a dinner for, for subtle highlight
  onAction: (action: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
  onRefetch: () => void;
};

export function WeekView({ week, activeDayDate, onAction, onPatch, onRefetch }: Props) {
  useLang();
  const dinners = useMemo(() => groupByDay(week.dinners), [week.dinners]);
  // Rating & retrospective live on weeks you've actually cooked through.
  // Hiding them while planning keeps the card tidy and reflects the lifecycle.
  const rateable = week.status === "ordered";

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 px-6 py-8">
      <WeekHeader week={week} onAction={onAction} onPatch={onPatch} />
      {week.exceptions.length > 0 && <Exceptions items={week.exceptions} />}
      <section className="flex flex-col gap-3">
        {dinners.length === 0 ? (
          <div className="rounded-lg border border-dashed border-stone-300 bg-stone-50 px-6 py-10 text-center text-stone-500 dark:border-stone-700 dark:bg-stone-900/50 dark:text-stone-400">
            <p>{t("week.no_dinners")}</p>
            <button
              type="button"
              onClick={() =>
                onAction(
                  t("week.plan_dinners_prompt", {
                    start: week.start_date,
                    end: week.end_date,
                  }),
                )
              }
              className="mt-3 rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm hover:bg-stone-100 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
            >
              {t("week.plan_dinners")}
            </button>
          </div>
        ) : (
          dinners.map((d) => (
            <DinnerCard
              key={d.id}
              dinner={d}
              dimmed={activeDayDate !== null && activeDayDate !== d.day_date}
              active={activeDayDate === d.day_date}
              rateable={rateable}
              onAction={onAction}
              onRatingChanged={onRefetch}
            />
          ))
        )}
      </section>
      {week.cart_items.length > 0 && <CartSection items={week.cart_items} />}
      <Lifecycle week={week} onAction={onAction} onPatch={onPatch} />
      {rateable && <WeekRetrospective week={week} />}
    </div>
  );
}

function Lifecycle({
  week,
  onAction,
  onPatch,
}: {
  week: WeekDetail;
  onAction: (a: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
}) {
  const hasRetro = week.retrospectives.length > 0;

  // Next recommended step based on status.
  const next: { label: string; run: () => void } | null = (() => {
    switch (week.status) {
      case "draft":
        return {
          label: t("lifecycle.build_cart"),
          run: () =>
            onAction(
              t("lifecycle.build_cart_prompt", {
                start: week.start_date,
                end: week.end_date,
              }),
            ),
        };
      case "cart_built":
        return {
          label: t("lifecycle.mark_ordered"),
          run: () =>
            void onPatch({
              status: "ordered",
              order_date: week.order_date ?? today(),
            }),
        };
      case "ordered":
        if (!hasRetro)
          return {
            label: t("lifecycle.record_retrospective"),
            run: () =>
              onAction(
                t("lifecycle.retrospective_prompt", {
                  start: week.start_date,
                  end: week.end_date,
                }),
              ),
          };
        return null;
    }
  })();

  return (
    <section className="rounded-md border border-stone-200 bg-white px-4 py-3 dark:border-stone-800 dark:bg-stone-900">
      <div className="flex flex-wrap items-center gap-2">
        <div className="mr-auto text-xs text-stone-500 dark:text-stone-400">
          {t("lifecycle.current")}:{" "}
          <span className="font-medium text-stone-800 dark:text-stone-200">
            {t(`status.${week.status}`)}
          </span>
        </div>
        {next && (
          <button
            type="button"
            onClick={next.run}
            className="rounded-md bg-stone-900 px-3 py-1.5 text-xs font-medium text-stone-50 hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
          >
            {next.label}
          </button>
        )}
        <a
          href="https://www.willys.se/"
          target="_blank"
          rel="noreferrer"
          className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
        >
          {t("lifecycle.open_willys")} ↗
        </a>
        {!hasRetro &&
          week.status !== "draft" &&
          next?.label !== t("lifecycle.record_retrospective") && (
            <button
              type="button"
              onClick={() =>
                onAction(
                  t("lifecycle.retrospective_prompt", {
                    start: week.start_date,
                    end: week.end_date,
                  }),
                )
              }
              className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
            >
              {t("lifecycle.record_retrospective")}
            </button>
          )}
        <StatusMenu current={week.status} onPick={(s) => void onPatch({ status: s })} />
      </div>
    </section>
  );
}

function StatusMenu({
  current,
  onPick,
}: {
  current: WeekDetail["status"];
  onPick: (s: WeekDetail["status"]) => void;
}) {
  const [open, setOpen] = useState(false);
  const statuses: WeekDetail["status"][] = ["draft", "cart_built", "ordered"];
  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
      >
        {t("lifecycle.set_status")} ▾
      </button>
      {open && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setOpen(false)}
            onKeyDown={() => setOpen(false)}
            role="button"
            tabIndex={-1}
            aria-label="close menu"
          />
          <div className="absolute right-0 top-full z-20 mt-1 min-w-40 rounded-md border border-stone-200 bg-white p-1 shadow-lg dark:border-stone-700 dark:bg-stone-800">
            {statuses.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => {
                  setOpen(false);
                  if (s !== current) onPick(s);
                }}
                className={`block w-full rounded px-3 py-1.5 text-left text-xs ${
                  s === current
                    ? "bg-stone-100 text-stone-900 dark:bg-stone-700 dark:text-stone-50"
                    : "text-stone-700 hover:bg-stone-50 dark:text-stone-300 dark:hover:bg-stone-700"
                }`}
              >
                {t(`status.${s}`)}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function CartSection({ items }: { items: CartItem[] }) {
  // Default open if the cart has items. An empty collapsed section with
  // "(0)" next to it isn't telling you anything; a populated one hiding
  // 74 rows behind a small caret isn't either.
  const [open, setOpen] = useState(items.length > 0);
  const total = items.length;
  const showNote = items.some((i) => i.reason_md && i.reason_md.trim() !== "");
  const runningTotal = items.reduce((acc, i) => acc + (i.snapshot?.line_total ?? 0), 0);
  return (
    <section className="rounded-md border border-stone-200 bg-white dark:border-stone-800 dark:bg-stone-900">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between px-4 py-3 text-left"
      >
        <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
          {t("cart.title")}{" "}
          <span className="ml-2 text-sm text-stone-500 tabular-nums dark:text-stone-400">
            ({total})
          </span>
        </h2>
        <span className="flex items-center gap-3 text-sm text-stone-500 tabular-nums dark:text-stone-400">
          {runningTotal > 0 && <span>{formatKronor(runningTotal)}</span>}
          <span className="text-stone-400 dark:text-stone-500">{open ? "▾" : "▸"}</span>
        </span>
      </button>
      {open && (
        <div className="border-t border-stone-100 dark:border-stone-800">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-xs uppercase tracking-wide text-stone-500 dark:bg-stone-800/50 dark:text-stone-400">
              <tr>
                <th className="px-4 py-2 text-left font-medium">{t("cart.product")}</th>
                <th className="px-4 py-2 text-right font-medium">{t("cart.qty")}</th>
                <th className="px-4 py-2 text-right font-medium">{t("cart.price")}</th>
                {showNote && (
                  <th className="px-4 py-2 text-left font-medium">{t("cart.reason")}</th>
                )}
              </tr>
            </thead>
            <tbody>
              {items.map((item) => {
                const name = item.snapshot?.name ?? "";
                const lineTotal = item.snapshot?.line_total;
                return (
                  <tr key={item.id} className="border-t border-stone-100 dark:border-stone-800">
                    <td className="px-4 py-1.5 text-xs">
                      {name ? (
                        <>
                          <div className="text-stone-800 dark:text-stone-100">{name}</div>
                          <div className="font-mono text-[10px] tabular-nums text-stone-400 dark:text-stone-500">
                            {item.product_code}
                          </div>
                        </>
                      ) : (
                        <div className="font-mono tabular-nums text-stone-700 dark:text-stone-300">
                          {item.product_code}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-1.5 text-right font-mono text-xs tabular-nums text-stone-700 dark:text-stone-300">
                      {formatQty(item.qty)}
                    </td>
                    <td className="px-4 py-1.5 text-right font-mono text-xs tabular-nums text-stone-700 dark:text-stone-300">
                      {lineTotal ? formatKronor(lineTotal) : ""}
                    </td>
                    {showNote && (
                      <td className="px-4 py-1.5 text-xs text-stone-600 dark:text-stone-400">
                        {item.reason_md}
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function formatKronor(v: number): string {
  // Swedish number style: "1 299,50 kr"
  return `${v
    .toFixed(2)
    .replace(".", ",")
    .replace(/\B(?=(\d{3})+(?!\d))/g, " ")} kr`;
}

function formatQty(q: number): string {
  if (Number.isInteger(q)) return String(q);
  return q.toFixed(2).replace(/\.?0+$/, "");
}

function today(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function WeekHeader({
  week,
  onAction,
  onPatch,
}: {
  week: WeekDetail;
  onAction: (a: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
}) {
  return (
    <header className="border-b border-stone-200 pb-4 dark:border-stone-800">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h1 className="font-serif text-3xl tracking-tight text-stone-900 dark:text-stone-100">
            <EditableText
              value={week.iso_week}
              label="iso week"
              onCommit={(v) => onPatch({ iso_week: v })}
              className="-mx-1"
            />
          </h1>
          <div className="mt-1 flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-stone-600 dark:text-stone-400">
            <EditableDate
              value={week.start_date}
              label="start date"
              onCommit={(v) => (v ? onPatch({ start_date: v }) : Promise.resolve())}
            />
            <span>→</span>
            <EditableDate
              value={week.end_date}
              label="end date"
              onCommit={(v) => (v ? onPatch({ end_date: v }) : Promise.resolve())}
            />
            <span className="ml-2 rounded-full bg-stone-100 px-2 py-0.5 text-xs font-medium uppercase tracking-wide text-stone-600 dark:bg-stone-800 dark:text-stone-300">
              {t(`status.${week.status}`)}
            </span>
          </div>
          <div className="mt-2 text-sm italic text-stone-600 dark:text-stone-400">
            <EditableText
              value={week.notes_md}
              label={t("week.notes_label")}
              placeholder={t("week.notes_placeholder")}
              multiline
              onCommit={(v) => onPatch({ notes_md: v })}
              className="-mx-1"
            />
          </div>
        </div>
        <div className="flex shrink-0 gap-2">
          <a
            href={`/weeks/${encodeURIComponent(week.iso_week)}/print`}
            target="_blank"
            rel="noreferrer"
            className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          >
            {t("week.print")}
          </a>
          <button
            type="button"
            onClick={() =>
              onAction(
                t("week.add_dinner_prompt", {
                  start: week.start_date,
                  end: week.end_date,
                }),
              )
            }
            className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          >
            {t("week.add_dinner")}
          </button>
          <button
            type="button"
            onClick={() =>
              onAction(
                t("week.regenerate_prompt", {
                  start: week.start_date,
                  end: week.end_date,
                }),
              )
            }
            className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          >
            {t("week.regenerate")}
          </button>
        </div>
      </div>
    </header>
  );
}

function Exceptions({ items }: { items: Exception[] }) {
  return (
    <aside className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
      <div className="font-medium">{t("week.this_week")}</div>
      <ul className="mt-1 list-disc pl-5">
        {items.map((e) => (
          <li key={e.id}>
            <span className="font-mono text-xs uppercase tracking-wide text-amber-700 dark:text-amber-400">
              {e.kind}
            </span>{" "}
            {e.description}
          </li>
        ))}
      </ul>
    </aside>
  );
}

function WeekRetrospective({ week }: { week: WeekDetail }) {
  const initial = week.retrospectives[0]?.notes_md ?? "";
  const [value, setValue] = useState(initial);
  const [status, setStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const timerRef = useRef<number | null>(null);

  // Reset local state when switching to a different week.
  useEffect(() => {
    setValue(initial);
    setStatus("idle");
  }, [initial]);

  const save = (next: string) => {
    setStatus("saving");
    setWeekRetrospective(week.id, next)
      .then(() => setStatus("saved"))
      .catch(() => setStatus("error"));
  };

  const onChange = (next: string) => {
    setValue(next);
    if (timerRef.current) window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(() => save(next), 700);
  };

  return (
    <section className="mt-2 rounded-md border border-stone-200 bg-stone-50 px-4 py-3 dark:border-stone-800 dark:bg-stone-900/50">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
          {t("week.retrospective")}
        </h2>
        <span className="text-xs text-stone-400 dark:text-stone-500">
          {status === "saving"
            ? t("retro.saving")
            : status === "saved"
              ? t("retro.saved")
              : status === "error"
                ? t("retro.error")
                : ""}
        </span>
      </div>
      <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">{t("retro.hint")}</p>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        placeholder={t("retro.placeholder")}
        className="mt-2 w-full resize-y rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
      />
    </section>
  );
}

function RatingControl({ dinner, onChanged }: { dinner: Dinner; onChanged: () => void }) {
  const [rating, setRating] = useState<DinnerRating | null>(dinner.rating);
  const [notes, setNotes] = useState(dinner.rating_notes);
  const [status, setStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const timerRef = useRef<number | null>(null);

  // If the prop changes (e.g. week refetch after agent tool) pick it up.
  useEffect(() => {
    setRating(dinner.rating);
    setNotes(dinner.rating_notes);
  }, [dinner.rating, dinner.rating_notes]);

  const persist = (nextRating: DinnerRating, nextNotes: string) => {
    setStatus("saving");
    setDinnerRating(dinner.id, nextRating, nextNotes)
      .then(() => {
        setStatus("saved");
        onChanged();
      })
      .catch(() => setStatus("error"));
  };

  const pickRating = (next: DinnerRating) => {
    setRating(next);
    if (timerRef.current) window.clearTimeout(timerRef.current);
    persist(next, notes);
  };

  const changeNotes = (next: string) => {
    setNotes(next);
    if (!rating) return; // notes alone aren't meaningful without a rating
    if (timerRef.current) window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(() => persist(rating, next), 700);
  };

  const clear = () => {
    setRating(null);
    setNotes("");
    setStatus("saving");
    clearDinnerRating(dinner.id)
      .then(() => {
        setStatus("idle");
        onChanged();
      })
      .catch(() => setStatus("error"));
  };

  return (
    <div className="mt-3 border-t border-stone-100 pt-3 dark:border-stone-800">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-xs text-stone-500 dark:text-stone-400">
          {rating ? t("rating.your_verdict") : t("rating.how_was_it")}
        </span>
        {(["loved", "liked", "meh", "disliked"] as DinnerRating[]).map((r) => {
          const selected = rating === r;
          return (
            <button
              key={r}
              type="button"
              onClick={() => pickRating(r)}
              className={`rounded-full border px-2.5 py-0.5 text-xs transition-colors ${
                selected
                  ? ratingSelectedClass(r)
                  : "border-stone-300 bg-white text-stone-600 hover:bg-stone-100 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-300 dark:hover:bg-stone-700"
              }`}
            >
              {t(`rating.${r}`)}
            </button>
          );
        })}
        {rating && (
          <>
            <span className="ml-auto text-xs text-stone-400 dark:text-stone-500">
              {status === "saving"
                ? t("retro.saving")
                : status === "saved"
                  ? t("retro.saved")
                  : status === "error"
                    ? t("retro.error")
                    : ""}
            </span>
            <button
              type="button"
              onClick={clear}
              className="text-xs text-stone-500 hover:text-stone-800 dark:text-stone-400 dark:hover:text-stone-200"
            >
              {t("rating.clear")}
            </button>
          </>
        )}
      </div>
      {rating && (
        <textarea
          value={notes}
          onChange={(e) => changeNotes(e.target.value)}
          rows={2}
          placeholder={t("rating.notes_placeholder")}
          className="mt-2 w-full resize-none rounded-md border border-stone-200 bg-white px-3 py-2 text-sm text-stone-800 shadow-sm outline-none focus:border-stone-500 dark:border-stone-800 dark:bg-stone-900 dark:text-stone-100 dark:placeholder:text-stone-500"
        />
      )}
    </div>
  );
}

function ratingSelectedClass(r: DinnerRating): string {
  switch (r) {
    case "loved":
      return "border-rose-300 bg-rose-100 text-rose-800 dark:border-rose-800 dark:bg-rose-950/60 dark:text-rose-200";
    case "liked":
      return "border-emerald-300 bg-emerald-100 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-200";
    case "meh":
      return "border-stone-400 bg-stone-200 text-stone-800 dark:border-stone-600 dark:bg-stone-700 dark:text-stone-100";
    case "disliked":
      return "border-amber-300 bg-amber-100 text-amber-900 dark:border-amber-800 dark:bg-amber-950/60 dark:text-amber-200";
  }
}

function DinnerCard({
  dinner,
  dimmed,
  active,
  rateable,
  onAction,
  onRatingChanged,
}: {
  dinner: Dinner;
  dimmed: boolean;
  active: boolean;
  rateable: boolean;
  onAction: (a: string) => void;
  onRatingChanged: () => void;
}) {
  const [adjustOpen, setAdjustOpen] = useState(false);
  const [adjustDraft, setAdjustDraft] = useState("");
  const sourcing = Object.entries(dinner.sourcing || {});

  const submitAdjust = () => {
    const request = adjustDraft.trim();
    if (!request) return;
    onAction(
      t("dinner.adjust_prompt", {
        date: dinner.day_date,
        id: dinner.id,
        name: dinner.dish_name,
        request,
      }),
    );
    setAdjustOpen(false);
    setAdjustDraft("");
  };
  return (
    <article
      className={`rounded-lg border bg-white transition-colors dark:bg-stone-900 ${
        active
          ? "border-stone-900 ring-2 ring-stone-900/10 dark:border-stone-100 dark:ring-stone-100/20"
          : dimmed
            ? "border-stone-200 opacity-60 dark:border-stone-800"
            : "border-stone-200 dark:border-stone-800"
      }`}
    >
      <header className="flex items-start justify-between gap-3 px-4 pt-4 pb-3">
        <div>
          <div className="text-xs font-mono uppercase tracking-wide text-stone-500 tabular-nums dark:text-stone-400">
            {dinner.day_date} · {formatWeekday(dinner.day_date)}
          </div>
          <h3 className="mt-0.5 font-serif text-xl text-stone-900 dark:text-stone-100">
            {dinner.dish_name || t("dinner.untitled")}
          </h3>
          <div className="mt-1 flex flex-wrap gap-1.5 text-xs">
            {dinner.cuisine && (
              <span className="rounded-full bg-stone-100 px-2 py-0.5 text-stone-600 dark:bg-stone-800 dark:text-stone-300">
                {dinner.cuisine}
              </span>
            )}
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-stone-600 dark:bg-stone-800 dark:text-stone-300">
              {dinner.servings} {t("dinner.servings")}
            </span>
            {sourcing.map(([src, desc]) => (
              <span
                key={src}
                className={`rounded-full px-2 py-0.5 ${
                  src === "fishmonger"
                    ? "bg-sky-100 text-sky-800 dark:bg-sky-950/60 dark:text-sky-300"
                    : src === "butcher"
                      ? "bg-rose-100 text-rose-800 dark:bg-rose-950/60 dark:text-rose-300"
                      : src === "bakery"
                        ? "bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300"
                        : "bg-stone-100 text-stone-600 dark:bg-stone-800 dark:text-stone-300"
                }`}
                title={desc}
              >
                {src}
              </span>
            ))}
          </div>
          {dinner.notes && (
            <p className="mt-2 text-sm italic text-stone-600 dark:text-stone-400">{dinner.notes}</p>
          )}
        </div>
        <div className="flex shrink-0 items-start gap-2">
          {dinner.rating && (
            <span
              className={`rounded-full border px-2 py-0.5 text-xs ${ratingSelectedClass(dinner.rating)}`}
              title={dinner.rating_notes || undefined}
            >
              {t(`rating.${dinner.rating}`)}
            </span>
          )}
          <button
            type="button"
            onClick={() => setAdjustOpen((o) => !o)}
            className={`shrink-0 rounded-md border px-2.5 py-1 text-xs ${
              adjustOpen
                ? "border-stone-900 bg-stone-900 text-stone-50 dark:border-stone-100 dark:bg-stone-100 dark:text-stone-900"
                : "border-stone-300 bg-white text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
            }`}
          >
            {t("dinner.adjust")}
          </button>
        </div>
      </header>
      {adjustOpen && (
        <div className="border-t border-stone-100 bg-stone-50/60 px-4 py-3 dark:border-stone-800 dark:bg-stone-950/60">
          <textarea
            value={adjustDraft}
            onChange={(e) => setAdjustDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                submitAdjust();
              } else if (e.key === "Escape") {
                setAdjustOpen(false);
                setAdjustDraft("");
              }
            }}
            rows={2}
            placeholder={t("dinner.adjust_placeholder")}
            className="w-full resize-none rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
            // biome-ignore lint/a11y/noAutofocus: focus on open is expected UX
            autoFocus
          />
          <div className="mt-2 flex items-center justify-between gap-2">
            <span className="text-[11px] text-stone-500 dark:text-stone-400">
              {t("dinner.adjust_hint")}
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => {
                  setAdjustOpen(false);
                  setAdjustDraft("");
                }}
                className="rounded-md border border-stone-300 bg-white px-2.5 py-1 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
              >
                {t("dinner.adjust_cancel")}
              </button>
              <button
                type="button"
                onClick={submitAdjust}
                disabled={!adjustDraft.trim()}
                className="rounded-md bg-stone-900 px-3 py-1 text-xs font-medium text-stone-50 hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
              >
                {t("dinner.adjust_send")}
              </button>
            </div>
          </div>
        </div>
      )}
      {dinner.recipe_md && (
        <details className="group border-t border-stone-100 px-4 py-2 text-sm dark:border-stone-800">
          <summary className="cursor-pointer select-none text-stone-600 group-open:text-stone-900 dark:text-stone-400 dark:group-open:text-stone-100">
            {t("dinner.recipe")}
          </summary>
          <Markdown
            source={dinner.recipe_md}
            variant="compact"
            className="mt-2"
            stripLeadingHeading={dinner.dish_name}
            headingShift={2}
          />
        </details>
      )}
      {rateable && (
        <div className="px-4 pb-3">
          <RatingControl dinner={dinner} onChanged={onRatingChanged} />
        </div>
      )}
    </article>
  );
}

function groupByDay(dinners: Dinner[]): Dinner[] {
  return [...dinners].sort((a, b) => a.day_date.localeCompare(b.day_date));
}
