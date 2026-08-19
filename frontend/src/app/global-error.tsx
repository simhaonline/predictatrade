"use client";
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="en">
      <body style={{ margin: 0, fontFamily: "system-ui, sans-serif", background: "#0f172a", color: "#f8fafc" }}>
        <div style={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center" }}>
          <div style={{ textAlign: "center", maxWidth: "400px" }}>
            <h1 style={{ fontSize: "1.5rem", fontWeight: 700, marginBottom: "0.5rem" }}>Application Error</h1>
            <p style={{ fontSize: "0.875rem", color: "#94a3b8", marginBottom: "1rem" }}>{error.message || "A critical error occurred."}</p>
            {error.digest && <p style={{ fontSize: "0.75rem", color: "#64748b" }}>Error ID: {error.digest}</p>}
            <button onClick={reset} style={{ marginTop: "1rem", padding: "0.5rem 1rem", background: "#3b82f6", color: "#ffffff", border: "none", borderRadius: "0.375rem", fontSize: "0.875rem", cursor: "pointer" }}>
              Try again
            </button>
          </div>
        </div>
      </body>
    </html>
  );
}
