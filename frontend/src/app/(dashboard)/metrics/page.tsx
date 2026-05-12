import { MetricsPanel } from "@/components/metrics/metrics-panel";
import { MOCK_SERVICES } from "@/lib/mock-data";

export const metadata = { title: "Metrics" };

export default function MetricsPage() {
  const services = MOCK_SERVICES.map((s) => ({ id: s.id, name: s.name }));
  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Metrics</h1>
        <p className="text-sm text-muted-foreground">
          Container CPU, memory, network and request rate. Backed by Prometheus.
        </p>
      </header>
      <MetricsPanel services={services} />
    </div>
  );
}
