import { useCallback, useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import {
  type Backup,
  type BackupConfig,
  backupDownloadURL,
  deleteBackup,
  getBackupConfig,
  listBackups,
  patchBackupConfig,
  takeBackup,
} from "../lib/api";

export function BackupsSection() {
  useLang();
  const [backups, setBackups] = useState<Backup[] | null>(null);
  const [cfg, setCfg] = useState<BackupConfig | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<"idle" | "taking" | "saving">("idle");

  const refresh = useCallback(async () => {
    try {
      const [list, c] = await Promise.all([listBackups(), getBackupConfig()]);
      setBackups(list);
      setCfg(c);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const onTake = async () => {
    setBusy("taking");
    setError(null);
    try {
      await takeBackup();
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("idle");
    }
  };

  const onDelete = async (filename: string) => {
    if (!window.confirm(t("backups.delete_confirm"))) return;
    setError(null);
    try {
      await deleteBackup(filename);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const onToggleNightly = async (enabled: boolean) => {
    setBusy("saving");
    setError(null);
    try {
      const next = await patchBackupConfig({ nightly_enabled: enabled });
      setCfg(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("idle");
    }
  };

  const onChangeKeep = async (keep: number) => {
    if (!Number.isFinite(keep) || keep <= 0) return;
    setBusy("saving");
    setError(null);
    try {
      const next = await patchBackupConfig({ nightly_keep: keep });
      setCfg(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("idle");
    }
  };

  return (
    <section className="mt-6 border-t border-stone-200 pt-4 dark:border-stone-800">
      <h3 className="font-serif text-base text-stone-900 dark:text-stone-100">
        {t("backups.title")}
      </h3>
      <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">{t("backups.subtitle")}</p>

      {error && (
        <div className="mt-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200">
          {error}
        </div>
      )}

      {cfg && (
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-stone-700 dark:text-stone-300">
            <input
              type="checkbox"
              checked={cfg.nightly_enabled}
              disabled={!cfg.can_write || busy === "saving"}
              onChange={(e) => void onToggleNightly(e.target.checked)}
            />
            <span>{t("backups.nightly")}</span>
          </label>
          {cfg.nightly_enabled && (
            <label className="flex items-center gap-2 text-sm text-stone-700 dark:text-stone-300">
              <span>{t("backups.keep")}</span>
              <input
                type="number"
                min={1}
                value={cfg.nightly_keep}
                disabled={busy === "saving"}
                onChange={(e) => void onChangeKeep(Number.parseInt(e.target.value, 10))}
                className="w-20 rounded-md border border-stone-300 bg-white px-2 py-1 text-sm dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
              />
              <span className="text-xs text-stone-500 dark:text-stone-400">
                {t("backups.keep_unit")}
              </span>
            </label>
          )}
          <button
            type="button"
            onClick={() => void onTake()}
            disabled={!cfg.can_write || busy !== "idle"}
            className="ml-auto rounded-md border border-stone-300 bg-white px-3 py-1.5 text-xs font-medium text-stone-800 shadow-sm hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:hover:bg-stone-700"
          >
            {busy === "taking" ? t("backups.taking") : t("backups.take_now")}
          </button>
        </div>
      )}

      {cfg && !cfg.can_write && (
        <p className="mt-2 text-xs text-amber-700 dark:text-amber-400">
          {t("backups.nightly_disabled")}
        </p>
      )}

      <div className="mt-3 overflow-x-auto">
        {backups && backups.length === 0 ? (
          <p className="py-3 text-xs text-stone-500 dark:text-stone-400">{t("backups.empty")}</p>
        ) : (
          <table className="w-full text-left text-xs">
            <thead className="text-stone-500 dark:text-stone-400">
              <tr>
                <th className="py-1.5 pr-3 font-medium">{t("backups.col.taken")}</th>
                <th className="py-1.5 pr-3 font-medium">{t("backups.col.reason")}</th>
                <th className="py-1.5 pr-3 font-medium tabular-nums">{t("backups.col.size")}</th>
                <th className="py-1.5" />
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-200 dark:divide-stone-800">
              {backups?.map((b) => (
                <tr key={b.filename} className="text-stone-800 dark:text-stone-200">
                  <td className="py-1.5 pr-3 tabular-nums">{formatTaken(b.created_at)}</td>
                  <td className="py-1.5 pr-3">
                    <ReasonChip reason={b.reason} label={b.label} />
                  </td>
                  <td className="py-1.5 pr-3 tabular-nums">{formatSize(b.size_bytes)}</td>
                  <td className="py-1.5">
                    <div className="flex justify-end gap-2">
                      <a
                        href={backupDownloadURL(b.filename)}
                        download
                        className="rounded px-2 py-0.5 text-stone-700 hover:bg-stone-200 dark:text-stone-300 dark:hover:bg-stone-700"
                      >
                        {t("backups.download")}
                      </a>
                      <button
                        type="button"
                        onClick={() => void onDelete(b.filename)}
                        className="rounded px-2 py-0.5 text-red-700 hover:bg-red-100 dark:text-red-300 dark:hover:bg-red-950/60"
                      >
                        {t("backups.delete")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <p className="mt-3 text-[11px] text-stone-500 dark:text-stone-400">
        {t("backups.restore_hint")}
      </p>
    </section>
  );
}

function ReasonChip({ reason, label }: { reason: Backup["reason"]; label: string }) {
  const colors: Record<string, string> = {
    "pre-migration": "bg-amber-100 text-amber-900 dark:bg-amber-950/60 dark:text-amber-200",
    manual: "bg-sky-100 text-sky-900 dark:bg-sky-950/60 dark:text-sky-200",
    nightly: "bg-emerald-100 text-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-200",
  };
  const cls = colors[reason] ?? "bg-stone-200 text-stone-800 dark:bg-stone-700 dark:text-stone-200";
  const reasonLabel = reason ? t(`backups.reason.${reason}` as Parameters<typeof t>[0]) : reason;
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`rounded px-1.5 py-0.5 font-medium ${cls}`}>{reasonLabel}</span>
      {label && <span className="text-stone-500 dark:text-stone-400">{label}</span>}
    </span>
  );
}

function formatTaken(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.valueOf())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}
