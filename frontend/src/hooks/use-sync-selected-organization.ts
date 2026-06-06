"use client";

import { useEffect, useRef } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { useOrganizations } from "@/hooks/use-organizations";
import { useOrganizationStore } from "@/stores/org-store";

/**
 * Keeps persisted nebula_org in sync with GET /organizations.
 * Clears stale IDs after DB resets or user switches.
 */
export function useSyncSelectedOrganization() {
  const t = useTranslations("dashboard.projects");
  const { data: orgs = [], isLoading, isFetched } = useOrganizations();
  const selected = useOrganizationStore((s) => s.selectedOrganizationId);
  const setOrg = useOrganizationStore((s) => s.setSelectedOrganizationId);
  const warnedRef = useRef(false);

  useEffect(() => {
    if (!isFetched || isLoading) return;

    if (orgs.length === 0) {
      if (selected !== null) setOrg(null);
      warnedRef.current = false;
      return;
    }

    const valid = selected !== null && orgs.some((o) => o.id === selected);
    if (valid) {
      warnedRef.current = false;
      return;
    }

    setOrg(orgs[0]!.id);
    if (selected !== null && !warnedRef.current) {
      warnedRef.current = true;
      toast.message(t("orgResynced"));
    }
  }, [orgs, selected, setOrg, isLoading, isFetched, t]);
}
