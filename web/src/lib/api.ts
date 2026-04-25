export type AgentEvent = {
  type: "text" | "tool_call_started" | "tool_call" | "tool_result" | "error" | "cancelled" | "done";
  text?: string;
  tool?: string;
  tool_id?: string;
  input?: string;
  result?: string;
  is_error?: boolean;
};

export type ChatMeta = { conversation_id: number };

export type Conversation = {
  id: number;
  title: string;
  started_at: string;
  updated_at: string;
};

export type Message = {
  id: number;
  role: "user" | "assistant" | "tool";
  text: string;
  created_at: string;
};

export type WeekSummary = {
  id: number;
  iso_week: string;
  start_date: string;
  end_date: string;
  delivery_date: string | null;
  order_date: string | null;
  status: "draft" | "cart_built" | "ordered";
  dinner_count: number;
  updated_at: string;
};

export type DinnerRating = "loved" | "liked" | "meh" | "disliked";

export type Dinner = {
  id: number;
  day_date: string;
  dish_id: number | null;
  dish_name: string;
  cuisine: string | null;
  servings: number;
  sourcing: Record<string, string>;
  recipe_md: string;
  notes: string;
  rating: DinnerRating | null;
  rating_notes: string;
};

export async function setDinnerRating(
  dinnerID: number,
  rating: DinnerRating,
  notes: string,
): Promise<void> {
  const r = await fetch(`/api/dinners/id/${dinnerID}/rating`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rating, notes }),
  });
  if (!r.ok) throw new Error(`set rating: ${r.status} ${await r.text()}`);
}

export async function clearDinnerRating(dinnerID: number): Promise<void> {
  const r = await fetch(`/api/dinners/id/${dinnerID}/rating`, { method: "DELETE" });
  if (!r.ok) throw new Error(`clear rating: ${r.status}`);
}

export async function setWeekRetrospective(weekID: number, notesMD: string): Promise<void> {
  const r = await fetch(`/api/weeks/id/${weekID}/retrospective`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ notes_md: notesMD }),
  });
  if (!r.ok) throw new Error(`set retrospective: ${r.status} ${await r.text()}`);
}

export type Exception = {
  id: number;
  kind: string;
  description: string;
};

export type Retrospective = {
  id: number;
  notes_md: string;
  created_at: string;
};

export type CartItemSnapshot = {
  name?: string;
  unit_price?: number;
  line_total?: number;
};

export type CartItem = {
  id: number;
  product_code: string;
  qty: number;
  reason_md: string;
  committed: boolean;
  snapshot: CartItemSnapshot;
};

export type WeekDetail = WeekSummary & {
  notes_md: string;
  dinners: Dinner[];
  exceptions: Exception[];
  retrospectives: Retrospective[];
  cart_items: CartItem[];
};

export async function listWeeks(): Promise<WeekSummary[]> {
  const r = await fetch("/api/weeks");
  if (!r.ok) throw new Error(`list weeks: ${r.status}`);
  const j = (await r.json()) as { weeks: WeekSummary[] };
  return j.weeks;
}

export async function getCurrentWeek(): Promise<WeekDetail | null> {
  const r = await fetch("/api/weeks/current");
  if (r.status === 204) return null;
  if (!r.ok) throw new Error(`current week: ${r.status}`);
  return (await r.json()) as WeekDetail;
}

export type WeekCreate = {
  start_date: string;
  end_date: string;
  notes_md: string;
};

export async function createWeek(input: WeekCreate): Promise<WeekDetail> {
  const r = await fetch("/api/weeks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!r.ok) throw new Error(`create week: ${r.status} ${await r.text()}`);
  return (await r.json()) as WeekDetail;
}

export async function getWeek(iso: string): Promise<WeekDetail> {
  const r = await fetch(`/api/weeks/${encodeURIComponent(iso)}`);
  if (!r.ok) throw new Error(`get week: ${r.status}`);
  return (await r.json()) as WeekDetail;
}

export async function getWeekById(id: number): Promise<WeekDetail> {
  const r = await fetch(`/api/weeks/id/${id}`);
  if (!r.ok) throw new Error(`get week: ${r.status}`);
  return (await r.json()) as WeekDetail;
}

// Tools whose completion means the current week's structured data changed.
// The UI refetches after any of these to keep the view live.
export const MUTATING_TOOLS = new Set([
  "create_week",
  "update_week",
  "add_dinner",
  "update_dinner",
  "delete_dinner",
  "add_exception",
  "record_retrospective",
  "update_preference",
  "update_household_settings",
]);

export type WeekPatch = Partial<{
  iso_week: string;
  start_date: string;
  end_date: string;
  delivery_date: string | null;
  order_date: string | null;
  status: WeekSummary["status"];
  notes_md: string;
}>;

export async function cloneWeek(
  sourceID: number,
  opts?: { start_date?: string },
): Promise<WeekDetail> {
  const body = opts?.start_date ? JSON.stringify({ start_date: opts.start_date }) : undefined;
  const r = await fetch(`/api/weeks/id/${sourceID}/clone`, {
    method: "POST",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body,
  });
  if (!r.ok) throw new Error(`clone week: ${r.status} ${await r.text()}`);
  return (await r.json()) as WeekDetail;
}

export async function deleteWeek(id: number): Promise<void> {
  const r = await fetch(`/api/weeks/id/${id}`, { method: "DELETE" });
  if (!r.ok && r.status !== 404) throw new Error(`delete week: ${r.status} ${await r.text()}`);
}

export async function patchWeek(id: number, patch: WeekPatch): Promise<WeekDetail> {
  const r = await fetch(`/api/weeks/id/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!r.ok) throw new Error(`patch week: ${r.status} ${await r.text()}`);
  return (await r.json()) as WeekDetail;
}

export type HouseholdSettings = {
  default_dinners: number;
  default_delivery_weekday: number; // 1=Mon … 7=Sun
  default_order_offset_days: number;
  default_servings: number;
  language: "sv" | "en";
  notes_md: string;
};

export async function getSettings(): Promise<HouseholdSettings> {
  const r = await fetch("/api/settings");
  if (!r.ok) throw new Error(`get settings: ${r.status}`);
  return (await r.json()) as HouseholdSettings;
}

export async function patchSettings(patch: Partial<HouseholdSettings>): Promise<HouseholdSettings> {
  const r = await fetch("/api/settings", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!r.ok) throw new Error(`patch settings: ${r.status} ${await r.text()}`);
  return (await r.json()) as HouseholdSettings;
}

// Providers (integrations)

export type ProviderFieldOption = { value: string; label: string };

export type ProviderField = {
  key: string;
  label: string;
  type: "text" | "password" | "select";
  options?: ProviderFieldOption[];
  default?: string;
  placeholder?: string;
  required?: boolean;
  hint?: string;
};

export type ProviderKindInfo = {
  kind: string;
  category: "llm" | "shopping";
  display_name: string;
  fields: ProviderField[];
};

export type Provider = {
  kind: string;
  enabled: boolean;
  config: Record<string, string>;
  updated_at?: string;
};

export type ProvidersEnvelope = {
  known: ProviderKindInfo[];
  providers: Provider[];
  // Random per-process sentinel returned in place of secret values. Echo it
  // back unchanged on PATCH to mean "leave this secret alone".
  sentinel: string;
};

export async function listProviders(): Promise<ProvidersEnvelope> {
  const r = await fetch("/api/providers");
  if (!r.ok) throw new Error(`list providers: ${r.status}`);
  return (await r.json()) as ProvidersEnvelope;
}

export async function patchProvider(
  kind: string,
  patch: { enabled?: boolean; config?: Record<string, string> },
): Promise<Provider> {
  const r = await fetch(`/api/providers/${encodeURIComponent(kind)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!r.ok) throw new Error(`patch provider: ${r.status} ${await r.text()}`);
  return (await r.json()) as Provider;
}

// Preferences

export type Preference = {
  category: string;
  body_md: string;
  updated_at: string;
};

export async function listPreferences(): Promise<Preference[]> {
  const r = await fetch("/api/preferences");
  if (!r.ok) throw new Error(`list preferences: ${r.status}`);
  const j = (await r.json()) as { preferences: Preference[] };
  return j.preferences;
}

export async function savePreference(category: string, bodyMD: string): Promise<Preference> {
  const r = await fetch(`/api/preferences/${encodeURIComponent(category)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ body_md: bodyMD }),
  });
  if (!r.ok) throw new Error(`save preference: ${r.status} ${await r.text()}`);
  return (await r.json()) as Preference;
}

export async function deletePreference(category: string): Promise<void> {
  const r = await fetch(`/api/preferences/${encodeURIComponent(category)}`, {
    method: "DELETE",
  });
  if (!r.ok) throw new Error(`delete preference: ${r.status}`);
}

export async function listConversations(): Promise<Conversation[]> {
  const r = await fetch("/api/conversations");
  if (!r.ok) throw new Error(`list conversations: ${r.status}`);
  const j = (await r.json()) as { conversations: Conversation[] };
  return j.conversations;
}

export async function getConversation(
  id: number,
): Promise<{ conversation: Conversation; messages: Message[] }> {
  const r = await fetch(`/api/conversations/${id}`);
  if (!r.ok) throw new Error(`get conversation: ${r.status}`);
  return (await r.json()) as { conversation: Conversation; messages: Message[] };
}

export async function getWeekConversation(
  weekID: number,
): Promise<{ conversation: Conversation; messages: Message[] } | null> {
  const r = await fetch(`/api/weeks/id/${weekID}/conversation`);
  if (r.status === 204) return null;
  if (!r.ok) throw new Error(`get week conversation: ${r.status}`);
  return (await r.json()) as { conversation: Conversation; messages: Message[] };
}

export async function deleteWeekConversations(weekID: number): Promise<void> {
  const r = await fetch(`/api/weeks/id/${weekID}/conversations`, { method: "DELETE" });
  if (!r.ok) throw new Error(`delete week conversations: ${r.status}`);
}

// Usage (LLM cost tracking)

export type UsageTotals = {
  cost_usd: number;
  input_tokens: number;
  cache_creation_input_tokens: number;
  cache_read_input_tokens: number;
  output_tokens: number;
  calls: number;
};

export type UsageDay = { date: string; cost_usd: number; calls: number };

export type UsageModel = UsageTotals & { model: string };

export type UsageWeek = {
  week_id: number;
  iso_week: string;
  cost_usd: number;
  calls: number;
};

export type UsageConversation = {
  conversation_id: number;
  title: string;
  week_id?: number;
  iso_week?: string;
  cost_usd: number;
  calls: number;
  last_used_at: string;
};

export type UsageSummary = {
  window_days: number;
  total: UsageTotals;
  by_day: UsageDay[];
  by_model: UsageModel[];
  by_week: UsageWeek[];
  recent_conversations: UsageConversation[];
};

export async function getUsageSummary(): Promise<UsageSummary> {
  const r = await fetch("/api/usage/summary");
  if (!r.ok) throw new Error(`usage summary: ${r.status}`);
  return (await r.json()) as UsageSummary;
}

export type VersionInfo = {
  version: string;
  commit: string;
  built_at: string;
};

export async function getVersion(): Promise<VersionInfo> {
  const r = await fetch("/api/version");
  if (!r.ok) throw new Error(`version: ${r.status}`);
  return (await r.json()) as VersionInfo;
}

export type UpdateStatus = {
  current: string;
  latest: string;
  has_update: boolean;
  url: string;
  can_apply: boolean;
  auto_enabled: boolean;
};

export async function getUpdates(): Promise<UpdateStatus> {
  const r = await fetch("/api/updates");
  if (!r.ok) throw new Error(`updates: ${r.status}`);
  return (await r.json()) as UpdateStatus;
}

export async function applyUpdate(): Promise<void> {
  const r = await fetch("/api/updates/apply", { method: "POST" });
  if (!r.ok && r.status !== 202) throw new Error(await r.text());
}

export type UpdateConfig = {
  auto_update_enabled: boolean;
  can_apply: boolean;
};

export async function getUpdateConfig(): Promise<UpdateConfig> {
  const r = await fetch("/api/update-config");
  if (!r.ok) throw new Error(`update-config: ${r.status}`);
  return (await r.json()) as UpdateConfig;
}

export async function patchUpdateConfig(p: {
  auto_update_enabled: boolean;
}): Promise<UpdateConfig> {
  const r = await fetch("/api/update-config", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(p),
  });
  if (!r.ok) throw new Error(await r.text());
  return (await r.json()) as UpdateConfig;
}

export type SetupStatus = {
  setup_complete: boolean;
  has_anthropic_key: boolean;
  has_preferences: boolean;
  has_family_members: boolean;
};

export async function getSetupStatus(): Promise<SetupStatus> {
  const r = await fetch("/api/setup-status");
  if (!r.ok) throw new Error(`setup-status: ${r.status}`);
  return (await r.json()) as SetupStatus;
}

export async function seedPreferences(): Promise<{ seeded: number }> {
  const r = await fetch("/api/preferences/seed", { method: "POST" });
  if (!r.ok) throw new Error(await r.text());
  return (await r.json()) as { seeded: number };
}

export type BackupReason = "pre-migration" | "manual" | "nightly" | "";

export type Backup = {
  filename: string;
  reason: BackupReason;
  label: string;
  size_bytes: number;
  created_at: string;
};

export async function listBackups(): Promise<Backup[]> {
  const r = await fetch("/api/backups");
  if (!r.ok) throw new Error(`backups: ${r.status}`);
  const body = (await r.json()) as { backups: Backup[] };
  return body.backups;
}

export async function takeBackup(): Promise<Backup> {
  const r = await fetch("/api/backups", { method: "POST" });
  if (!r.ok) throw new Error(await r.text());
  return (await r.json()) as Backup;
}

export async function deleteBackup(filename: string): Promise<void> {
  const r = await fetch(`/api/backups/${encodeURIComponent(filename)}`, { method: "DELETE" });
  if (!r.ok) throw new Error(await r.text());
}

export function backupDownloadURL(filename: string): string {
  return `/api/backups/${encodeURIComponent(filename)}/download`;
}

export type BackupConfig = {
  nightly_enabled: boolean;
  nightly_keep: number;
  can_write: boolean;
};

export async function getBackupConfig(): Promise<BackupConfig> {
  const r = await fetch("/api/backup-config");
  if (!r.ok) throw new Error(`backup-config: ${r.status}`);
  return (await r.json()) as BackupConfig;
}

export async function patchBackupConfig(p: Partial<BackupConfig>): Promise<BackupConfig> {
  const r = await fetch("/api/backup-config", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(p),
  });
  if (!r.ok) throw new Error(await r.text());
  return (await r.json()) as BackupConfig;
}

export type StreamHandlers = {
  onMeta?: (m: ChatMeta) => void;
  onEvent?: (e: AgentEvent) => void;
  onEnd?: (m: ChatMeta) => void;
  onError?: (err: Error) => void;
};

// streamChat POSTs to /api/chat and parses the SSE stream.
export async function streamChat(
  body: { conversation_id?: number; week_id?: number; message: string },
  handlers: StreamHandlers,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch("/api/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok || !response.body) {
    throw new Error(`chat: ${response.status} ${response.statusText}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let frameEnd: number;
    // biome-ignore lint/suspicious/noAssignInExpressions: SSE framing by blank-line
    while ((frameEnd = buffer.indexOf("\n\n")) >= 0) {
      const frame = buffer.slice(0, frameEnd);
      buffer = buffer.slice(frameEnd + 2);
      const parsed = parseFrame(frame);
      if (!parsed) continue;
      try {
        if (parsed.event === "meta") handlers.onMeta?.(parsed.data as ChatMeta);
        else if (parsed.event === "event") handlers.onEvent?.(parsed.data as AgentEvent);
        else if (parsed.event === "end") handlers.onEnd?.(parsed.data as ChatMeta);
      } catch (err) {
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)));
      }
    }
  }
}

function parseFrame(frame: string): { event: string; data: unknown } | null {
  let event = "message";
  const dataLines: string[] = [];
  for (const rawLine of frame.split("\n")) {
    if (rawLine.startsWith("event: ")) event = rawLine.slice(7).trim();
    else if (rawLine.startsWith("data: ")) dataLines.push(rawLine.slice(6));
  }
  if (dataLines.length === 0) return null;
  try {
    return { event, data: JSON.parse(dataLines.join("\n")) as unknown };
  } catch {
    return null;
  }
}
