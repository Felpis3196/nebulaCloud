import { env } from "@/lib/env";
import {
  clearTokens,
  persistTokens,
  readAccessToken,
  readRefreshToken,
} from "@/lib/auth";
import type { ApiEnvelope, ApiErrorBody, TokenPair } from "@/types/api";

/** Custom error thrown by the API client. */
export class ApiError extends Error {
  readonly status: number;
  readonly kind: ApiErrorBody["error"]["kind"];
  readonly code?: string;
  readonly details?: Record<string, string>;

  constructor(message: string, status: number, body?: ApiErrorBody) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.kind = body?.error.kind ?? "internal";
    this.code = body?.error.code;
    this.details = body?.error.details;
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  skipAuth?: boolean;
  /** Send cookies (e.g. GitHub OAuth token cookie on the API host). */
  withCredentials?: boolean;
  /** Internal: when set, suppresses the auto-refresh-and-retry flow. */
  _retried?: boolean;
}

/** Single in-flight refresh promise so concurrent 401s share work. */
let inflightRefresh: Promise<TokenPair> | null = null;

async function refreshSession(): Promise<TokenPair> {
  if (inflightRefresh) return inflightRefresh;
  const refreshToken = readRefreshToken();
  if (!refreshToken) {
    throw new ApiError("not authenticated", 401);
  }

  inflightRefresh = (async () => {
    try {
      const res = await fetch(`${env.NEXT_PUBLIC_API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (!res.ok) {
        const body = (await res.json().catch(() => undefined)) as ApiErrorBody | undefined;
        throw new ApiError("refresh failed", res.status, body);
      }
      const envelope = (await res.json()) as ApiEnvelope<TokenPair>;
      persistTokens(envelope.data);
      return envelope.data;
    } finally {
      inflightRefresh = null;
    }
  })();

  return inflightRefresh;
}

async function rawFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, skipAuth, withCredentials, headers, _retried, ...rest } = options;

  const init: RequestInit = {
    ...rest,
    credentials: withCredentials ? "include" : rest.credentials,
    headers: {
      Accept: "application/json",
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      ...(headers ?? {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  };

  if (!skipAuth) {
    const access = readAccessToken();
    if (access) {
      (init.headers as Record<string, string>).Authorization = `Bearer ${access}`;
    }
  }

  const url = path.startsWith("http") ? path : `${env.NEXT_PUBLIC_API_URL}${path}`;
  const res = await fetch(url, init);

  if (res.status === 204) return undefined as T;

  // Auto-refresh on 401, but only once per request.
  if (res.status === 401 && !skipAuth && !_retried) {
    try {
      await refreshSession();
      return rawFetch<T>(path, { ...options, _retried: true });
    } catch {
      clearTokens();
      // Bubble the original 401 — caller (auth store) handles redirect.
      const body = (await res.json().catch(() => undefined)) as ApiErrorBody | undefined;
      throw new ApiError("unauthenticated", 401, body);
    }
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => undefined)) as ApiErrorBody | undefined;
    throw new ApiError(body?.error.message ?? `HTTP ${res.status}`, res.status, body);
  }

  return (await res.json()) as T;
}

/**
 * Fetch a JSON envelope and unwrap `data`. Throws ApiError on failure.
 */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const body = await rawFetch<ApiEnvelope<T> | T>(path, options);
  if (body && typeof body === "object" && "data" in (body as object)) {
    return (body as ApiEnvelope<T>).data;
  }
  return body as T;
}

/** Same as api() but returns the full envelope (used when meta matters). */
export async function apiEnvelope<T>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiEnvelope<T>> {
  return rawFetch<ApiEnvelope<T>>(path, options);
}

/** API call that includes cookies (GitHub OAuth session on the API origin). */
export async function apiWithCredentials<T>(path: string): Promise<T> {
  return api<T>(path, { skipAuth: true, withCredentials: true });
}
