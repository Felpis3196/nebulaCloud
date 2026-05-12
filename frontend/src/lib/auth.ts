import Cookies from "js-cookie";
import type { TokenPair, User } from "@/types/api";

/**
 * Cookie-backed token store.
 *
 * Cookies (not localStorage) so Next.js middleware can gate routes on the
 * server side. Both cookies are set with `Secure` (in HTTPS) and
 * `SameSite=lax`. `httpOnly` is intentionally off here because the client
 * needs to attach the access token to requests; Phase 9 migrates to a
 * Route Handler proxy + httpOnly refresh.
 */
export const COOKIE_ACCESS = "nebula_access";
export const COOKIE_REFRESH = "nebula_refresh";
export const COOKIE_USER = "nebula_user";

/** Persist a token pair returned by the API. */
export function persistTokens(pair: TokenPair) {
  const isHttps = typeof window !== "undefined" && window.location.protocol === "https:";
  const accessExpires = new Date(pair.access_expiry);
  const refreshExpires = new Date(pair.refresh_expiry);

  Cookies.set(COOKIE_ACCESS, pair.access_token, {
    expires: accessExpires,
    sameSite: "lax",
    secure: isHttps,
    path: "/",
  });
  Cookies.set(COOKIE_REFRESH, pair.refresh_token, {
    expires: refreshExpires,
    sameSite: "lax",
    secure: isHttps,
    path: "/",
  });
  Cookies.set(COOKIE_USER, encodeURIComponent(JSON.stringify(pair.user)), {
    expires: refreshExpires,
    sameSite: "lax",
    secure: isHttps,
    path: "/",
  });
}

/** Forget every cookie this app owns. */
export function clearTokens() {
  Cookies.remove(COOKIE_ACCESS, { path: "/" });
  Cookies.remove(COOKIE_REFRESH, { path: "/" });
  Cookies.remove(COOKIE_USER, { path: "/" });
}

/** Read the access token client-side (returns "" when absent). */
export function readAccessToken(): string {
  return Cookies.get(COOKIE_ACCESS) ?? "";
}

export function readRefreshToken(): string {
  return Cookies.get(COOKIE_REFRESH) ?? "";
}

export function readPersistedUser(): User | null {
  const raw = Cookies.get(COOKIE_USER);
  if (!raw) return null;
  try {
    return JSON.parse(decodeURIComponent(raw)) as User;
  } catch {
    return null;
  }
}
