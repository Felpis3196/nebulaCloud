"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";

/**
 * Hydrates the auth store from cookies on mount, and exposes the
 * commonly-used auth surface as a single hook.
 */
export function useAuth() {
  const state = useAuthStore();
  useEffect(() => {
    state.hydrate();
  }, [state]);
  return state;
}
