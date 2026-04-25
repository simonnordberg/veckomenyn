import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { applyUpdate, getUpdates, getVersion, type UpdateStatus } from "../lib/api";

const DISMISSED_KEY = "veckomenyn.update.dismissed";
const UPGRADED_FROM_KEY = "veckomenyn.update.upgraded_from";
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
  const [upgradedFrom, setUpgradedFrom] = useState<string | null>(null);
  const [upgradedTo, setUpgradedTo] = useState<string | null>(null);

  // After a triggered update reloads the page, check whether the running
  // version differs from the version we recorded just before triggering.
  // If it does, show a one-shot success toast, then clear the marker so
  // the next page load is silent.
  useEffect(() => {
    const from = window.sessionStorage.getItem(UPGRADED_FROM_KEY);
    if (!from) return;
    let cancelled = false;
    getVersion()
      .then((v) => {
        if (cancelled) return;
        if (v.version && v.version !== from) {
          setUpgradedFrom(from);
          setUpgradedTo(v.version);
        }
        window.sessionStorage.removeItem(UPGRADED_FROM_KEY);
      })
      .catch(() => {
        /* version unreachable; leave marker for next reload */
      });
    return () => {
      cancelled = true;
    };
  }, []);

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

  // Post-update success toast, rendered independently of the
  // available-update banner so it shows even when there's nothing new.
  const successToast =
    upgradedTo && upgradedFrom !== upgradedTo ? (
      <UpdateSuccessToast
        from={upgradedFrom ?? "?"}
        to={upgradedTo}
        onDismiss={() => {
          setUpgradedFrom(null);
          setUpgradedTo(null);
        }}
      />
    ) : null;

  if (!status?.has_update || dismissed === status.latest) return successToast;

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
    // Stash the current version in sessionStorage so the post-reload
    // banner can recognise that an upgrade just landed.
    window.sessionStorage.setItem(UPGRADED_FROM_KEY, status.current);
    try {
      await applyUpdate();
    } catch (e) {
      window.sessionStorage.removeItem(UPGRADED_FROM_KEY);
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

function UpdateSuccessToast({
  from,
  to,
  onDismiss,
}: {
  from: string;
  to: string;
  onDismiss: () => void;
}) {
  // Auto-dismiss after a few seconds; user can also close manually.
  useEffect(() => {
    const id = window.setTimeout(onDismiss, 8000);
    return () => window.clearTimeout(id);
  }, [onDismiss]);
  const isSemver = (v: string) => /^\d+\.\d+\.\d+$/.test(v);
  const notesURL = isSemver(to)
    ? `https://github.com/simonnordberg/veckomenyn/releases/tag/v${to}`
    : null;
  return (
    <div className="flex flex-wrap items-center gap-3 border-b border-emerald-200 bg-emerald-50 px-4 py-2 text-xs text-emerald-900 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200">
      <span className="font-medium">{t("update.success", { from, to })}</span>
      {notesURL && (
        <a
          href={notesURL}
          target="_blank"
          rel="noreferrer"
          className="underline-offset-2 hover:underline"
        >
          {t("update.notes")} ↗
        </a>
      )}
      <button
        type="button"
        onClick={onDismiss}
        className="ml-auto rounded-md px-2 py-0.5 text-emerald-700 hover:bg-emerald-100 dark:text-emerald-300 dark:hover:bg-emerald-900/40"
      >
        {t("update.dismiss")}
      </button>
    </div>
  );
}
