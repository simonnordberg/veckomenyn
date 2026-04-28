import { useEffect } from "react";
import { t, useLang } from "../i18n";
import { type Toast, toast as toastAPI, useToasts } from "../lib/toast";

export function ToastViewport() {
  useLang();
  const items = useToasts();

  // Escape dismisses the most recent toast. Only registered while there's
  // something to dismiss so we don't swallow Escape elsewhere.
  useEffect(() => {
    if (items.length === 0) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      toastAPI.dismiss(items[items.length - 1].id);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [items]);

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex flex-col items-center gap-2 p-4 sm:items-end">
      {items.map((item) => (
        <ToastItem key={item.id} toast={item} />
      ))}
    </div>
  );
}

function ToastItem({ toast: item }: { toast: Toast }) {
  useEffect(() => {
    if (item.duration <= 0) return;
    const handle = window.setTimeout(() => toastAPI.dismiss(item.id), item.duration);
    return () => window.clearTimeout(handle);
  }, [item.id, item.duration]);

  const surface = surfaceClass(item.variant);

  return (
    <div
      role={item.variant === "error" ? "alert" : "status"}
      aria-live={item.variant === "error" ? "assertive" : "polite"}
      className={`pointer-events-auto flex w-full max-w-md items-start gap-3 rounded-md border px-4 py-3 shadow-md motion-safe:animate-[toast-in_180ms_ease-out] ${surface}`}
    >
      <ToastIcon variant={item.variant} />
      <p className="flex-1 text-sm leading-snug">{item.message}</p>
      {item.action && (
        <button
          type="button"
          onClick={() => {
            void item.action?.onClick();
            toastAPI.dismiss(item.id);
          }}
          className="shrink-0 rounded-md border border-current/25 px-2 py-1 text-xs font-medium hover:bg-black/5 dark:hover:bg-white/10"
        >
          {item.action.label}
        </button>
      )}
      <button
        type="button"
        aria-label={t("toast.dismiss")}
        onClick={() => toastAPI.dismiss(item.id)}
        className="-mt-1 -mr-1 shrink-0 rounded-md p-1 opacity-60 hover:bg-black/5 hover:opacity-100 dark:hover:bg-white/10"
      >
        <svg width="14" height="14" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" />
        </svg>
      </button>
    </div>
  );
}

function surfaceClass(variant: Toast["variant"]): string {
  switch (variant) {
    case "success":
      return "border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-100";
    case "error":
      return "border-red-300 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-100";
    case "info":
      return "border-stone-300 bg-white text-stone-900 dark:border-stone-700 dark:bg-stone-900 dark:text-stone-100";
  }
}

function ToastIcon({ variant }: { variant: Toast["variant"] }) {
  if (variant === "success") {
    return (
      <svg
        width="18"
        height="18"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        className="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400"
      >
        <path d="M16.704 5.29a1 1 0 0 1 .006 1.414l-7.07 7.13a1 1 0 0 1-1.42.005l-3.93-3.93a1 1 0 1 1 1.42-1.41l3.22 3.22 6.36-6.42a1 1 0 0 1 1.414-.009Z" />
      </svg>
    );
  }
  if (variant === "error") {
    return (
      <svg
        width="18"
        height="18"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
        className="mt-0.5 shrink-0 text-red-600 dark:text-red-400"
      >
        <path d="M10 1.667A8.333 8.333 0 1 0 18.333 10 8.343 8.343 0 0 0 10 1.667ZM10 5.5a.917.917 0 0 1 .917.917v4a.917.917 0 1 1-1.834 0v-4A.917.917 0 0 1 10 5.5Zm0 9.917a1.083 1.083 0 1 1 0-2.167 1.083 1.083 0 0 1 0 2.167Z" />
      </svg>
    );
  }
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
      className="mt-0.5 shrink-0 text-stone-500 dark:text-stone-400"
    >
      <path d="M10 1.667A8.333 8.333 0 1 0 18.333 10 8.343 8.343 0 0 0 10 1.667ZM10 14.5a.917.917 0 0 1-.917-.917v-4a.917.917 0 1 1 1.834 0v4A.917.917 0 0 1 10 14.5Zm0-9.917a1.083 1.083 0 1 1 0 2.167 1.083 1.083 0 0 1 0-2.167Z" />
    </svg>
  );
}
