import { DomainsTable } from "@/components/domains/domains-table";
import { MOCK_DOMAINS, MOCK_SERVICES } from "@/lib/mock-data";

export default async function ProjectDomainsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const serviceIds = new Set(MOCK_SERVICES.filter((s) => s.project_id === id).map((s) => s.id));
  const domains = MOCK_DOMAINS.filter((d) => serviceIds.has(d.service_id));
  return <DomainsTable domains={domains} />;
}
