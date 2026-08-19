"use client";
export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-pat-bg-surface text-pat-text-primary">
      <div className="text-center space-y-4 max-w-md">
        <h1 className="text-2xl font-bold">Something went wrong</h1>
        <p className="text-sm text-pat-text-secondary">{error.message || "An unexpected error occurred."}</p>
        {error.digest && <p className="text-xs text-pat-text-muted">Error ID: {error.digest}</p>}
        <button onClick={reset} className="px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90">
          Try again
        </button>
      </div>
    </div>
  );
}
