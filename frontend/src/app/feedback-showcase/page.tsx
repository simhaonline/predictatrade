import type { Metadata } from "next";
import Link from "next/link";
import { IconStarFilled, IconQuote } from "@tabler/icons-react";

export const metadata: Metadata = {
  title: "What our clients say — Predict-A-Trade",
  description: "Featured feedback from Predict-A-Trade clients about the XAUUSD market analysis platform.",
};

// revalidate every 10 minutes — moderation changes propagate without a rebuild
export const revalidate = 600;

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "https://api.predictatrade.com/api/v1";

interface FeaturedItem {
  first_name: string;
  category: string;
  rating: number;
  message: string;
  featured_at: string;
}

const CATEGORY_LABEL: Record<string, string> = {
  bug: "Bug report",
  feature_request: "Feature request",
  usability: "Usability",
  performance: "Performance",
  billing: "Billing",
  other: "General",
};

async function getFeatured(): Promise<FeaturedItem[]> {
  try {
    const res = await fetch(`${API_BASE}/feedback/public/featured?limit=12`, {
      next: { revalidate: 600 },
    });
    if (!res.ok) return [];
    const data = (await res.json()) as { items: FeaturedItem[] };
    return data.items ?? [];
  } catch {
    return [];
  }
}

export default async function FeedbackShowcasePage() {
  const items = await getFeatured();

  return (
    <main className="min-h-screen bg-pat-bg-page text-pat-text-primary">
      <div className="max-w-4xl mx-auto px-6 py-14">
        <header className="text-center mb-10">
          <h1 className="text-2xl md:text-3xl font-bold">What our clients say</h1>
          <p className="text-sm md:text-base text-pat-text-secondary mt-2">
            Featured feedback from Predict-A-Trade clients — shared with permission, first names only.
          </p>
        </header>

        {items.length === 0 ? (
          <div className="rounded-lg border border-pat-border bg-pat-card p-10 text-center text-pat-text-secondary">
            Featured client feedback will appear here soon.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {items.map((item, i) => (
              <figure
                key={i}
                className="rounded-lg border border-pat-border bg-pat-card p-5 flex flex-col"
              >
                <div className="flex items-center justify-between mb-3">
                  <span className="text-[10px] px-2 py-0.5 rounded-full border border-pat-info/30 bg-pat-info/10 text-pat-info font-medium">
                    {CATEGORY_LABEL[item.category] ?? item.category}
                  </span>
                  <span className="flex items-center gap-0.5 text-pat-warning" aria-label={`${item.rating} out of 5`}>
                    {[1, 2, 3, 4, 5].map((n) => (
                      <IconStarFilled
                        key={n}
                        size={13}
                        className={n <= item.rating ? "text-pat-warning" : "text-pat-border"}
                      />
                    ))}
                  </span>
                </div>
                <IconQuote size={20} className="text-pat-text-muted mb-2" />
                <blockquote className="text-sm text-pat-text-secondary leading-relaxed flex-1">
                  {item.message}
                </blockquote>
                <figcaption className="mt-4 text-xs text-pat-text-muted">
                  — {item.first_name} · {new Date(item.featured_at).toLocaleDateString("en-GB", { month: "short", year: "numeric" })}
                </figcaption>
              </figure>
            ))}
          </div>
        )}

        <footer className="mt-12 text-center space-y-3">
          <Link
            href="/register"
            className="inline-block text-sm px-6 py-2.5 rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover transition-colors"
          >
            Create your account
          </Link>
          <p className="text-[11px] text-pat-text-muted">
            © {new Date().getUTCFullYear()} Predict-A-Trade. All rights reserved. Market analysis platform — nothing on
            this page is financial advice.
          </p>
        </footer>
      </div>
    </main>
  );
}