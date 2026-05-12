import { env } from "@/lib/env";
import { readAccessToken } from "@/lib/auth";

/**
 * WebSocket factory for streaming endpoints (Phase 6+).
 *
 * The backend has not implemented streaming yet, so this is wired but not
 * called from any production path. Log viewer "Live" toggle synthesises
 * lines locally; once the API ships `/api/v1/services/:id/logs/stream`,
 * `connectStream("/api/v1/services/abc/logs/stream")` will replace it.
 */
export function connectStream(path: string): WebSocket {
  const base = env.NEXT_PUBLIC_API_URL.replace(/^http/, "ws");
  const token = readAccessToken();
  const url = new URL(`${base}${path}`);
  if (token) url.searchParams.set("token", token);
  return new WebSocket(url.toString());
}
