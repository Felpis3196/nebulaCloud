import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * cn merges Tailwind class names, resolving conflicts via tailwind-merge.
 * Used by every shadcn primitive and most components in the app.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format a Date | string | number into a "5m ago"-style relative label. */
export function relativeTime(value: Date | string | number): string {
  const d = value instanceof Date ? value : new Date(value);
  const diff = Date.now() - d.getTime();
  const sec = Math.round(diff / 1000);
  const min = Math.round(sec / 60);
  const hr = Math.round(min / 60);
  const day = Math.round(hr / 24);
  if (Number.isNaN(diff)) return "—";
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  if (min < 60) return `${min}m ago`;
  if (hr < 24) return `${hr}h ago`;
  if (day < 7) return `${day}d ago`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/** Pretty-print a millisecond duration: 850ms, 4.2s, 1m 12s. */
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(s < 10 ? 1 : 0)}s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s - m * 60);
  return `${m}m ${rs}s`;
}

/** Pretty-print a byte count: 12B, 4.3KB, 1.2MB. */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let idx = 0;
  let value = bytes;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx += 1;
  }
  return `${value.toFixed(value < 10 && idx > 0 ? 1 : 0)}${units[idx]}`;
}

/** Truncate a commit SHA to its first 7 chars, safe for empty input. */
export function shortSha(sha: string | null | undefined, len = 7): string {
  if (!sha) return "—";
  return sha.length <= len ? sha : sha.slice(0, len);
}

/** Pluralise a word naively. */
export function pluralize(count: number, singular: string, plural?: string): string {
  return count === 1 ? singular : (plural ?? `${singular}s`);
}
