import { type KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";
import { formatMonthYear, shortWeekdaysMondayFirst, t } from "../i18n";

// ---------------------------------------------------------------------------
// EditableDate: custom calendar popover
// ---------------------------------------------------------------------------

type DateProps = {
  value: string | null;
  label: string;
  onCommit: (next: string | null) => Promise<void> | void;
  nullable?: boolean;
  className?: string;
};

export function EditableDate({
  value,
  label,
  onCommit,
  nullable = false,
  className = "",
}: DateProps) {
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const popRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (popRef.current?.contains(target) || triggerRef.current?.contains(target)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent | globalThis.KeyboardEvent) => {
      if ((e as globalThis.KeyboardEvent).key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey as EventListener);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey as EventListener);
    };
  }, [open]);

  const select = async (iso: string | null) => {
    if (iso === value) {
      setOpen(false);
      return;
    }
    if (iso === null && !nullable) {
      setOpen(false);
      return;
    }
    setSaving(true);
    try {
      await onCommit(iso);
      setOpen(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <span className={`relative inline-block ${className}`}>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((o) => !o)}
        disabled={saving}
        aria-label={t("week.edit_label", { label })}
        aria-expanded={open}
        className="rounded px-1 py-0.5 text-left font-mono tabular-nums hover:bg-stone-100 dark:hover:bg-stone-800"
      >
        {value ? value : <span className="text-stone-400">{t("week.set_label", { label })}</span>}
      </button>
      {open && (
        <div
          ref={popRef}
          className="absolute left-0 top-full z-40 mt-1 rounded-md border border-stone-200 bg-white p-3 shadow-lg dark:border-stone-700 dark:bg-stone-800"
        >
          <CalendarPopup value={value} nullable={nullable} onSelect={select} />
        </div>
      )}
    </span>
  );
}

type CalendarProps = {
  value: string | null;
  nullable: boolean;
  onSelect: (iso: string | null) => void;
};

function CalendarPopup({ value, nullable, onSelect }: CalendarProps) {
  const initial = value ? parseISO(value) : new Date();
  const [viewMonth, setViewMonth] = useState<{ year: number; month: number }>({
    year: initial.getFullYear(),
    month: initial.getMonth(),
  });

  const cells = useMemo(() => buildMonth(viewMonth.year, viewMonth.month), [viewMonth]);
  const todayISO = formatISO(new Date());

  const shiftMonth = (delta: number) => {
    setViewMonth((v) => {
      const d = new Date(v.year, v.month + delta, 1);
      return { year: d.getFullYear(), month: d.getMonth() };
    });
  };

  const monthLabel = formatMonthYear(viewMonth.year, viewMonth.month);

  return (
    <div className="w-64 text-sm text-stone-800 dark:text-stone-100">
      <div className="mb-2 flex items-center justify-between">
        <button
          type="button"
          onClick={() => shiftMonth(-1)}
          className="rounded px-2 py-1 text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-700"
          aria-label={t("calendar.prev_month")}
        >
          ‹
        </button>
        <span className="font-medium">{monthLabel}</span>
        <button
          type="button"
          onClick={() => shiftMonth(1)}
          className="rounded px-2 py-1 text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-700"
          aria-label={t("calendar.next_month")}
        >
          ›
        </button>
      </div>
      <div className="grid grid-cols-7 gap-0.5 text-center">
        {shortWeekdaysMondayFirst().map((d, i) => (
          <div
            key={i}
            className="py-1 text-[10px] font-medium uppercase tracking-wide text-stone-400"
          >
            {d}
          </div>
        ))}
        {cells.map((cell) => {
          const isSelected = cell.iso === value;
          const isToday = cell.iso === todayISO;
          const inMonth = cell.month === viewMonth.month;
          return (
            <button
              key={cell.iso}
              type="button"
              onClick={() => onSelect(cell.iso)}
              className={`rounded py-1 font-mono text-xs tabular-nums ${
                isSelected
                  ? "bg-stone-900 text-stone-50 hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
                  : `hover:bg-stone-100 dark:hover:bg-stone-700 ${
                      isToday ? "ring-1 ring-inset ring-stone-400 dark:ring-stone-500" : ""
                    } ${
                      inMonth
                        ? "text-stone-800 dark:text-stone-100"
                        : "text-stone-300 dark:text-stone-600"
                    }`
              }`}
            >
              {cell.day}
            </button>
          );
        })}
      </div>
      <div className="mt-3 flex justify-between">
        <button
          type="button"
          onClick={() => onSelect(formatISO(new Date()))}
          className="rounded px-2 py-1 text-xs text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-700"
        >
          {t("calendar.today")}
        </button>
        {nullable && (
          <button
            type="button"
            onClick={() => onSelect(null)}
            className="rounded px-2 py-1 text-xs text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-700"
          >
            {t("calendar.clear")}
          </button>
        )}
      </div>
    </div>
  );
}

type Cell = { iso: string; day: number; month: number };

function buildMonth(year: number, month: number): Cell[] {
  const first = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0).getDate();
  const firstJs = first.getDay(); // 0=Sun
  const firstIso = firstJs === 0 ? 7 : firstJs; // 1=Mon
  const leading = firstIso - 1;

  const cells: Cell[] = [];
  for (let i = leading; i > 0; i--) {
    const d = new Date(year, month, 1 - i);
    cells.push({ iso: formatISO(d), day: d.getDate(), month: d.getMonth() });
  }
  for (let i = 1; i <= lastDay; i++) {
    cells.push({ iso: formatISO(new Date(year, month, i)), day: i, month });
  }
  let nextI = 1;
  while (cells.length < 42) {
    const d = new Date(year, month + 1, nextI);
    cells.push({ iso: formatISO(d), day: d.getDate(), month: d.getMonth() });
    nextI += 1;
  }
  return cells;
}

function parseISO(iso: string): Date {
  return new Date(`${iso}T00:00:00`);
}

function formatISO(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

// ---------------------------------------------------------------------------
// EditableText: inline text/textarea
// ---------------------------------------------------------------------------

type TextProps = {
  value: string;
  label: string;
  placeholder?: string;
  onCommit: (next: string) => Promise<void> | void;
  multiline?: boolean;
  className?: string;
};

export function EditableText({
  value,
  label,
  placeholder,
  onCommit,
  multiline = false,
  className = "",
}: TextProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);

  useEffect(() => {
    setDraft(value);
  }, [value]);

  const commit = async () => {
    if (draft === value) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await onCommit(draft);
      setEditing(false);
    } catch {
      // keep editing open
    } finally {
      setSaving(false);
    }
  };

  const handleKey = (e: KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    if (e.key === "Escape") {
      setDraft(value);
      setEditing(false);
    }
    if (e.key === "Enter" && !e.shiftKey && !multiline) {
      e.preventDefault();
      void commit();
    }
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && multiline) {
      e.preventDefault();
      void commit();
    }
  };

  if (!editing) {
    return (
      <button
        type="button"
        onClick={() => setEditing(true)}
        className={`rounded px-1 py-0.5 text-left hover:bg-stone-100 dark:hover:bg-stone-800 ${className}`}
        aria-label={t("week.edit_label", { label })}
      >
        {value || (
          <span className="text-stone-400">
            {placeholder || t("editable.add_label", { label })}
          </span>
        )}
      </button>
    );
  }

  if (multiline) {
    return (
      <textarea
        ref={inputRef as React.RefObject<HTMLTextAreaElement>}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => void commit()}
        onKeyDown={handleKey}
        disabled={saving}
        placeholder={placeholder}
        rows={3}
        className={`w-full resize-y rounded-sm border border-stone-400 bg-white px-1 py-0.5 text-sm shadow-sm dark:border-stone-600 dark:bg-stone-800 dark:text-stone-100 ${className}`}
      />
    );
  }

  return (
    <input
      ref={inputRef as React.RefObject<HTMLInputElement>}
      type="text"
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={() => void commit()}
      onKeyDown={handleKey}
      disabled={saving}
      placeholder={placeholder}
      className={`rounded-sm border border-stone-400 bg-white px-1 py-0.5 text-sm shadow-sm dark:border-stone-600 dark:bg-stone-800 dark:text-stone-100 ${className}`}
    />
  );
}
