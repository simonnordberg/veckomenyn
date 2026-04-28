import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { formatPeriod, formatWeekday, t, useLang } from "../i18n";
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
  type WeekSummary,
} from "../lib/api";
import { toast } from "../lib/toast";
import { EditableDate, EditableText } from "./Editable";
import { Markdown } from "./Markdown";

type Props = {
  week: WeekDetail;
  activeDayDate: string | null; // day the agent is currently writing a dinner for, for subtle highlight
  onAction: (action: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
  onRefetch: () => void;
  onDuplicate: (source: WeekSummary) => void;
};

export function WeekView({
  week,
  activeDayDate,
  onAction,
  onPatch,
  onRefetch,
  onDuplicate,
}: Props) {
  useLang();
  const dinners = useMemo(() => groupByDay(week.dinners), [week.dinners]);
  // Rating & retrospective live on weeks you've actually cooked through.
  // Hiding them while planning keeps the card tidy and reflects the lifecycle.
  const rateable = week.status === "ordered";
  // Menu edits are only allowed in draft. Past planning the user goes back
  // to draft via StatusMenu, which prompts before reopening edits.
  const locked = week.status !== "draft";

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-6 sm:px-6 sm:py-8">
      <WeekHeader week={week} locked={locked} onAction={onAction} onPatch={onPatch} />
      {week.exceptions.length > 0 && <Exceptions items={week.exceptions} />}
      <section className="flex flex-col gap-3">
        {dinners.length === 0 ? (
          <div className="rounded-lg border border-dashed border-stone-300 bg-stone-50 px-6 py-10 text-center text-stone-500 dark:border-stone-700 dark:bg-stone-900/50 dark:text-stone-400">
            <p>{t("week.no_dinners")}</p>
            {!locked && (
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
            )}
          </div>
        ) : (
          dinners.map((d) => (
            <DinnerCard
              key={d.id}
              dinner={d}
              dimmed={activeDayDate !== null && activeDayDate !== d.day_date}
              active={activeDayDate === d.day_date}
              rateable={rateable}
              locked={locked}
              onAction={onAction}
              onRatingChanged={onRefetch}
            />
          ))
        )}
      </section>
      {week.cart_items.length > 0 && <CartSection items={week.cart_items} />}
      <Lifecycle week={week} onAction={onAction} onPatch={onPatch} onDuplicate={onDuplicate} />
      {rateable && <WeekRetrospective week={week} />}
    </div>
  );
}

function Lifecycle({
  week,
  onAction,
  onPatch,
  onDuplicate,
}: {
  week: WeekDetail;
  onAction: (a: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
  onDuplicate: (source: WeekSummary) => void;
}) {
  const hasRetro = week.retrospectives.length > 0;

  // Next recommended step based on status. For an ordered + retro'd week, the
  // natural next step is starting next week from this one (clone).
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
        return {
          label: t("week.clone_next"),
          run: () => onDuplicate(week),
        };
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

const STATUS_ORDER: WeekDetail["status"][] = ["draft", "cart_built", "ordered"];

function isBackwardTransition(from: WeekDetail["status"], to: WeekDetail["status"]): boolean {
  return STATUS_ORDER.indexOf(to) < STATUS_ORDER.indexOf(from);
}

function StatusMenu({
  current,
  onPick,
}: {
  current: WeekDetail["status"];
  onPick: (s: WeekDetail["status"]) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
      >
        {t("lifecycle.set_status")} ▾
      </button>
      {open && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setOpen(false)}
            onKeyDown={(e) => {
              if (e.key === "Escape") setOpen(false);
            }}
            role="button"
            tabIndex={-1}
            aria-label={t("toast.dismiss")}
          />
          <div
            role="menu"
            className="absolute right-0 top-full z-20 mt-1 min-w-40 rounded-md border border-stone-200 bg-white p-1 shadow-lg dark:border-stone-700 dark:bg-stone-800"
          >
            {STATUS_ORDER.map((s) => (
              <button
                key={s}
                type="button"
                role="menuitem"
                onClick={() => {
                  setOpen(false);
                  if (s === current) return;
                  // Going back unlocks edits; ask first so a stray click on
                  // an ordered week can't silently reopen the menu.
                  if (
                    isBackwardTransition(current, s) &&
                    !window.confirm(t("week.unlock_confirm", { target: t(`status.${s}`) }))
                  ) {
                    return;
                  }
                  onPick(s);
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
        <div className="overflow-x-auto border-t border-stone-100 dark:border-stone-800">
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
  locked,
  onAction,
  onPatch,
}: {
  week: WeekDetail;
  locked: boolean;
  onAction: (a: string) => void;
  onPatch: (patch: WeekPatch) => Promise<void>;
}) {
  return (
    <header className="border-b border-stone-200 pb-4 dark:border-stone-800">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
        <div className="min-w-0 flex-1">
          <h1 className="font-serif text-2xl tracking-tight text-stone-900 sm:text-3xl dark:text-stone-100">
            {formatPeriod(week.start_date, week.end_date)}
          </h1>
          <div className="mt-1 flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-stone-600 dark:text-stone-400">
            {locked ? (
              <span className="px-1 py-0.5 font-mono tabular-nums text-stone-700 dark:text-stone-300">
                {week.start_date}
              </span>
            ) : (
              <EditableDate
                value={week.start_date}
                label="start date"
                onCommit={(v) => (v ? onPatch({ start_date: v }) : Promise.resolve())}
              />
            )}
            <span>→</span>
            {locked ? (
              <span className="px-1 py-0.5 font-mono tabular-nums text-stone-700 dark:text-stone-300">
                {week.end_date}
              </span>
            ) : (
              <EditableDate
                value={week.end_date}
                label="end date"
                onCommit={(v) => {
                  if (!v) return Promise.resolve();
                  // Shrinking past existing dinners drops them on the server,
                  // so ask first with the count so the user can back out.
                  if (v < week.end_date) {
                    const lost = week.dinners.filter((d) => d.day_date > v).length;
                    if (lost > 0 && !window.confirm(t("week.truncate_confirm", { count: lost }))) {
                      return Promise.resolve();
                    }
                  }
                  return onPatch({ end_date: v });
                }}
              />
            )}
            <span className="ml-2 rounded-full bg-stone-100 px-2 py-0.5 text-xs font-medium uppercase tracking-wide text-stone-600 dark:bg-stone-800 dark:text-stone-300">
              {t(`status.${week.status}`)}
            </span>
          </div>
          {(week.notes_md || !locked) && (
            <div className="mt-2 text-sm italic text-stone-600 dark:text-stone-400">
              {locked ? (
                <span className="px-1">{week.notes_md}</span>
              ) : (
                <EditableText
                  value={week.notes_md}
                  label={t("week.notes_label")}
                  placeholder={t("week.notes_placeholder")}
                  multiline
                  onCommit={(v) => onPatch({ notes_md: v })}
                  className="-mx-1"
                />
              )}
            </div>
          )}
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <a
            href={`/weeks/${week.id}/print`}
            target="_blank"
            rel="noreferrer"
            className="rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          >
            {t("week.print")}
          </a>
          {!locked && (
            <>
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
            </>
          )}
        </div>
      </div>
      {locked && (
        <p className="mt-2 text-xs text-stone-500 dark:text-stone-400">
          {t("week.locked_hint", { status: t(`status.${week.status}`) })}
        </p>
      )}
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

// Retry strategy: input debounces by 700ms; failed saves auto-retry up to
// MAX_ATTEMPTS with exponential backoff. The error toast updates in place
// (constant id per week) and offers a manual retry once auto-retry exhausts.
const RETRO_DEBOUNCE_MS = 700;
const RETRO_MAX_ATTEMPTS = 3;
const RETRO_BACKOFFS_MS = [1000, 3000, 9000];

function WeekRetrospective({ week }: { week: WeekDetail }) {
  const initial = week.retrospectives[0]?.notes_md ?? "";
  const [value, setValue] = useState(initial);
  const savedRef = useRef(initial);
  const dirtyRef = useRef(false);
  const debounceRef = useRef<number | null>(null);
  const retryRef = useRef<number | null>(null);
  const attemptsRef = useRef(0);
  const toastID = `retro-${week.id}`;

  // Reset local state when switching to a different week.
  useEffect(() => {
    setValue(initial);
    savedRef.current = initial;
    dirtyRef.current = false;
    attemptsRef.current = 0;
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
    if (retryRef.current) window.clearTimeout(retryRef.current);
    toast.dismiss(toastID);
  }, [initial, toastID]);

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      if (debounceRef.current) window.clearTimeout(debounceRef.current);
      if (retryRef.current) window.clearTimeout(retryRef.current);
    };
  }, []);

  // Warn before navigating away with unsaved changes. Registered once;
  // dirtyRef tracks whether the latest value matches the last successful save.
  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (!dirtyRef.current) return;
      e.preventDefault();
      e.returnValue = t("toast.unsaved_changes");
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, []);

  const save = useCallback(
    async (next: string): Promise<void> => {
      try {
        await setWeekRetrospective(week.id, next);
        savedRef.current = next;
        dirtyRef.current = false;
        attemptsRef.current = 0;
        toast.dismiss(toastID);
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        attemptsRef.current += 1;
        if (attemptsRef.current < RETRO_MAX_ATTEMPTS) {
          toast.error(`${t("toast.save_failed_retrying")} ${message}`, { id: toastID });
          const delay =
            RETRO_BACKOFFS_MS[Math.min(attemptsRef.current - 1, RETRO_BACKOFFS_MS.length - 1)];
          retryRef.current = window.setTimeout(() => void save(next), delay);
        } else {
          toast.error(`${t("toast.save_failed")}: ${message}`, {
            id: toastID,
            action: {
              label: t("toast.retry"),
              onClick: () => {
                attemptsRef.current = 0;
                void save(next);
              },
            },
          });
        }
      }
    },
    [week.id, toastID],
  );

  const onChange = (next: string) => {
    setValue(next);
    dirtyRef.current = next !== savedRef.current;
    if (debounceRef.current) window.clearTimeout(debounceRef.current);
    if (retryRef.current) window.clearTimeout(retryRef.current);
    // Drop the prior error toast: its manual-retry button captures the old
    // value and would overwrite a newer save if the user clicks it later.
    toast.dismiss(toastID);
    attemptsRef.current = 0;
    debounceRef.current = window.setTimeout(() => void save(next), RETRO_DEBOUNCE_MS);
  };

  return (
    <section className="mt-2 rounded-md border border-stone-200 bg-stone-50 px-4 py-3 dark:border-stone-800 dark:bg-stone-900/50">
      <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
        {t("week.retrospective")}
      </h2>
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
  const timerRef = useRef<number | null>(null);

  // If the prop changes (e.g. week refetch after agent tool) pick it up.
  useEffect(() => {
    setRating(dinner.rating);
    setNotes(dinner.rating_notes);
  }, [dinner.rating, dinner.rating_notes]);

  const persist = (nextRating: DinnerRating, nextNotes: string) => {
    setDinnerRating(dinner.id, nextRating, nextNotes)
      .then(() => {
        onChanged();
      })
      .catch((err: Error) => toast.error(err.message));
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
    clearDinnerRating(dinner.id)
      .then(() => {
        onChanged();
      })
      .catch((err: Error) => toast.error(err.message));
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
          <button
            type="button"
            onClick={clear}
            className="ml-auto text-xs text-stone-500 hover:text-stone-800 dark:text-stone-400 dark:hover:text-stone-200"
          >
            {t("rating.clear")}
          </button>
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
  locked,
  onAction,
  onRatingChanged,
}: {
  dinner: Dinner;
  dimmed: boolean;
  active: boolean;
  rateable: boolean;
  locked: boolean;
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
          {!locked && (
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
          )}
        </div>
      </header>
      {!locked && adjustOpen && (
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
