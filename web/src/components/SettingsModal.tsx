import { useCallback, useEffect, useRef, useState } from "react";
import { setLang, t, useLang } from "../i18n";
import {
  getSettings,
  getVersion,
  type HouseholdSettings,
  patchSettings,
  type VersionInfo,
} from "../lib/api";
import { navigate } from "../lib/route";
import { setTheme, type Theme, useTheme } from "../lib/theme";
import { BackupsSection } from "./BackupsSection";
import { IntegrationsSection } from "./IntegrationsSection";
import { UpdatesSection } from "./UpdatesSection";

type Props = {
  open: boolean;
  onClose: () => void;
};

type SaveStatus = "idle" | "saving" | "saved" | "error";

const SAVE_DEBOUNCE_MS = 400;
const SAVED_FLASH_MS = 1500;

export function SettingsModal({ open, onClose }: Props) {
  useLang();
  const { theme } = useTheme();
  const [settings, setSettings] = useState<HouseholdSettings | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saveStatus, setSaveStatus] = useState<SaveStatus>("idle");

  // Accumulates field changes between debounced flushes so rapid edits to
  // different fields ship as one PATCH.
  const pendingPatch = useRef<Partial<HouseholdSettings>>({});
  const saveTimer = useRef<number | null>(null);

  const flush = useCallback(async () => {
    const toSend = pendingPatch.current;
    pendingPatch.current = {};
    if (saveTimer.current) {
      window.clearTimeout(saveTimer.current);
      saveTimer.current = null;
    }
    if (Object.keys(toSend).length === 0) return;
    setSaveStatus("saving");
    setError(null);
    try {
      const next = await patchSettings(toSend);
      setSettings(next);
      if (toSend.language) setLang(next.language);
      setSaveStatus("saved");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setSaveStatus("error");
    }
  }, []);

  const scheduleSave = useCallback(
    (patch: Partial<HouseholdSettings>) => {
      pendingPatch.current = { ...pendingPatch.current, ...patch };
      if (saveTimer.current) window.clearTimeout(saveTimer.current);
      saveTimer.current = window.setTimeout(() => {
        void flush();
      }, SAVE_DEBOUNCE_MS);
    },
    [flush],
  );

  // Load on open; flush pending patch on close so the user doesn't lose
  // edits made right before dismissing the modal.
  useEffect(() => {
    if (!open) {
      if (Object.keys(pendingPatch.current).length > 0) void flush();
      return;
    }
    setError(null);
    setSaveStatus("idle");
    getSettings()
      .then(setSettings)
      .catch((e: Error) => setError(e.message));
  }, [open, flush]);

  // "Saved" is a momentary confirmation; fade it back to idle shortly.
  useEffect(() => {
    if (saveStatus !== "saved") return;
    const handle = window.setTimeout(() => setSaveStatus("idle"), SAVED_FLASH_MS);
    return () => window.clearTimeout(handle);
  }, [saveStatus]);

  useEffect(() => {
    return () => {
      if (saveTimer.current) window.clearTimeout(saveTimer.current);
    };
  }, []);

  const update = <K extends keyof HouseholdSettings>(key: K, value: HouseholdSettings[K]) => {
    setSettings((prev) => (prev ? { ...prev, [key]: value } : prev));
    if (key === "language") setLang(value as "sv" | "en");
    scheduleSave({ [key]: value } as Partial<HouseholdSettings>);
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-30 flex items-center justify-center bg-stone-900/40 p-3 backdrop-blur-[1px] sm:p-6 dark:bg-black/60"
      onClick={onClose}
      onKeyDown={(e) => e.key === "Escape" && onClose()}
      role="button"
      tabIndex={-1}
      aria-label={t("settings.close")}
    >
      <div
        className="flex max-h-[90vh] w-full max-w-lg flex-col rounded-lg border border-stone-200 bg-white shadow-xl dark:border-stone-800 dark:bg-stone-900"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={t("settings.title")}
      >
        <header className="flex shrink-0 items-center justify-between gap-3 border-b border-stone-200 px-5 py-3 dark:border-stone-800">
          <div className="flex items-baseline gap-3">
            <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
              {t("settings.title")}
            </h2>
            <SaveIndicator status={saveStatus} />
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-stone-500 hover:bg-stone-100 dark:hover:bg-stone-800"
            aria-label={t("settings.close")}
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
          {!settings && !error && <p className="text-sm text-stone-500">{t("settings.loading")}</p>}
          {settings && (
            <div className="flex flex-col gap-4">
              <Field label={t("settings.theme")}>
                <ThemePicker value={theme} onChange={setTheme} />
              </Field>
              <Field label={t("settings.language")}>
                <select
                  value={settings.language}
                  onChange={(e) => update("language", e.target.value as "sv" | "en")}
                  className="w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
                >
                  <option value="sv">{t("settings.language_sv")}</option>
                  <option value="en">{t("settings.language_en")}</option>
                </select>
              </Field>
              <Field label={t("settings.num_dinners")} hint={t("settings.num_dinners_hint")}>
                <input
                  type="number"
                  value={settings.default_dinners}
                  min={1}
                  max={14}
                  onChange={(e) =>
                    update("default_dinners", Number.parseInt(e.target.value, 10) || 1)
                  }
                  className="w-32 rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
                />
              </Field>
              <Field
                label={t("settings.default_servings")}
                hint={t("settings.default_servings_hint")}
              >
                <input
                  type="number"
                  value={settings.default_servings}
                  min={1}
                  max={20}
                  onChange={(e) =>
                    update("default_servings", Number.parseInt(e.target.value, 10) || 1)
                  }
                  className="w-32 rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
                />
              </Field>
              <Field label={t("settings.notes")}>
                <textarea
                  value={settings.notes_md}
                  onChange={(e) => update("notes_md", e.target.value)}
                  rows={3}
                  className="w-full resize-none rounded-md border border-stone-300 bg-white px-3 py-2 text-sm shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
                />
              </Field>
            </div>
          )}
          <IntegrationsSection />
          <BackupsSection />
          <UpdatesSection />
          <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-3 dark:border-stone-800">
            <button
              type="button"
              onClick={() => navigate({ kind: "usage" })}
              className="text-xs font-medium text-stone-700 underline-offset-2 hover:underline dark:text-stone-300"
            >
              {t("usage.open")}
            </button>
            <SettingsVersion />
          </div>
        </div>
      </div>
    </div>
  );
}

// SettingsVersion shows the running build at the bottom of Settings so
// users have one obvious place to see "what version am I on" when
// reporting issues or comparing against the update banner.
function SettingsVersion() {
  const [info, setInfo] = useState<VersionInfo | null>(null);
  useEffect(() => {
    let cancelled = false;
    getVersion()
      .then((v) => {
        if (!cancelled) setInfo(v);
      })
      .catch(() => {
        /* hide on failure */
      });
    return () => {
      cancelled = true;
    };
  }, []);
  if (!info) return null;
  const isRelease = /^\d+\.\d+\.\d+$/.test(info.version);
  const label = isRelease ? `v${info.version}` : info.version;
  const className = "text-xs tabular-nums text-stone-500 dark:text-stone-400";
  if (!isRelease) {
    return (
      <span className={className} title={info.commit}>
        {label}
      </span>
    );
  }
  return (
    <a
      href={`https://github.com/simonnordberg/veckomenyn/releases/tag/v${info.version}`}
      target="_blank"
      rel="noreferrer"
      className={`${className} underline-offset-2 hover:underline`}
      title={info.commit}
    >
      {label}
    </a>
  );
}

function SaveIndicator({ status }: { status: SaveStatus }) {
  if (status === "idle" || status === "error") return null;
  return (
    <span className="text-xs text-stone-500 dark:text-stone-400" aria-live="polite" role="status">
      {status === "saving" ? t("settings.saving") : t("settings.saved")}
    </span>
  );
}

function ThemePicker({ value, onChange }: { value: Theme; onChange: (t: Theme) => void }) {
  const opts: { id: Theme; label: string }[] = [
    { id: "system", label: t("settings.theme_system") },
    { id: "light", label: t("settings.theme_light") },
    { id: "dark", label: t("settings.theme_dark") },
  ];
  return (
    <div
      role="radiogroup"
      aria-label={t("settings.theme")}
      className="inline-flex rounded-md border border-stone-300 bg-white p-0.5 text-xs dark:border-stone-700 dark:bg-stone-800"
    >
      {opts.map((o) => (
        <button
          key={o.id}
          type="button"
          role="radio"
          aria-checked={value === o.id}
          onClick={() => onChange(o.id)}
          className={`rounded px-3 py-1 font-medium transition-colors ${
            value === o.id
              ? "bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900"
              : "text-stone-700 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-700"
          }`}
        >
          {o.label}
        </button>
      ))}
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
