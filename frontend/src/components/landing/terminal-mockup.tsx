"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { CheckCircle2, Loader2 } from "lucide-react";

interface Line {
  text: string;
  variant?: "prompt" | "info" | "success" | "muted";
  delay: number;
}

const SCRIPT: Line[] = [
  { text: "$ git push origin main", variant: "prompt", delay: 200 },
  { text: "→ webhook received from github.com/acme/payments-api", variant: "muted", delay: 600 },
  { text: "✓ Detected stack: Node.js (paketo buildpack)", variant: "info", delay: 1000 },
  { text: "▸ Building image  layer 4/9  ...........", variant: "info", delay: 1400 },
  { text: "✓ Image pushed: registry/payments-api:8a3c2f0", variant: "success", delay: 1900 },
  { text: "▸ Deploying  rolling 1/3 → healthy 3/3", variant: "info", delay: 2300 },
  { text: "✓ Live at  https://payments-api.nebula.app", variant: "success", delay: 2700 },
];

export function TerminalMockup() {
  const [lines, setLines] = useState<Line[]>([]);
  const [done, setDone] = useState(false);

  useEffect(() => {
    const timeouts = SCRIPT.map((line, i) =>
      setTimeout(() => {
        setLines((prev) => [...prev, line]);
        if (i === SCRIPT.length - 1) {
          setTimeout(() => setDone(true), 400);
        }
      }, line.delay),
    );
    return () => {
      timeouts.forEach(clearTimeout);
    };
  }, []);

  return (
    <div className="relative">
      {/* Glow */}
      <div
        aria-hidden
        className="absolute -inset-8 -z-10 rounded-3xl opacity-50 blur-3xl"
        style={{
          background:
            "radial-gradient(circle at 30% 20%, hsl(239 84% 60% / 0.5), transparent 60%), radial-gradient(circle at 80% 80%, hsl(305 80% 60% / 0.45), transparent 55%)",
        }}
      />

      <div className="overflow-hidden rounded-xl border border-border/60 bg-card/80 shadow-2xl backdrop-blur-xl">
        {/* Title bar */}
        <div className="flex items-center gap-1.5 border-b border-border/60 bg-card/60 px-4 py-3">
          <span className="h-2.5 w-2.5 rounded-full bg-rose-500/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-400/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-400/70" />
          <div className="ml-3 flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-mono">~/projects/payments-api</span>
            <span className="text-muted-foreground/60">— main</span>
          </div>
          {!done ? (
            <div className="ml-auto flex items-center gap-1.5 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              deploying
            </div>
          ) : (
            <div className="ml-auto flex items-center gap-1.5 text-xs text-success">
              <CheckCircle2 className="h-3 w-3" />
              live
            </div>
          )}
        </div>

        {/* Body */}
        <div className="space-y-1.5 px-5 py-5 font-mono text-xs leading-relaxed">
          {lines.map((line, i) => (
            <motion.div
              key={i}
              initial={{ opacity: 0, x: -6 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.25 }}
              className={
                line.variant === "prompt"
                  ? "text-foreground"
                  : line.variant === "success"
                    ? "text-success"
                    : line.variant === "info"
                      ? "text-info"
                      : "text-muted-foreground"
              }
            >
              {line.text}
            </motion.div>
          ))}
          {!done && (
            <motion.span
              animate={{ opacity: [1, 0.2, 1] }}
              transition={{ duration: 1.1, repeat: Infinity }}
              className="inline-block h-3.5 w-2 translate-y-0.5 bg-foreground/70"
              aria-hidden
            />
          )}
        </div>
      </div>
    </div>
  );
}
