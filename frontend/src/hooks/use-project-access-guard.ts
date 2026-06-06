"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { ApiError } from "@/lib/api-client";

/** Handles project fetch errors without sending users to the generic 404 page. */
export function useProjectAccessGuard(
  projectId: string,
  isError: boolean,
  error: unknown,
  isFetched: boolean,
) {
  const router = useRouter();
  const t = useTranslations("dashboard.projects");

  useEffect(() => {
    if (!isFetched || !isError || !(error instanceof ApiError)) return;

    const msg = error.message.toLowerCase();
    if (msg.includes("membership")) {
      toast.error(t("membershipError"));
      router.replace("/projects");
      return;
    }
    if (error.status === 404) {
      toast.error(t("projectNotFound"));
      router.replace("/projects");
    }
  }, [isError, error, isFetched, projectId, router, t]);
}
