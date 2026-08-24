"use client";
import Link from "next/link";
import { Fragment } from "react";
import { usePathname } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { useState } from "react";
import { IconMenu2, IconX } from "@tabler/icons-react";
import { adminNavigation } from "@/config/navigation/admin-navigation";
import { userNavigation } from "@/config/navigation/user-navigation";
import { isAdminRole, panelLabelForRole } from "@/lib/roles";
import type { NavItem } from "@/config/navigation/admin-navigation";

export default function Sidebar() {
  const { user, sessionState } = useAuth();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);

  const isAdminRoute = pathname.startsWith("/admin");
  const isUserRoute = pathname.startsWith("/dashboard");

  let items: NavItem[];
  let panelLabel: string;

  if (sessionState === 'LOADING') {
    items = [];
    panelLabel = 'Loading…';
  } else if (isAdminRoute && isAdminRole(user?.role)) {
    items = adminNavigation;
    panelLabel = '';
  } else if (isUserRoute && !isAdminRole(user?.role)) {
    items = userNavigation;
    panelLabel = '';
  } else if (isAdminRoute && !isAdminRole(user?.role)) {
    items = [];
    panelLabel = 'Access Denied';
  } else if (isUserRoute && isAdminRole(user?.role)) {
    items = adminNavigation;
    panelLabel = '';
  } else {
    items = isAdminRole(user?.role) ? adminNavigation : userNavigation;
    panelLabel = '';
  }

  return (
    <>
      <button
        onClick={() => setOpen(!open)}
        className="lg:hidden fixed top-4 left-4 z-50 p-2 bg-pat-bg-sidebar rounded-md border border-pat-border-sidebar text-pat-text-sidebar"
        aria-label="Toggle navigation"
      >
        {open ? <IconX size={20} /> : <IconMenu2 size={20} />}
      </button>
      {open && <div className="fixed inset-0 bg-black/50 z-30 lg:hidden" onClick={() => setOpen(false)} />}
      <aside
        className={"fixed lg:sticky top-0 left-0 z-40 h-screen w-64 bg-pat-bg-sidebar border-r border-pat-border-sidebar flex flex-col transition-transform " + (open ? "translate-x-0" : "-translate-x-full lg:translate-x-0")}
      >
        <div className="p-4 border-b border-pat-border-sidebar">
          <Link href="/" className="flex items-center gap-2">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/predict-a-trade_icon-only_white.svg" alt="logo" className="h-8 w-8" />
            <span className="font-semibold text-sm text-pat-text-sidebar-active">Predict-A-Trade</span>
          </Link>
        </div>
        <nav className="flex-1 overflow-y-auto p-3 space-y-1" aria-label="Main navigation">
          {items.map((item, idx) => {
            const active = pathname === item.href || pathname.startsWith(item.href + "/");
            const showSection = !!item.section && (idx === 0 || items[idx - 1].section !== item.section);
            return (
              <Fragment key={item.href}>
                {showSection && (
                  <div className="px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-wider text-pat-text-sidebar-muted">{item.section}</div>
                )}
                <Link
                  href={item.href}
                  onClick={() => setOpen(false)}
                  className={"flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors " + (active ? "bg-pat-bg-sidebar-active text-pat-text-sidebar-active" : "text-pat-text-sidebar hover:bg-pat-bg-sidebar-hover hover:text-pat-text-sidebar-active")}
                >
                  <item.icon size={18} />
                  {item.label}
                </Link>
              </Fragment>
            );
          })}
          {items.length === 0 && sessionState === 'LOADING' && (
            <div className="px-3 py-2 text-xs text-pat-text-sidebar-muted">Loading session…</div>
          )}
        </nav>
        <div className="p-3 border-t border-pat-border-sidebar">
          <div className="text-xs text-pat-text-sidebar-muted" data-testid="panel-label">
            {panelLabel}
          </div>
        </div>
      </aside>
    </>
  );
}
