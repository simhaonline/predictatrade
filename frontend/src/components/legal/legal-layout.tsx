"use client";
import Footer from "@/components/layout/footer";

export default function LegalLayout({
  children,
  title,
  lastUpdated,
}: {
  children: React.ReactNode;
  title: string;
  lastUpdated: string;
}) {
  return (
    <div className="min-h-screen bg-pat-bg-page flex flex-col">
      <div className="flex-1 max-w-3xl mx-auto w-full px-4 py-8">
        <div className="bg-pat-bg-surface border border-pat-card-border rounded-lg p-8 shadow-sm">
          <h1 className="text-2xl font-bold text-pat-text-primary mb-2">{title}</h1>
          <p className="text-xs text-pat-text-muted mb-6">Last updated: {lastUpdated}</p>
          <div className="prose prose-sm max-w-none space-y-4 text-pat-text-secondary
            [&_h2]:text-pat-text-primary [&_h2]:text-lg [&_h2]:font-semibold [&_h2]:mt-6
            [&_h3]:text-pat-text-primary [&_h3]:font-medium [&_h3]:mt-4
            [&_p]:leading-relaxed [&_li]:text-pat-text-secondary
            [&_a]:text-pat-primary [&_a:hover]:underline
            [&_strong]:text-pat-text-primary
            [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:space-y-1">
            {children}
          </div>
        </div>
      </div>
      <Footer />
    </div>
  );
}
