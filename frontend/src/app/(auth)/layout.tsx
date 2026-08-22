import Footer from "@/components/layout/footer";
import ThemeControl from "@/components/auth/theme-control";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-dvh flex overflow-hidden bg-pat-bg-page">
      {/* Visual Panel — left side (hidden on mobile) */}
      <aside className="hidden lg:flex flex-col justify-between p-8 xl:p-12 relative overflow-hidden"
             style={{ flex: "1.03fr", minWidth: "400px", background: "#0c0e12" }}>
        {/* Dark gradient overlay */}
        <div className="absolute inset-0" style={{
          background: "linear-gradient(180deg, rgba(8,10,14,0.48) 0%, rgba(8,10,14,0.16) 36%, rgba(8,10,14,0.74) 100%)"
        }} />
        {/* Grid pattern overlay */}
        <div className="absolute inset-0 opacity-20 pointer-events-none" style={{
          backgroundImage: "linear-gradient(rgba(255,255,255,0.09) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.09) 1px, transparent 1px)",
          backgroundSize: "84px 84px",
          maskImage: "linear-gradient(to bottom, transparent, #000 28%, #000 78%, transparent)"
        }} />

        {/* Header */}
      <div className="relative z-10 flex items-center justify-between">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/predict-a-trade_horizontal_white.svg" alt="Predict-A-Trade" style={{ width: "clamp(165px, 13vw, 204px)", height: "auto" }} />
          <span className="text-white/70 font-bold uppercase tracking-widest" style={{ fontSize: "9px" }}>
            <span className="inline-block w-2 h-2 rounded-full bg-blue-400 mr-2" style={{ boxShadow: "0 0 0 5px rgba(142,178,255,0.13)" }} />
            Market intelligence
          </span>
        </div>

        {/* Content */}
      <div className="relative z-10" style={{ maxWidth: "630px", width: "90%" }}>
          <p className="text-blue-300 font-bold uppercase tracking-widest mb-5" style={{ fontSize: "10px" }}>
            Independent perspective · Established 2016
          </p>
          <h2 className="font-serif text-white" style={{
            fontSize: "clamp(50px, 5.2vw, 80px)",
            lineHeight: "0.94",
            fontWeight: 600,
            letterSpacing: "-0.052em"
          }}>
            See the signal <em className="text-blue-300 italic font-medium">before</em> the noise.
          </h2>
          <p className="text-white/68 mt-6" style={{ maxWidth: "490px", fontSize: "14px", lineHeight: "1.75" }}>
            Read market movement with more clarity, context and conviction. Every signal is designed to be inspected — not simply followed.
          </p>
        </div>

        {/* Footer */}
      <div className="relative z-10 flex items-center gap-4 text-white/54 font-bold uppercase tracking-widest" style={{ fontSize: "9px" }}>
          <span>Signal study 01</span>
          <span className="flex-1 h-px bg-white/20" />
          <span>Predict-A-Trade / 2026</span>
        </div>
      </aside>

      {/* Form Panel — right side */}
      <div className="flex flex-col relative overflow-hidden"
           style={{ flex: "0.97fr", minWidth: "0" }}>
        {/* Radial gradient background */}
        <div className="absolute inset-0 pointer-events-none opacity-30" style={{
          background: "radial-gradient(circle at 92% 5%, rgba(32,95,220,0.08), transparent 26%)"
        }} />

        {/* Theme control */}
      <div className="absolute top-3 right-3 z-10">
          <ThemeControl />
        </div>

        {/* Top bar */}
      <div className="flex items-center justify-end px-6 xl:px-12 pt-6 relative z-10">
          <a href="https://predictatrade.com" className="flex items-center gap-2 text-pat-text-muted hover:text-pat-primary transition-colors font-bold uppercase tracking-widest" style={{ fontSize: "10px" }}>
            <span>Back to website</span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="w-4 h-4"><path d="M7 17 17 7M8 7h9v9"/></svg>
          </a>
        </div>

        {/* Form content */}
      <div className="flex-1 flex items-center justify-center overflow-y-auto px-6 xl:px-12 relative z-10">
          <div className="w-full" style={{ maxWidth: "480px" }}>
            {children}
          </div>
        </div>

        {/* Footer */}
      <div className="relative z-10">
          <Footer />
        </div>
      </div>
    </div>
  );
}
