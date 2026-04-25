import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { getUpdates, type UpdateStatus } from "../lib/api";

const DISMISSED_KEY = "veckomenyn.update.dismissed";
const UPGRADE_COMMAND = "podman compose pull && podman compose up -d";

export function UpdateBanner() {
  useLang();
  const [status, setStatus] = useState<UpdateStatus | null>(null);
  const [dismissed, setDismissed] = useState<string>(
    () => window.localStorage.getItem(DISMISSED_KEY) ?? "",
  );
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    getUpdates()
      .then((s) => {
        if (!cancelled) setStatus(s);
      })
      .catch(() => {
        /* silently keep banner hidden on error */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!status?.has_update || dismissed === status.latest) return null;

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(UPGRADE_COMMAND);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard denied; user can still copy manually */
    }
  };

  const onDismiss = () => {
    window.localStorage.setItem(DISMISSED_KEY, status.latest);
    setDismissed(status.latest);
  };

  return (
    <div className="flex flex-wrap items-center gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
      <span className="font-medium">{t("update.available", { version: status.latest })}</span>
      {status.url && (
        <a
          href={status.url}
          target="_blank"
          rel="noreferrer"
          className="underline-offset-2 hover:underline"
        >
          {t("update.notes")} ↗
        </a>
      )}
      <button
        type="button"
        onClick={() => void onCopy()}
        className="rounded-md border border-amber-300 bg-white/60 px-2 py-0.5 font-medium hover:bg-white dark:border-amber-800 dark:bg-amber-950/60 dark:hover:bg-amber-950"
      >
        {copied ? t("update.copied") : t("update.copy_command")}
      </button>
      <button
        type="button"
        onClick={onDismiss}
        className="ml-auto rounded-md px-2 py-0.5 text-amber-700 hover:bg-amber-100 dark:text-amber-300 dark:hover:bg-amber-900/40"
      >
        {t("update.dismiss")}
      </button>
    </div>
  );
}
