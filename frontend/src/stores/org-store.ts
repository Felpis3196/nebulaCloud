"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

interface OrgStore {
  selectedOrganizationId: string | null;
  setSelectedOrganizationId: (id: string | null) => void;
}

export const useOrganizationStore = create<OrgStore>()(
  persist(
    (set) => ({
      selectedOrganizationId: null,
      setSelectedOrganizationId: (id) => set({ selectedOrganizationId: id }),
    }),
    { name: "nebula_org" },
  ),
);
