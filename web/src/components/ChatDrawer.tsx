import { useEffect, useRef, useState } from "react";
import { t, useLang } from "../i18n";
import { type AgentEvent, listProviders } from "../lib/api";
import { Markdown } from "./Markdown";

export type ChatEntry =
  | { kind: "user"; text: string }
  | { kind: "assistant"; text: string }
  | { kind: "tool"; name: string; input: string; result?: string; isError?: boolean }
  | { kind: "error"; text: string }
  | { kind: "cancelled" };

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  busy: boolean;
  entries: ChatEntry[];
  onSend: (message: string) => void;
  onClear?: () => void;
};

export function ChatDrawer({ open, onOpenChange, busy, entries, onSend, onClear }: Props) {
  useLang();
  const [input, setInput] = useState("");
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [entries]);

  useEffect(() => {
    if (open) textareaRef.current?.focus();
  }, [open]);

  const submit = () => {
    const message = input.trim();
    if (!message || busy) return;
    setInput("");
    onSend(message);
  };

  if (!open) return null;

  return (
    <aside
      className="fixed inset-0 z-30 flex h-full flex-col border-l border-stone-200 bg-white md:static md:z-auto md:w-96 md:shrink-0 dark:border-stone-800 dark:bg-stone-900"
      aria-label={t("chat.aria")}
    >
      <header className="flex items-center justify-between border-b border-stone-200 px-4 py-3 dark:border-stone-800">
        <div className="flex items-baseline gap-2">
          <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
            {t("chat.title")}
          </h2>
          <ActiveModelChip open={open} />
        </div>
        <div className="flex items-center gap-1">
          {onClear && entries.length > 0 && (
            <button
              type="button"
              onClick={() => {
                if (window.confirm(t("chat.clear_confirm"))) onClear();
              }}
              disabled={busy}
              className="rounded-md px-2 py-1 text-xs text-stone-600 hover:bg-stone-100 disabled:opacity-40 dark:text-stone-300 dark:hover:bg-stone-800"
              title={t("chat.clear")}
            >
              {t("chat.clear")}
            </button>
          )}
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            className="rounded-md p-1 text-stone-500 hover:bg-stone-100 dark:text-stone-400 dark:hover:bg-stone-800"
            aria-label={t("topbar.close_chat")}
          >
            <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
            </svg>
          </button>
        </div>
      </header>
      <div ref={scrollRef} className="flex-1 overflow-x-hidden overflow-y-auto px-4 py-4">
        {entries.length === 0 && !busy && (
          <p className="text-sm text-stone-500 dark:text-stone-400">{t("chat.empty")}</p>
        )}
        <div className="flex flex-col gap-3">
          {entries.map((e, i) => (
            <Entry key={i} entry={e} />
          ))}
          {busy && entries.length > 0 && entries[entries.length - 1].kind !== "assistant" && (
            <ThinkingIndicator />
          )}
        </div>
      </div>
      <div className="border-t border-stone-200 bg-stone-50/60 p-3 dark:border-stone-800 dark:bg-stone-950/60">
        <textarea
          ref={textareaRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          rows={2}
          placeholder={t("chat.placeholder")}
          className="w-full resize-none rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
          disabled={busy}
        />
        <div className="mt-2 flex justify-end">
          <button
            type="button"
            onClick={submit}
            disabled={busy || !input.trim()}
            className="rounded-md bg-stone-900 px-3 py-1.5 text-sm font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
          >
            {busy ? "…" : t("chat.send")}
          </button>
        </div>
      </div>
    </aside>
  );
}

function Entry({ entry }: { entry: ChatEntry }) {
  switch (entry.kind) {
    case "user":
      return (
        <div className="flex justify-end">
          <div className="max-w-[85%] whitespace-pre-wrap rounded-2xl rounded-br-sm bg-stone-900 px-3 py-2 text-sm text-stone-50 dark:bg-stone-100 dark:text-stone-900">
            {entry.text}
          </div>
        </div>
      );
    case "assistant":
      return <Markdown source={entry.text} variant="compact" />;
    case "tool":
      return <ToolEntry entry={entry} />;
    case "error":
      return (
        <div className="rounded-md border border-red-200 bg-red-50 px-2.5 py-1.5 text-xs text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {entry.text}
        </div>
      );
    case "cancelled":
      return (
        <div className="rounded-md border border-stone-200 bg-stone-50 px-2.5 py-1.5 text-xs italic text-stone-500 dark:border-stone-800 dark:bg-stone-900 dark:text-stone-400">
          {t("chat.cancelled")}
        </div>
      );
  }
}

function ToolEntry({
  entry,
}: {
  entry: { kind: "tool"; name: string; input: string; result?: string; isError?: boolean };
}) {
  const pending = entry.result === undefined;
  return (
    <details
      className={`group rounded-md border px-2.5 py-1.5 text-xs ${
        entry.isError
          ? "border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/50"
          : "border-stone-200 bg-stone-50 dark:border-stone-800 dark:bg-stone-900/50"
      }`}
    >
      <summary className="cursor-pointer list-none select-none font-mono text-stone-600 dark:text-stone-300">
        {pending ? "⠋" : entry.isError ? "✗" : "✓"} {entry.name}
      </summary>
      <div className="mt-2 space-y-1.5">
        <div>
          <div className="text-stone-500 dark:text-stone-400">input</div>
          <pre className="overflow-x-auto rounded bg-white p-1.5 font-mono text-[10px] text-stone-800 dark:bg-stone-800 dark:text-stone-200">
            {prettyJSON(entry.input)}
          </pre>
        </div>
        {entry.result !== undefined && (
          <div>
            <div className="text-stone-500 dark:text-stone-400">result</div>
            <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded bg-white p-1.5 font-mono text-[10px] text-stone-800 dark:bg-stone-800 dark:text-stone-200">
              {entry.result}
            </pre>
          </div>
        )}
      </div>
    </details>
  );
}

// ActiveModelChip shows the currently-configured Anthropic model in the
// chat header so users see what they're paying for during use, not just at
// config time. Refetches when the drawer opens so a Settings change is
// reflected without a page reload.
function ActiveModelChip({ open }: { open: boolean }) {
  const [model, setModel] = useState<string | null>(null);
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    listProviders()
      .then((env) => {
        if (cancelled) return;
        const anthropic = env.providers.find((p) => p.kind === "anthropic");
        setModel(anthropic?.config?.model ?? null);
      })
      .catch(() => {
        /* hide chip on error */
      });
    return () => {
      cancelled = true;
    };
  }, [open]);
  if (!model) return null;
  return (
    <span
      className="rounded bg-stone-100 px-1.5 py-0.5 text-[11px] tabular-nums text-stone-500 dark:bg-stone-800 dark:text-stone-400"
      title={model}
    >
      {shortModelName(model)}
    </span>
  );
}

// shortModelName renders a Claude model id as a human-friendly short name:
// "claude-sonnet-4-6" -> "Sonnet 4.6". Generic enough that future Claude
// model IDs work without a code change.
function shortModelName(id: string): string {
  const parts = id.replace(/^claude-/, "").split("-");
  if (parts.length === 0) return id;
  const name = parts[0].charAt(0).toUpperCase() + parts[0].slice(1);
  const version = parts.slice(1).join(".");
  return version ? `${name} ${version}` : name;
}

// ThinkingIndicator fills the silent gaps: after the user submits, and between
// a tool_result and the next text/tool event. The existing condition upstream
// hides it as soon as the model starts streaming text.
function ThinkingIndicator() {
  const phrases = [
    t("chat.deliberating.pondering"),
    t("chat.deliberating.simmering"),
    t("chat.deliberating.tasting"),
    t("chat.deliberating.chopping"),
    t("chat.deliberating.plating"),
    t("chat.deliberating.cookbook"),
    t("chat.deliberating.marinating"),
    t("chat.deliberating.whisking"),
    t("chat.deliberating.browning"),
    t("chat.deliberating.pantry"),
    t("chat.deliberating.kneading"),
    t("chat.deliberating.garnishing"),
    t("chat.deliberating.watched_pot"),
  ];
  // Start on a random phrase so two back-to-back waits don't always open
  // with the same word.
  const [idx, setIdx] = useState(() => Math.floor(Math.random() * phrases.length));
  useEffect(() => {
    const id = window.setInterval(() => {
      setIdx((i) => (i + 1) % phrases.length);
    }, 2400);
    return () => window.clearInterval(id);
  }, [phrases.length]);

  return (
    <div
      className="flex items-center gap-2 text-xs text-stone-500 dark:text-stone-400"
      aria-live="polite"
    >
      <span className="flex items-end gap-0.5" aria-hidden>
        <Dot delay={0} />
        <Dot delay={160} />
        <Dot delay={320} />
      </span>
      <span className="italic">{phrases[idx]}…</span>
    </div>
  );
}

function Dot({ delay }: { delay: number }) {
  return (
    <span
      className="inline-block h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 dark:bg-stone-500"
      style={{ animationDelay: `${delay}ms`, animationDuration: "1s" }}
    />
  );
}

function prettyJSON(s: string): string {
  try {
    return JSON.stringify(JSON.parse(s), null, 2);
  } catch {
    return s;
  }
}

export function applyAgentEvent(
  entries: ChatEntry[],
  ev: AgentEvent,
  state: { assistantIndex: { current: number }; toolsByID: Map<string, number> },
): void {
  switch (ev.type) {
    case "text": {
      if (!ev.text) return;
      const idx = state.assistantIndex.current;
      if (idx >= 0 && entries[idx]?.kind === "assistant") {
        entries[idx] = { kind: "assistant", text: entries[idx].text + ev.text };
      } else {
        entries.push({ kind: "assistant", text: ev.text });
        state.assistantIndex.current = entries.length - 1;
      }
      break;
    }
    case "tool_call_started":
      // We draw the card when tool_call (with full input) arrives; no-op here.
      break;
    case "tool_call": {
      state.assistantIndex.current = -1;
      entries.push({
        kind: "tool",
        name: ev.tool || "tool",
        input: ev.input || "{}",
      });
      if (ev.tool_id) state.toolsByID.set(ev.tool_id, entries.length - 1);
      break;
    }
    case "tool_result": {
      const idx = ev.tool_id ? state.toolsByID.get(ev.tool_id) : undefined;
      if (idx !== undefined && entries[idx]?.kind === "tool") {
        const prev = entries[idx];
        entries[idx] = {
          kind: "tool",
          name: prev.name,
          input: prev.input,
          result: ev.result,
          isError: ev.is_error,
        };
      }
      break;
    }
    case "error":
      entries.push({ kind: "error", text: ev.result || "unknown error" });
      break;
    case "cancelled":
      entries.push({ kind: "cancelled" });
      break;
    case "done":
      break;
  }
}
