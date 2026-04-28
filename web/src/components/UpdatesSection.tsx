import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import {
  checkUpdates,
  getUpdateConfig,
  patchUpdateConfig,
  type UpdateConfig,
  type UpdateStatus,
} from "../lib/api";
import { UPDATES_REFRESHED_EVENT } from "../lib/events";

export function UpdatesSection() {
  useLang();
  const [cfg, setCfg] = useState<UpdateConfig | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [checking, setChecking] = useState(false);
  const [checkResult, setCheckResult] = useState<UpdateStatus | null>(null);

  useEffect(() => {
    let cancelled = false;
    getUpdateConfig()
      .then((c) => {
        if (!cancelled) setCfg(c);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const onToggle = async (enabled: boolean) => {
    setBusy(true);
    setError(null);
    try {
      const next = await patchUpdateConfig({ auto_update_enabled: enabled });
      setCfg(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const onCheck = async () => {
    setChecking(true);
    setError(null);
    setCheckResult(null);
    try {
      const status = await checkUpdates();
      setCheckResult(status);
      window.dispatchEvent(new Event(UPDATES_REFRESHED_EVENT));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setChecking(false);
    }
  };

  return (
    <section className="mt-6 border-t border-stone-200 pt-4 dark:border-stone-800">
      <h3 className="font-serif text-base text-stone-900 dark:text-stone-100">
        {t("update.section_title")}
      </h3>
      <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">
        {t("update.section_subtitle")}
      </p>

      {error && (
        <div className="mt-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200">
          {error}
        </div>
      )}

      {cfg && (
        <div className="mt-3 flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-stone-700 dark:text-stone-300">
            <input
              type="checkbox"
              checked={cfg.auto_update_enabled}
              disabled={!cfg.can_apply || busy}
              onChange={(e) => void onToggle(e.target.checked)}
            />
            <span>{t("update.auto_label")}</span>
          </label>
          {cfg.can_apply && (
            <span className="text-xs text-stone-500 dark:text-stone-400">
              {t("update.auto_hint")}
            </span>
          )}
        </div>
      )}

      {cfg && !cfg.can_apply && (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
          {t("update.auto_unavailable")}
        </p>
      )}

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={() => void onCheck()}
          disabled={checking}
          className="rounded-md border border-stone-300 bg-white px-3 py-1 text-xs font-medium text-stone-700 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-200 dark:hover:bg-stone-700"
        >
          {checking ? t("update.checking") : t("update.check_now")}
        </button>
        {checkResult && !checking && (
          <span className="text-xs text-stone-600 dark:text-stone-400">
            {checkResult.has_update
              ? t("update.found", { version: checkResult.latest })
              : t("update.up_to_date", { version: checkResult.current })}
          </span>
        )}
      </div>
    </section>
  );
}
