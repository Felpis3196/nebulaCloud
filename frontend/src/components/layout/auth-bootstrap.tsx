"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";

/**
 * Mounted at the top of every protected layout. Does two things:
 *
 *  1. hydrates the auth store from cookies on first render
 *  2. lazily verifies the session via /me; on failure, kicks back to /login
 */
export function AuthBootstrap() {
  const router = useRouter();
  const hydrate = useAuthStore((s) => s.hydrate);
  const fetchMe = useAuthStore((s) => s.fetchMe);
  const user = useAuthStore((s) => s.user);

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    void fetchMe().then((me) => {
      if (!cancelled && !me) router.replace("/login");
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return null;
}
