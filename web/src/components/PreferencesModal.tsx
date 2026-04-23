import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { deletePreference, listPreferences, type Preference, savePreference } from "../lib/api";

type Props = {
  open: boolean;
  onClose: () => void;
};

export function PreferencesModal({ open, onClose }: Props) {
  useLang();
  const [prefs, setPrefs] = useState<Preference[] | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [newCategory, setNewCategory] = useState("");

  const refresh = () =>
    listPreferences()
      .then((rows) => {
        setPrefs(rows);
        if (!selected && rows.length > 0) {
          setSelected(rows[0].category);
          setDraft(rows[0].body_md);
        }
      })
      .catch((e: Error) => setError(e.message));

  useEffect(() => {
    if (!open) return;
    setError(null);
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    if (!selected || !prefs) return;
    const p = prefs.find((x) => x.category === selected);
    setDraft(p?.body_md ?? "");
  }, [selected, prefs]);

  const save = async () => {
    if (!selected) return;
    setSaving(true);
    setError(null);
    try {
      const next = await savePreference(selected, draft);
      setPrefs((cur) => (cur ? cur.map((p) => (p.category === next.category ? next : p)) : cur));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const createNew = async () => {
    const cat = newCategory.trim().toLowerCase().replace(/\s+/g, "_");
    if (!cat) return;
    setSaving(true);
    setError(null);
    try {
      await savePreference(cat, "");
      setNewCategory("");
      await refresh();
      setSelected(cat);
      setDraft("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!selected) return;
    if (!confirm(t("prefs.confirm_delete", { category: selected }))) return;
    setSaving(true);
    setError(null);
    try {
      await deletePreference(selected);
      setSelected(null);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
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
        className="flex h-[85vh] w-full max-w-4xl flex-col rounded-lg border border-stone-200 bg-white shadow-xl dark:border-stone-800 dark:bg-stone-900"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <header className="flex shrink-0 items-center justify-between border-b border-stone-200 px-5 py-3 dark:border-stone-800">
          <h2 className="font-serif text-lg text-stone-900 dark:text-stone-100">
            {t("prefs.title")}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-stone-500 hover:bg-stone-100 dark:text-stone-400 dark:hover:bg-stone-800"
            aria-label={t("settings.close")}
          >
            <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
            </svg>
          </button>
        </header>
        {error && (
          <div className="m-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
            {error}
          </div>
        )}
        <div className="flex flex-1 overflow-hidden">
          <aside className="flex w-40 shrink-0 flex-col border-r border-stone-200 bg-stone-50/60 sm:w-56 dark:border-stone-800 dark:bg-stone-950/60">
            <p className="px-3 pt-3 text-xs text-stone-500 dark:text-stone-400">
              {t("prefs.subtitle")}
            </p>
            <nav className="mt-2 flex-1 overflow-y-auto px-2">
              <ul className="space-y-0.5">
                {prefs?.map((p) => (
                  <li key={p.category}>
                    <button
                      type="button"
                      onClick={() => setSelected(p.category)}
                      className={`w-full rounded-md px-2.5 py-1.5 text-left font-mono text-xs ${
                        selected === p.category
                          ? "bg-stone-200 text-stone-900 dark:bg-stone-800 dark:text-stone-50"
                          : "text-stone-700 hover:bg-stone-200/60 dark:text-stone-300 dark:hover:bg-stone-800/60"
                      }`}
                    >
                      {p.category}
                    </button>
                  </li>
                ))}
              </ul>
            </nav>
            <div className="border-t border-stone-200 p-3 dark:border-stone-800">
              <label className="block text-xs font-medium text-stone-700 dark:text-stone-300">
                {t("prefs.new_category")}
              </label>
              <div className="mt-1 flex gap-1">
                <input
                  type="text"
                  value={newCategory}
                  onChange={(e) => setNewCategory(e.target.value)}
                  placeholder={t("prefs.new_category_placeholder")}
                  className="flex-1 rounded-md border border-stone-300 bg-white px-2 py-1 text-xs shadow-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
                />
                <button
                  type="button"
                  onClick={() => void createNew()}
                  disabled={saving || !newCategory.trim()}
                  className="rounded-md bg-stone-900 px-2 py-1 text-xs text-stone-50 disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900"
                >
                  +
                </button>
              </div>
            </div>
          </aside>
          <section className="flex min-w-0 flex-1 flex-col">
            {selected ? (
              <>
                <header className="flex shrink-0 items-center justify-between border-b border-stone-200 px-4 py-2 dark:border-stone-800">
                  <span className="font-mono text-sm text-stone-700 dark:text-stone-300">
                    {selected}
                  </span>
                  <button
                    type="button"
                    onClick={() => void remove()}
                    disabled={saving}
                    className="rounded-md px-2 py-1 text-xs text-red-700 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/40"
                  >
                    {t("prefs.delete")}
                  </button>
                </header>
                <textarea
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  className="flex-1 resize-none border-0 bg-white px-4 py-3 font-mono text-sm leading-relaxed outline-none focus:ring-0 dark:bg-stone-900 dark:text-stone-100"
                  placeholder={t("prefs.body_placeholder")}
                />
                <div className="flex shrink-0 items-center justify-end gap-2 border-t border-stone-200 px-4 py-2 dark:border-stone-800">
                  <button
                    type="button"
                    onClick={() => void save()}
                    disabled={saving}
                    className="rounded-md bg-stone-900 px-3 py-1.5 text-sm text-stone-50 hover:bg-stone-800 disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
                  >
                    {saving ? t("settings.saving") : t("settings.save")}
                  </button>
                </div>
              </>
            ) : (
              <div className="flex flex-1 items-center justify-center text-sm text-stone-500 dark:text-stone-400">
                {prefs === null ? t("settings.loading") : t("prefs.pick_one")}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}
