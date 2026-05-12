"use client";

import { create } from "zustand";
import { api } from "@/lib/api-client";
import { clearTokens, persistTokens, readPersistedUser } from "@/lib/auth";
import type { TokenPair, User } from "@/types/api";

interface AuthState {
  user: User | null;
  isAuthenticating: boolean;
  hydrated: boolean;
  hydrate: () => void;
  login: (email: string, password: string) => Promise<User>;
  register: (email: string, password: string, displayName?: string) => Promise<User>;
  fetchMe: () => Promise<User | null>;
  logout: () => Promise<void>;
  setUser: (user: User | null) => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticating: false,
  hydrated: false,

  hydrate: () => {
    if (get().hydrated) return;
    set({ user: readPersistedUser(), hydrated: true });
  },

  setUser: (user) => set({ user }),

  login: async (email, password) => {
    set({ isAuthenticating: true });
    try {
      const pair = await api<TokenPair>("/api/v1/auth/login", {
        method: "POST",
        body: { email, password },
        skipAuth: true,
      });
      persistTokens(pair);
      set({ user: pair.user });
      return pair.user;
    } finally {
      set({ isAuthenticating: false });
    }
  },

  register: async (email, password, displayName) => {
    set({ isAuthenticating: true });
    try {
      await api<User>("/api/v1/auth/register", {
        method: "POST",
        body: { email, password, display_name: displayName },
        skipAuth: true,
      });
      // Auto-login after register.
      const pair = await api<TokenPair>("/api/v1/auth/login", {
        method: "POST",
        body: { email, password },
        skipAuth: true,
      });
      persistTokens(pair);
      set({ user: pair.user });
      return pair.user;
    } finally {
      set({ isAuthenticating: false });
    }
  },

  fetchMe: async () => {
    try {
      const me = await api<User>("/api/v1/me");
      set({ user: me });
      return me;
    } catch {
      set({ user: null });
      clearTokens();
      return null;
    }
  },

  logout: async () => {
    try {
      await api("/api/v1/auth/logout", { method: "POST", body: {} });
    } catch {
      /* best effort */
    } finally {
      clearTokens();
      set({ user: null });
    }
  },
}));
