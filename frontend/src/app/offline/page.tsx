import Link from "next/link";

export const metadata = { title: "Offline — Predict-A-Trade" };

export default function OfflinePage() {
  return (
    <main className="min-h-screen flex items-center justify-center bg-pat-bg-base px-4">
      <div className="text-center space-y-4 max-w-md">
        <h1 className="text-2xl font-bold text-pat-text-primary">You are offline</h1>
        <p className="text-sm text-pat-text-secondary">
          Live market data and signals require an active connection. Trading truth is never
          served from cache, so this screen is shown instead of stale data.
        </p>
        <Link
          href="/"
          className="inline-block rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground"
        >
          Try again
        </Link>
      </div>
    </main>
  );
}
