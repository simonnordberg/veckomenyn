import { type FormEvent, useEffect, useState } from "react";
import { setLang, t, useLang } from "../i18n";
import { getSettings, type HouseholdSettings, patchSettings } from "../lib/api";
import { setTheme, type Theme, useTheme } from "../lib/theme";
import { IntegrationsSection } from "./IntegrationsSection";

type Props = {
  open: boolean;
  onClose: () => void;
};

export function SettingsModal({ open, onClose }: Props) {
  useLang();
  const { theme } = useTheme();
  const [settings, setSettings] = useState<HouseholdSettings | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  useEffect(() => {
    if (!open) return;
    setError(null);
    getSettings()
      .then(setSettings)
      .catch((e: Error) => setError(e.message));
  }, [open]);

  const update = <K extends keyof HouseholdSettings>(key: K, value: HouseholdSettings[K]) => {
    setSettings((prev) => (prev ? { ...prev, [key]: value } : prev));
    if (key === "language") setLang(value as "sv" | "en");
  };

  const save = async (e: FormEvent) => {
    e.preventDefault();
    if (!settings || saving) return;
    setSaving(true);
    setError(null);
    try {
      const next = await patchSettings(settings);
      setSettings(next);
      setLang(next.language);
      setSavedAt(Date.now());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-30 flex items-center justify-center bg-stone-900/40 backdrop-blur-[1px] dark:bg-black/60"
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
        <header className="flex shrink-0 items-center justify-between border-b border-stone-200 px-5 py-3 dark:border-stone-800">
          <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
            {t("settings.title")}
          </h2>
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
            <form onSubmit={save} className="flex flex-col gap-4">
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
              <div className="flex items-center justify-between">
                <span className="text-xs text-stone-500">{savedAt ? t("settings.saved") : ""}</span>
                <button
                  type="submit"
                  disabled={saving}
                  className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
                >
                  {saving ? t("settings.saving") : t("settings.save")}
                </button>
              </div>
            </form>
          )}
          <IntegrationsSection />
        </div>
      </div>
    </div>
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
