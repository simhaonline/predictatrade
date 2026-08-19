import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-pat-bg-surface text-pat-text-primary">
      <div className="text-center space-y-4">
        <h1 className="text-6xl font-bold text-neutral-800">404</h1>
        <p className="text-sm text-pat-text-secondary">The page you are looking for does not exist.</p>
        <Link href="/" className="inline-block px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90">
          Go Home
        </Link>
      </div>
    </div>
  );
}
