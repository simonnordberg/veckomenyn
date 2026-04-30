import { useEffect, useState } from "react";
import { t, useLang } from "../i18n";
import { listProviders, type Provider, type ProviderKindInfo, patchProvider } from "../lib/api";
import { toast } from "../lib/toast";

export function IntegrationsSection() {
  useLang();
  const [known, setKnown] = useState<ProviderKindInfo[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [sentinel, setSentinel] = useState("");

  const refresh = () =>
    listProviders()
      .then((env) => {
        setKnown(env.known);
        setProviders(env.providers);
        setSentinel(env.sentinel);
      })
      .catch((e: Error) => toast.error(e.message));

  useEffect(() => {
    void refresh();
  }, []);

  const llmKinds = known.filter((k) => k.category === "llm");
  const otherKinds = known.filter((k) => k.category !== "llm");

  return (
    <section className="mt-6 border-t border-stone-200 pt-4 dark:border-stone-800">
      <h3 className="font-serif text-base text-stone-900 dark:text-stone-100">
        {t("integrations.title")}
      </h3>
      <p className="mt-1 text-xs text-stone-500 dark:text-stone-400">
        {t("integrations.subtitle")}
      </p>
      <div className="mt-3 flex flex-col gap-3">
        {llmKinds.length > 0 && (
          <LLMProviderCard
            kinds={llmKinds}
            providers={providers}
            sentinel={sentinel}
            onSaved={refresh}
          />
        )}
        {otherKinds.map((info) => {
          const p = providers.find((x) => x.kind === info.kind) ?? {
            kind: info.kind,
            enabled: false,
            config: {},
          };
          return (
            <ProviderCard
              key={info.kind}
              info={info}
              provider={p}
              sentinel={sentinel}
              onSaved={refresh}
            />
          );
        })}
      </div>
    </section>
  );
}

function LLMProviderCard({
  kinds,
  providers,
  sentinel,
  onSaved,
}: {
  kinds: ProviderKindInfo[];
  providers: Provider[];
  sentinel: string;
  onSaved: () => void;
}) {
  const enabledProvider = providers.find((p) => p.enabled && kinds.some((k) => k.kind === p.kind));
  const [selectedKind, setSelectedKind] = useState(enabledProvider?.kind ?? kinds[0]?.kind ?? "");
  const selectedInfo = kinds.find((k) => k.kind === selectedKind);

  const getProvider = (kind: string) =>
    providers.find((x) => x.kind === kind) ?? { kind, enabled: false, config: {} };

  const initialConfig = (info: ProviderKindInfo, p: Provider) => {
    const c = { ...p.config };
    for (const f of info.fields) {
      if (f.type === "select" && !c[f.key] && f.default) {
        c[f.key] = f.default;
      }
    }
    return c;
  };

  const currentProvider = getProvider(selectedKind);
  const [config, setConfig] = useState<Record<string, string>>(() =>
    selectedInfo ? initialConfig(selectedInfo, currentProvider) : {},
  );
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (!selectedInfo) return;
    const p = getProvider(selectedKind);
    setConfig(initialConfig(selectedInfo, p));
  }, [selectedKind, providers]);

  const dirty = (() => {
    if (!selectedInfo) return false;
    const wasEnabled = enabledProvider?.kind === selectedKind;
    if (!wasEnabled && selectedKind !== (enabledProvider?.kind ?? kinds[0]?.kind)) return true;
    const p = getProvider(selectedKind);
    return selectedInfo.fields.some((f) => (config[f.key] ?? "") !== (p.config[f.key] ?? ""));
  })();

  const save = async () => {
    if (pending || !selectedInfo) return;
    setPending(true);
    try {
      const disableOthers = kinds
        .filter((k) => k.kind !== selectedKind)
        .map((k) => {
          const p = getProvider(k.kind);
          if (p.enabled) return patchProvider(k.kind, { enabled: false, config: p.config });
          return null;
        })
        .filter(Boolean);
      await Promise.all([...disableOthers, patchProvider(selectedKind, { enabled: true, config })]);
      toast.success(t("toast.changes_saved"));
      onSaved();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setPending(false);
    }
  };

  if (!selectedInfo) return null;

  return (
    <article className="rounded-md border border-stone-200 bg-white p-3 dark:border-stone-800 dark:bg-stone-900">
      <h4 className="text-sm font-medium text-stone-900 dark:text-stone-100">
        {t("integrations.category_llm")}
      </h4>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {kinds.map((k) => (
          <button
            key={k.kind}
            type="button"
            onClick={() => setSelectedKind(k.kind)}
            className={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              selectedKind === k.kind
                ? "bg-stone-900 text-white dark:bg-stone-100 dark:text-stone-900"
                : "bg-stone-100 text-stone-600 hover:bg-stone-200 dark:bg-stone-800 dark:text-stone-400 dark:hover:bg-stone-700"
            }`}
          >
            {k.display_name}
          </button>
        ))}
      </div>
      <div className="mt-3 flex flex-col gap-2">
        <FieldList
          fields={selectedInfo.fields}
          config={config}
          sentinel={sentinel}
          onChange={(key, val) => setConfig((c) => ({ ...c, [key]: val }))}
        />
      </div>
      <div className="mt-3 flex items-center justify-end">
        <button
          type="button"
          onClick={() => void save()}
          disabled={pending || !dirty}
          className="rounded-md bg-stone-900 px-3 py-1 text-xs font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
        >
          {pending ? t("settings.saving") : t("settings.save")}
        </button>
      </div>
    </article>
  );
}

function FieldList({
  fields,
  config,
  sentinel,
  onChange,
}: {
  fields: ProviderKindInfo["fields"];
  config: Record<string, string>;
  sentinel: string;
  onChange: (key: string, value: string) => void;
}) {
  return (
    <>
      {fields.map((f) => (
        <label
          key={f.key}
          className="flex flex-col gap-1 text-xs text-stone-700 dark:text-stone-300"
        >
          <span className="font-medium">
            {f.label}
            {f.required && <span className="text-red-500"> *</span>}
          </span>
          {f.type === "select" ? (
            <select
              value={config[f.key] ?? f.default ?? ""}
              onChange={(e) => onChange(f.key, e.target.value)}
              className="rounded-md border border-stone-300 bg-white px-2.5 py-1.5 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100"
            >
              {(f.options ?? []).map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          ) : (
            <input
              type={f.type === "password" ? "password" : "text"}
              value={config[f.key] ?? ""}
              onChange={(e) => onChange(f.key, e.target.value)}
              placeholder={f.placeholder ?? ""}
              className="rounded-md border border-stone-300 bg-white px-2.5 py-1.5 text-sm shadow-sm outline-none focus:border-stone-500 dark:border-stone-700 dark:bg-stone-800 dark:text-stone-100 dark:placeholder:text-stone-500"
            />
          )}
          {f.hint && (
            <span className="text-[10px] text-stone-500 dark:text-stone-400">{f.hint}</span>
          )}
          {f.type === "password" && sentinel !== "" && config[f.key] === sentinel && (
            <span className="text-[10px] text-stone-400 dark:text-stone-500">
              {t("integrations.secret_set_hint")}
            </span>
          )}
        </label>
      ))}
    </>
  );
}

function ProviderCard({
  info,
  provider,
  sentinel,
  onSaved,
}: {
  info: ProviderKindInfo;
  provider: Provider;
  sentinel: string;
  onSaved: () => void;
}) {
  const initialConfig = (p: Provider) => {
    const c = { ...p.config };
    for (const f of info.fields) {
      if (f.type === "select" && !c[f.key] && f.default) {
        c[f.key] = f.default;
      }
    }
    return c;
  };
  const [enabled, setEnabled] = useState(provider.enabled);
  const [config, setConfig] = useState<Record<string, string>>(() => initialConfig(provider));
  const [pending, setPending] = useState(false);

  useEffect(() => {
    setEnabled(provider.enabled);
    setConfig(initialConfig(provider));
  }, [provider]);

  const dirty =
    enabled !== provider.enabled ||
    info.fields.some((f) => (config[f.key] ?? "") !== (provider.config[f.key] ?? ""));

  const save = async () => {
    if (pending || !dirty) return;
    setPending(true);
    try {
      await patchProvider(info.kind, { enabled, config });
      toast.success(t("toast.changes_saved"));
      onSaved();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setPending(false);
    }
  };

  return (
    <article className="rounded-md border border-stone-200 bg-white p-3 dark:border-stone-800 dark:bg-stone-900">
      <header className="flex items-start justify-between gap-3">
        <div>
          <h4 className="text-sm font-medium text-stone-900 dark:text-stone-100">
            {info.display_name}
          </h4>
          <p className="text-xs text-stone-500 dark:text-stone-400">
            {t("integrations.category_shopping")}
          </p>
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-xs text-stone-700 dark:text-stone-300">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="h-4 w-4 accent-stone-900 dark:accent-stone-100"
          />
          {t("integrations.enabled")}
        </label>
      </header>
      <div className="mt-3 flex flex-col gap-2">
        <FieldList
          fields={info.fields}
          config={config}
          sentinel={sentinel}
          onChange={(key, val) => setConfig((c) => ({ ...c, [key]: val }))}
        />
      </div>
      <div className="mt-3 flex items-center justify-end">
        <button
          type="button"
          onClick={() => void save()}
          disabled={pending || !dirty}
          className="rounded-md bg-stone-900 px-3 py-1 text-xs font-medium text-stone-50 shadow-sm hover:bg-stone-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-stone-100 dark:text-stone-900 dark:hover:bg-stone-200"
        >
          {pending ? t("settings.saving") : t("settings.save")}
        </button>
      </div>
    </article>
  );
}
