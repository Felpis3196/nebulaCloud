"use client";

import { useEffect } from "react";
import { AlertOctagon, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";

interface Props {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function GlobalError({ error, reset }: Props) {
  useEffect(() => {
    if (process.env.NODE_ENV !== "production") {
      console.error(error);
    }
  }, [error]);

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-destructive/10 text-destructive ring-1 ring-destructive/30">
        <AlertOctagon className="h-5 w-5" />
      </div>
      <h1 className="mt-4 text-2xl font-semibold tracking-tight">Something went wrong.</h1>
      <p className="mt-2 max-w-md text-sm text-muted-foreground">
        An unexpected error happened while rendering this view. The error has been logged;
        try again or head back to your dashboard.
      </p>
      {error.digest && (
        <p className="mt-3 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
          digest {error.digest}
        </p>
      )}
      <div className="mt-6">
        <Button variant="gradient" onClick={() => reset()}>
          <RefreshCw />
          Try again
        </Button>
      </div>
    </div>
  );
}
