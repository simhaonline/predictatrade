"use client";
import Sidebar from "./sidebar";
import Topbar from "./topbar";
import Footer from "./footer";
import { useAuth } from "@/providers/auth-provider";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { isAdminRole } from "@/lib/roles";

export default function AppShell({ children }: { children: React.ReactNode }) {
  const { sessionState, user } = useAuth();
  const pathname = usePathname();

  useEffect(() => {
    if (sessionState !== 'AUTHENTICATED' || !user) return;
    const isAdmin = isAdminRole(user.role);
    const onAdminRoute = pathname.startsWith('/admin');
    const onUserRoute = pathname.startsWith('/dashboard');
    if (isAdmin && onUserRoute) {
      // Hard navigation to avoid RSC 404 errors
      window.location.href = '/admin/dashboard';
    } else if (!isAdmin && onAdminRoute) {
      window.location.href = '/dashboard/live';
    }
  }, [sessionState, user, pathname]);

  if (sessionState === 'LOADING') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-pat-bg-page">
        <div className="text-center space-y-3">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-2 border-pat-border-strong border-t-pat-primary" />
          <div className="text-sm text-pat-text-muted">Loading session…</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-pat-bg-page">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0">
        <Topbar />
        <main className="flex-1 p-4 overflow-auto">
          {children}
        </main>
        <Footer />
      </div>
    </div>
  );
}
