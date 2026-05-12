import type { Metadata, Viewport } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";

import { Providers } from "@/components/providers";
import { Toaster } from "@/components/ui/sonner";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "NebulaCloud — Deploy from Git in seconds",
    template: "%s · NebulaCloud",
  },
  description:
    "Self-hosted PaaS for deploying applications straight from Git. Realtime logs, metrics, custom domains, and a polished dashboard.",
  applicationName: "NebulaCloud",
  authors: [{ name: "NebulaCloud" }],
  openGraph: {
    title: "NebulaCloud",
    description:
      "Self-hosted PaaS for deploying applications straight from Git.",
    type: "website",
  },
};

export const viewport: Viewport = {
  themeColor: "#09090b",
  colorScheme: "dark",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${GeistSans.variable} ${GeistMono.variable}`}
    >
      <body className="min-h-screen bg-background font-sans">
        <Providers>{children}</Providers>
        <Toaster position="bottom-right" />
      </body>
    </html>
  );
}
