import { useSyncExternalStore } from "react";

export type ToastVariant = "success" | "error" | "info";

export type ToastAction = {
  label: string;
  onClick: () => void | Promise<void>;
};

export type Toast = {
  id: string;
  variant: ToastVariant;
  message: string;
  action?: ToastAction;
  // Auto-dismiss after this many ms. 0 means sticky until dismissed.
  duration: number;
};

export type ToastOptions = {
  action?: ToastAction;
  duration?: number;
  // Pass to update an existing toast in place (e.g. retry-with-new-message).
  id?: string;
};

const MAX_VISIBLE = 3;
const DEFAULTS: Record<ToastVariant, number> = { success: 3000, info: 3000, error: 0 };

let toasts: Toast[] = [];
let nextID = 1;
const listeners = new Set<() => void>();

function emit(): void {
  for (const fn of listeners) fn();
}

function add(variant: ToastVariant, message: string, opts?: ToastOptions): string {
  const id = opts?.id ?? `t${nextID++}`;
  const duration = opts?.duration ?? DEFAULTS[variant];
  const item: Toast = { id, variant, message, action: opts?.action, duration };
  const existing = toasts.findIndex((t) => t.id === id);
  if (existing >= 0) {
    const next = toasts.slice();
    next[existing] = item;
    toasts = next;
  } else {
    toasts = capQueue([...toasts, item]);
  }
  emit();
  return id;
}

// capQueue trims the queue down to MAX_VISIBLE, evicting non-error toasts
// (oldest first) before any error toasts. Errors are sticky and important;
// a flurry of success/info toasts shouldn't bump them off-screen.
function capQueue(next: Toast[]): Toast[] {
  if (next.length <= MAX_VISIBLE) return next;
  const result = next.slice();
  let toDrop = result.length - MAX_VISIBLE;
  for (let i = 0; i < result.length && toDrop > 0; ) {
    if (result[i].variant !== "error") {
      result.splice(i, 1);
      toDrop--;
    } else {
      i++;
    }
  }
  // If everything left is errors and we're still over cap, drop the oldest.
  if (result.length > MAX_VISIBLE) return result.slice(-MAX_VISIBLE);
  return result;
}

function dismiss(id: string): void {
  const next = toasts.filter((t) => t.id !== id);
  if (next.length === toasts.length) return;
  toasts = next;
  emit();
}

function clear(): void {
  if (toasts.length === 0) return;
  toasts = [];
  emit();
}

export const toast = {
  success(message: string, opts?: ToastOptions): string {
    return add("success", message, opts);
  },
  error(message: string, opts?: ToastOptions): string {
    return add("error", message, opts);
  },
  info(message: string, opts?: ToastOptions): string {
    return add("info", message, opts);
  },
  dismiss,
  clear,
};

function subscribe(cb: () => void): () => void {
  listeners.add(cb);
  return () => {
    listeners.delete(cb);
  };
}

function getSnapshot(): Toast[] {
  return toasts;
}

export function useToasts(): Toast[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
