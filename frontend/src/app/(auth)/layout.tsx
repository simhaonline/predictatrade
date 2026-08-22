import Footer from "@/components/layout/footer";
import ThemeControl from "@/components/auth/theme-control";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column", background: "#f7f6f2", overflow: "hidden" }}>
      {/* Theme control — top right */}
      <div style={{ position: "absolute", top: "12px", right: "12px", zIndex: 50 }}>
        <ThemeControl />
      </div>

      {/* Main content — centered form */}
      <div style={{ flex: 1, display: "flex", alignItems: "center", justifyContent: "center", padding: "20px", overflowY: "auto" }}>
        <div style={{ width: "100%", maxWidth: "440px" }}>
          {/* Brand logo at top */}
          <div style={{ textAlign: "center", marginBottom: "24px" }}>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/predict-a-trade_horizontal.svg" alt="Predict-A-Trade" style={{ width: "200px", height: "auto", margin: "0 auto" }} />
          </div>
          {children}
        </div>
      </div>

      {/* Footer */}
      <div style={{ flexShrink: 0 }}>
        <Footer />
      </div>
    </div>
  );
}
