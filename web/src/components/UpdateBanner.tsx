import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { applyUpdate, getUpdates, getVersion, type UpdateStatus } from "../lib/api";

const DISMISSED_KEY = "veckomenyn.update.dismissed";
const UPGRADE_COMMAND = "podman compose pull && podman compose up -d";

type ApplyState = "idle" | "applying" | "waiting" | "failed";

export function UpdateBanner() {
  useLang();
  const [status, setStatus] = useState<UpdateStatus | null>(null);
  const [dismissed, setDismissed] = useState<string>(
    () => window.localStorage.getItem(DISMISSED_KEY) ?? "",
  );
  const [copied, setCopied] = useState(false);
  const [applyState, setApplyState] = useState<ApplyState>("idle");
  const [applyError, setApplyError] = useState<string | null>(null);

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

  const onApply = async () => {
    setApplyState("applying");
    setApplyError(null);
    try {
      await applyUpdate();
    } catch (e) {
      setApplyError(e instanceof Error ? e.message : String(e));
      setApplyState("failed");
      return;
    }
    // Trigger fired. The orchestrator now pulls and recreates; this page
    // is about to lose its backend. Poll /api/version until the new
    // version reports back, then reload.
    setApplyState("waiting");
    const target = status.latest;
    const started = Date.now();
    const tick = async () => {
      if (Date.now() - started > 5 * 60 * 1000) {
        setApplyError(t("update.apply_timeout"));
        setApplyState("failed");
        return;
      }
      try {
        const v = await getVersion();
        if (v.version === target) {
          window.location.reload();
          return;
        }
      } catch {
        /* server is restarting, keep polling */
      }
      window.setTimeout(() => void tick(), 2000);
    };
    window.setTimeout(() => void tick(), 3000);
  };

  // GitHub compare URL works only when both versions are clean semver.
  const isSemver = (v: string) => /^\d+\.\d+\.\d+$/.test(v);
  const compareURL =
    isSemver(status.current) && isSemver(status.latest)
      ? `https://github.com/simonnordberg/veckomenyn/compare/v${status.current}...v${status.latest}`
      : null;

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
      {compareURL && (
        <a
          href={compareURL}
          target="_blank"
          rel="noreferrer"
          className="underline-offset-2 hover:underline"
        >
          {t("update.compare")} ↗
        </a>
      )}
      {status.can_apply ? (
        <button
          type="button"
          onClick={() => void onApply()}
          disabled={applyState === "applying" || applyState === "waiting"}
          className="rounded-md border border-amber-300 bg-white/60 px-2 py-0.5 font-medium hover:bg-white disabled:cursor-not-allowed disabled:opacity-60 dark:border-amber-800 dark:bg-amber-950/60 dark:hover:bg-amber-950"
        >
          {applyState === "idle"
            ? t("update.apply")
            : applyState === "applying"
              ? t("update.applying")
              : applyState === "waiting"
                ? t("update.waiting")
                : t("update.apply")}
        </button>
      ) : (
        <button
          type="button"
          onClick={() => void onCopy()}
          className="rounded-md border border-amber-300 bg-white/60 px-2 py-0.5 font-medium hover:bg-white dark:border-amber-800 dark:bg-amber-950/60 dark:hover:bg-amber-950"
        >
          {copied ? t("update.copied") : t("update.copy_command")}
        </button>
      )}
      {applyError && <span className="text-red-700 dark:text-red-300">{applyError}</span>}
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
