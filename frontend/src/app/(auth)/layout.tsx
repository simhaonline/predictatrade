import Footer from "@/components/layout/footer";
import ThemeControl from "@/components/auth/theme-control";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="h-dvh flex flex-col bg-pat-bg-page overflow-hidden relative">
      {/* Display Preferences — top-right, shared with dashboard theme system */}
      <div className="absolute top-3 right-3 z-10">
        <ThemeControl />
      </div>

      <div className="flex-1 flex items-center justify-center overflow-y-auto"
           style={{ padding: "clamp(0.75rem, 3vh, 1.5rem)" }}>
        <div className="w-full max-w-sm" style={{ maxHeight: "100%" }}>
          {children}
        </div>
      </div>
      <Footer />
    </div>
  );
}
