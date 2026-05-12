"use client";

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Cloud } from "lucide-react";

const QUOTES: { quote: string; author: string }[] = [
  {
    quote: "Push to main, ship in seconds. NebulaCloud feels like git push to production.",
    author: "Internal SRE team",
  },
  {
    quote: "Container builds, custom domains, automatic TLS — all on infra we control.",
    author: "Platform engineering",
  },
  {
    quote: "Realtime logs and metrics next to each deploy. No more SSHing into boxes.",
    author: "DevOps lead",
  },
];

export function AuthArtPanel() {
  const [idx, setIdx] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setIdx((i) => (i + 1) % QUOTES.length), 6000);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="relative hidden h-full overflow-hidden lg:flex">
      {/* Background gradient */}
      <div
        className="absolute inset-0 animate-gradient-shift opacity-90"
        style={{
          backgroundImage:
            "linear-gradient(120deg, hsl(239 84% 24%), hsl(265 85% 30%), hsl(305 80% 32%), hsl(239 84% 24%))",
          backgroundSize: "200% 200%",
        }}
      />
      <div
        className="absolute inset-0 opacity-30 mix-blend-overlay"
        style={{
          backgroundImage:
            "radial-gradient(circle at 20% 20%, hsl(290 90% 60% / 0.4), transparent 40%), radial-gradient(circle at 80% 80%, hsl(217 91% 60% / 0.4), transparent 40%)",
        }}
      />
      <div className="absolute inset-0 bg-[linear-gradient(to_bottom,transparent,hsl(240_10%_4%/0.8))]" />
      <div className="absolute inset-0 bg-grid-faint bg-grid opacity-[0.06]" />

      <div className="relative z-10 flex h-full w-full flex-col justify-between p-12 text-white">
        <div className="flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 ring-1 ring-white/15 backdrop-blur">
            <Cloud className="h-5 w-5" />
          </div>
          <span className="text-base font-semibold tracking-tight">NebulaCloud</span>
        </div>

        <div className="space-y-6">
          <AnimatePresence mode="wait">
            <motion.blockquote
              key={idx}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.4 }}
              className="space-y-3"
            >
              <p className="text-2xl font-medium leading-tight tracking-tight text-white/95">
                "{QUOTES[idx]?.quote}"
              </p>
              <footer className="text-sm text-white/60">— {QUOTES[idx]?.author}</footer>
            </motion.blockquote>
          </AnimatePresence>

          <div className="flex gap-1.5">
            {QUOTES.map((_, i) => (
              <div
                key={i}
                className={`h-1 rounded-full transition-all ${
                  i === idx ? "w-6 bg-white/80" : "w-3 bg-white/25"
                }`}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
