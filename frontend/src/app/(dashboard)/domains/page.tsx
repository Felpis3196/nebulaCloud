import { DomainsTable } from "@/components/domains/domains-table";
import { MOCK_DOMAINS } from "@/lib/mock-data";

export const metadata = { title: "Domains" };

export default function DomainsPage() {
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Domains</h1>
        <p className="text-sm text-muted-foreground">
          Bring your own hostname. NebulaCloud handles TLS via Let's Encrypt.
        </p>
      </header>
      <DomainsTable domains={MOCK_DOMAINS} />
    </div>
  );
}
