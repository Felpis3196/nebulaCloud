import { api, ApiError } from "@/lib/api-client";
import type { Organization } from "@/types/api";
import { useOrganizationStore } from "@/stores/org-store";

function personalOrgSlug(): string {
  const suffix =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID().replace(/-/g, "").slice(0, 8)
      : String(Date.now());
  return `personal-${suffix}`;
}

/** Creates a personal org when the user has none; returns the org list. */
export async function ensurePersonalOrganization(): Promise<Organization[]> {
  let orgs = await api<Organization[]>("/api/v1/organizations");
  if (orgs.length > 0) {
    const selected = useOrganizationStore.getState().selectedOrganizationId;
    const valid = selected !== null && orgs.some((o) => o.id === selected);
    if (!valid) {
      useOrganizationStore.getState().setSelectedOrganizationId(orgs[0]!.id);
    }
    return orgs;
  }

  try {
    const created = await api<Organization>("/api/v1/organizations", {
      method: "POST",
      body: { slug: personalOrgSlug(), name: "Personal" },
    });
    useOrganizationStore.getState().setSelectedOrganizationId(created.id);
    orgs = [created];
    return orgs;
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      orgs = await api<Organization[]>("/api/v1/organizations");
      if (orgs.length > 0) {
        useOrganizationStore.getState().setSelectedOrganizationId(orgs[0]!.id);
      }
      return orgs;
    }
    throw err;
  }
}
