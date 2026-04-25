import { useState } from "react";
import { t, useLang } from "../i18n";
import { patchProvider, seedPreferences } from "../lib/api";
import { navigate } from "../lib/route";

type Step = "welcome" | "api_key" | "seed" | "done";

const STEPS: Step[] = ["welcome", "api_key", "seed"];

export function SetupWizard({ onComplete }: { onComplete: () => void }) {
  useLang();
  const [step, setStep] = useState<Step>("welcome");
  const [apiKey, setApiKey] = useState("");
  const [busy, setBusy] = useState<"idle" | "saving" | "seeding">("idle");
  const [seededCount, setSeededCount] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const next = () => {
    const i = STEPS.indexOf(step);
    if (i >= 0 && i < STEPS.length - 1) setStep(STEPS[i + 1]);
    else finish();
  };
  const back = () => {
    const i = STEPS.indexOf(step);
    if (i > 0) setStep(STEPS[i - 1]);
  };
  const finish = () => {
    onComplete();
    navigate({ kind: "current" }, { replace: true });
  };

  const saveKey = async () => {
    if (!apiKey.trim()) return;
    setBusy("saving");
    setError(null);
    try {
      await patchProvider("anthropic", {
        enabled: true,
        config: { api_key: apiKey.trim() },
      });
      next();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("idle");
    }
  };

  const onSeed = async () => {
    setBusy("seeding");
    setError(null);
    try {
      const r = await seedPreferences();
      setSeededCount(r.seeded);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("idle");
    }
  };

  const stepIndex = STEPS.indexOf(step);

  return (
    <div className="flex min-h-full items-center justify-center bg-stone-50 px-4 py-12 dark:bg-stone-950">
      <div className="w-full max-w-md rounded-xl border border-stone-200 bg-white p-6 shadow-sm dark:border-stone-800 dark:bg-stone-900">
        <p className="text-[11px] uppercase tracking-wider text-stone-500 dark:text-stone-400">
          {t("setup.step", { n: stepIndex + 1, total: STEPS.length })}
        </p>

        {step === "welcome" && <WelcomeStep />}
        {step === "api_key" && (
          <ApiKeyStep
            value={apiKey}
            onChange={setApiKey}
            onSubmit={() => void saveKey()}
            busy={busy === "saving"}
          />
        )}
        {step === "seed" && <SeedStep busy={busy === "seeding"} seededCount={seededCount} />}

        {error && (
          <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200">
            {error}
          </div>
        )}

        <div className="mt-5 flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={back}
            disabled={stepIndex === 0 || busy !== "idle"}
            className="rounded-md px-3 py-1.5 text-sm text-stone-600 hover:bg-stone-100 disabled:opacity-30 dark:text-stone-300 dark:hover:bg-stone-800"
          >
            {t("setup.back")}
          </button>
          {step === "welcome" && (
            <button
              type="button"
              onClick={next}
              className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
            >
              {t("setup.next")}
            </button>
          )}
          {step === "api_key" && (
            <button
              type="button"
              onClick={() => void saveKey()}
              disabled={!apiKey.trim() || busy !== "idle"}
              className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white hover:bg-stone-800 disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
            >
              {busy === "saving" ? t("setup.api_key_saving") : t("setup.api_key_save")}
            </button>
          )}
          {step === "seed" && (
            <div className="flex gap-2">
              {seededCount === null ? (
                <>
                  <button
                    type="button"
                    onClick={finish}
                    className="rounded-md px-3 py-2 text-sm text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-800"
                  >
                    {t("setup.skip")}
                  </button>
                  <button
                    type="button"
                    onClick={() => void onSeed()}
                    disabled={busy !== "idle"}
                    className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white hover:bg-stone-800 disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
                  >
                    {busy === "seeding" ? t("setup.seed_seeding") : t("setup.seed_button")}
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  onClick={finish}
                  className="rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white hover:bg-stone-800 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
                >
                  {t("setup.finish")}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function WelcomeStep() {
  return (
    <>
      <h1 className="mt-2 font-serif text-2xl text-stone-900 dark:text-stone-100">
        {t("setup.welcome_title")}
      </h1>
      <p className="mt-2 text-sm text-stone-600 dark:text-stone-400">{t("setup.welcome_body")}</p>
      <div className="mt-5 rounded-md border border-amber-200 bg-amber-50 px-3 py-3 text-xs text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
        <p className="font-semibold">{t("setup.lan_warning_title")}</p>
        <p className="mt-1">{t("setup.lan_warning_body")}</p>
      </div>
    </>
  );
}

function ApiKeyStep({
  value,
  onChange,
  onSubmit,
  busy,
}: {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
  busy: boolean;
}) {
  return (
    <>
      <h2 className="mt-2 font-serif text-xl text-stone-900 dark:text-stone-100">
        {t("setup.api_key_title")}
      </h2>
      <p className="mt-2 text-sm text-stone-600 dark:text-stone-400">{t("setup.api_key_body")}</p>
      <input
        type="password"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !busy && value.trim()) onSubmit();
        }}
        placeholder={t("setup.api_key_placeholder")}
        className="mt-4 w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm font-mono shadow-sm focus:border-stone-500 focus:outline-none dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
        ref={(el) => el?.focus()}
      />
      <p className="mt-2 text-xs text-stone-500 dark:text-stone-400">
        <a
          href="https://console.anthropic.com/settings/keys"
          target="_blank"
          rel="noreferrer"
          className="underline-offset-2 hover:underline"
        >
          {t("setup.api_key_help")} ↗
        </a>
      </p>
    </>
  );
}

function SeedStep({ busy, seededCount }: { busy: boolean; seededCount: number | null }) {
  return (
    <>
      <h2 className="mt-2 font-serif text-xl text-stone-900 dark:text-stone-100">
        {t("setup.seed_title")}
      </h2>
      <p className="mt-2 text-sm text-stone-600 dark:text-stone-400">{t("setup.seed_body")}</p>
      {seededCount !== null && (
        <p className="mt-3 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200">
          {t("setup.seed_done", { n: seededCount })}
        </p>
      )}
      {busy && (
        <p className="mt-3 text-xs text-stone-500 dark:text-stone-400">{t("setup.seed_seeding")}</p>
      )}
    </>
  );
}
