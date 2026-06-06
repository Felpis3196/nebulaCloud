"use client";

import { useSyncSelectedOrganization } from "@/hooks/use-sync-selected-organization";

/** Keeps persisted organization id aligned with the API on every dashboard page. */
export function OrgSyncBootstrap() {
  useSyncSelectedOrganization();
  return null;
}
