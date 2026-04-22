import { useCallback, useEffect, useRef, useState } from "react";
import { applyAgentEvent, ChatDrawer, type ChatEntry } from "./components/ChatDrawer";
import { PlanNewForm } from "./components/PlanNewForm";
import { PreferencesModal } from "./components/PreferencesModal";
import { PrintableWeek } from "./components/PrintableWeek";
import { SettingsModal } from "./components/SettingsModal";
import { WeeksSidebar } from "./components/WeeksSidebar";
import { WeekView } from "./components/WeekView";
import { setLang, t, useLang } from "./i18n";
import {
  type AgentEvent,
  deleteWeekConversations,
  getCurrentWeek,
  getSettings,
  getWeek,
  getWeekConversation,
  MUTATING_TOOLS,
  patchWeek,
  streamChat,
  type WeekDetail,
  type WeekPatch,
} from "./lib/api";
import { setTheme, useTheme } from "./lib/theme";

type Status = "loading" | "empty" | "ready" | "error";

export function App() {
  // Print mode uses a completely separate tree with no sidebar/chat/topbar.
  const printISO =
    typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("print") : null;
  if (printISO) return <PrintableWeek iso={printISO} />;
  return <Main />;
}

function Main() {
  const [status, setStatus] = useState<Status>("loading");
  const [week, setWeek] = useState<WeekDetail | null>(null);
  const [errorText, setErrorText] = useState<string | null>(null);

  const [chatOpen, setChatOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [preferencesOpen, setPreferencesOpen] = useState(false);
  const [chatEntries, setChatEntries] = useState<ChatEntry[]>([]);
  const [busy, setBusy] = useState(false);
  const [conversationID, setConversationID] = useState<number | null>(null);
  const [activeDay, setActiveDay] = useState<string | null>(null);
  const [selectedISO, setSelectedISO] = useState<string | null>(null); // null = current/latest
  const [sidebarRefresh, setSidebarRefresh] = useState(0);
  const abortRef = useRef<AbortController | null>(null);

  useLang(); // subscribe root to language changes

  // Load language once on mount; all other settings reads are on-demand.
  useEffect(() => {
    getSettings()
      .then((s) => setLang(s.language))
      .catch(() => {
        /* keep default */
      });
  }, []);

  const refreshWeek = useCallback(async () => {
    try {
      const fetched = selectedISO ? await getWeek(selectedISO) : await getCurrentWeek();
      if (fetched) {
        setWeek(fetched);
        setStatus("ready");
        if (selectedISO === null) setSelectedISO(fetched.iso_week);
      } else {
        setStatus("empty");
      }
      setSidebarRefresh((k) => k + 1);
    } catch (err) {
      setErrorText(err instanceof Error ? err.message : String(err));
      setStatus("error");
    }
  }, [selectedISO]);

  // When the active week changes, rehydrate the chat drawer from its
  // stored conversation so you can resume where you left off.
  useEffect(() => {
    if (!week) {
      setChatEntries([]);
      setConversationID(null);
      return;
    }
    let cancelled = false;
    getWeekConversation(week.id)
      .then((res) => {
        if (cancelled) return;
        if (!res) {
          setChatEntries([]);
          setConversationID(null);
          return;
        }
        setConversationID(res.conversation.id);
        setChatEntries(
          res.messages.map((m) =>
            m.role === "user"
              ? { kind: "user", text: m.text }
              : { kind: "assistant", text: m.text },
          ),
        );
      })
      .catch(() => {
        if (!cancelled) {
          setChatEntries([]);
          setConversationID(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [week?.id]);

  useEffect(() => {
    void refreshWeek();
  }, [refreshWeek]);

  const assistantIndex = useRef({ current: -1 });
  const toolsByID = useRef(new Map<string, number>());

  const handleAgentEvent = useCallback(
    (ev: AgentEvent) => {
      setChatEntries((prev) => {
        const next = prev.slice();
        applyAgentEvent(next, ev, {
          assistantIndex: assistantIndex.current,
          toolsByID: toolsByID.current,
        });
        return next;
      });

      if (ev.type === "tool_call" && ev.tool === "add_dinner" && ev.input) {
        // Best-effort: pull day_date out of the input so we can highlight the
        // card that's about to appear.
        try {
          const parsed = JSON.parse(ev.input) as { day_date?: string };
          if (parsed.day_date) setActiveDay(parsed.day_date);
        } catch {
          // ignore
        }
      }

      if (ev.type === "tool_result" && ev.tool && MUTATING_TOOLS.has(ev.tool)) {
        // Refetch to pick up the new rows — this is what makes the menu feel live.
        void refreshWeek();
      }
    },
    [refreshWeek],
  );

  const send = useCallback(
    (message: string) => {
      if (busy) return;
      const controller = new AbortController();
      abortRef.current = controller;
      setBusy(true);
      setChatOpen(true);
      assistantIndex.current = { current: -1 };
      toolsByID.current = new Map();
      setChatEntries((prev) => [...prev, { kind: "user", text: message }]);

      (async () => {
        try {
          await streamChat(
            {
              conversation_id: conversationID ?? undefined,
              week_id: week?.id,
              message,
            },
            {
              onMeta: ({ conversation_id }) => {
                if (conversationID == null) setConversationID(conversation_id);
              },
              onEvent: handleAgentEvent,
            },
            controller.signal,
          );
        } catch (err) {
          if (controller.signal.aborted) {
            setChatEntries((prev) => [...prev, { kind: "cancelled" }]);
          } else {
            setChatEntries((prev) => [
              ...prev,
              { kind: "error", text: err instanceof Error ? err.message : String(err) },
            ]);
          }
        } finally {
          abortRef.current = null;
          setBusy(false);
          setActiveDay(null);
          void refreshWeek();
        }
      })();
    },
    [busy, conversationID, handleAgentEvent, refreshWeek],
  );

  const cancel = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const clearChat = useCallback(async () => {
    if (!week) return;
    abortRef.current?.abort();
    try {
      await deleteWeekConversations(week.id);
    } catch (err) {
      console.error("clear chat", err);
    }
    setChatEntries([]);
    setConversationID(null);
  }, [week]);

  const patchCurrentWeek = useCallback(
    async (patch: WeekPatch) => {
      if (!week) return;
      const updated = await patchWeek(week.id, patch);
      setWeek(updated);
    },
    [week],
  );

  return (
    <div className="flex h-full w-full flex-col">
      <TopBar
        chatOpen={chatOpen}
        onToggleChat={() => setChatOpen((o) => !o)}
        onOpenSettings={() => setSettingsOpen(true)}
        onOpenPreferences={() => setPreferencesOpen(true)}
        onRefresh={() => void refreshWeek()}
        busy={busy}
        onCancel={cancel}
      />
      <div className="flex flex-1 overflow-hidden">
        <WeeksSidebar
          selectedISO={selectedISO}
          onSelect={setSelectedISO}
          refreshKey={sidebarRefresh}
        />
        <main className="flex-1 overflow-y-auto bg-stone-50 dark:bg-stone-950">
          {status === "loading" && <LoadingState />}
          {status === "error" && (
            <div className="mx-auto mt-12 max-w-xl rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
              {errorText}
            </div>
          )}
          {status === "empty" && <PlanNewForm onSubmit={send} busy={busy} />}
          {status === "ready" && week && (
            <WeekView
              week={week}
              activeDayDate={activeDay}
              onAction={send}
              onPatch={patchCurrentWeek}
            />
          )}
        </main>
        <ChatDrawer
          open={chatOpen}
          onOpenChange={setChatOpen}
          busy={busy}
          entries={chatEntries}
          onSend={send}
          onClear={week ? clearChat : undefined}
        />
      </div>
      <SettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <PreferencesModal open={preferencesOpen} onClose={() => setPreferencesOpen(false)} />
    </div>
  );
}

function TopBar({
  chatOpen,
  onToggleChat,
  onOpenSettings,
  onOpenPreferences,
  onRefresh,
  busy,
  onCancel,
}: {
  chatOpen: boolean;
  onToggleChat: () => void;
  onOpenSettings: () => void;
  onOpenPreferences: () => void;
  onRefresh: () => void;
  busy: boolean;
  onCancel: () => void;
}) {
  return (
    <header className="flex items-center justify-between border-b border-stone-200 bg-white px-5 py-3 dark:border-stone-800 dark:bg-stone-900">
      <div className="flex items-center gap-3">
        <span className="font-serif text-lg font-semibold tracking-tight text-stone-900 dark:text-stone-100">
          Veckomenyn<span className="text-orange-600 dark:text-orange-500">.</span>
        </span>
        {busy && (
          <button
            type="button"
            onClick={onCancel}
            className="flex items-center gap-1.5 rounded-md border border-stone-300 bg-white px-2 py-1 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
            title={t("topbar.stop")}
          >
            <span
              className="inline-block h-2 w-2 animate-pulse rounded-full bg-emerald-500"
              aria-hidden
            />
            {t("topbar.working")}
            <span className="ml-1 text-stone-400">·</span>
            <span className="font-medium text-stone-700 dark:text-stone-200">
              {t("topbar.stop")}
            </span>
          </button>
        )}
      </div>
      <div className="flex items-center gap-2">
        <ThemeToggleButton />
        <button
          type="button"
          onClick={onRefresh}
          className="rounded-md border border-stone-300 bg-white px-2.5 py-1 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
          title={t("topbar.refresh")}
        >
          {t("topbar.refresh")}
        </button>
        <button
          type="button"
          onClick={onOpenPreferences}
          className="rounded-md border border-stone-300 bg-white px-2.5 py-1 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
        >
          {t("topbar.preferences")}
        </button>
        <button
          type="button"
          onClick={onOpenSettings}
          className="rounded-md border border-stone-300 bg-white px-2.5 py-1 text-xs text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
        >
          {t("topbar.settings")}
        </button>
        <button
          type="button"
          onClick={onToggleChat}
          className={`rounded-md px-3 py-1.5 text-xs font-medium ${
            chatOpen
              ? "border border-stone-300 bg-white text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
              : "bg-stone-900 text-stone-50 hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
          }`}
        >
          {chatOpen ? t("topbar.close_chat") : t("topbar.open_chat")}
        </button>
      </div>
    </header>
  );
}

function ThemeToggleButton() {
  // One-click cycle: system → light → dark → system.
  const { theme, resolved } = useTheme();
  const next = theme === "system" ? "light" : theme === "light" ? "dark" : "system";
  const label =
    theme === "system"
      ? t("settings.theme_system")
      : theme === "light"
        ? t("settings.theme_light")
        : t("settings.theme_dark");
  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      title={`${t("topbar.theme")}: ${label}`}
      aria-label={`${t("topbar.theme")}: ${label}`}
      className="flex h-7 w-7 items-center justify-center rounded-md border border-stone-300 bg-white text-stone-700 hover:bg-stone-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
    >
      {resolved === "dark" ? (
        // Moon
        <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path d="M17.293 13.293a8 8 0 0 1-10.586-10.586 8 8 0 1 0 10.586 10.586Z" />
        </svg>
      ) : (
        // Sun
        <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path d="M10 3a1 1 0 0 1 1 1v1a1 1 0 1 1-2 0V4a1 1 0 0 1 1-1Zm0 12a1 1 0 0 1 1 1v1a1 1 0 1 1-2 0v-1a1 1 0 0 1 1-1Zm7-5a1 1 0 0 1-1 1h-1a1 1 0 1 1 0-2h1a1 1 0 0 1 1 1ZM4 10a1 1 0 0 1-1 1H2a1 1 0 1 1 0-2h1a1 1 0 0 1 1 1Zm11.07-5.07a1 1 0 0 1 0 1.414l-.707.707a1 1 0 1 1-1.414-1.414l.707-.707a1 1 0 0 1 1.414 0ZM6.05 13.95a1 1 0 0 1 0 1.414l-.707.707A1 1 0 1 1 3.93 14.66l.707-.708a1 1 0 0 1 1.414 0Zm9.02 1.414a1 1 0 0 1-1.414 0l-.707-.707a1 1 0 1 1 1.414-1.414l.707.707a1 1 0 0 1 0 1.414ZM6.05 6.05a1 1 0 0 1-1.414 0l-.707-.707A1 1 0 0 1 5.343 3.93l.708.707a1 1 0 0 1 0 1.414ZM10 6a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z" />
        </svg>
      )}
    </button>
  );
}

function LoadingState() {
  return (
    <div className="mx-auto max-w-md px-6 py-16 text-center text-sm text-stone-500 dark:text-stone-400">
      Loading…
    </div>
  );
}
