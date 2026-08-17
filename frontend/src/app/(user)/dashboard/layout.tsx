import { AppShell } from '@/components/ui/app-shell';

const userNavItems = [
  { section: 'TRADING', items: [
    { id: 'overview', label: 'Overview', icon: '📊', route: '/dashboard' },
    { id: 'chart', label: 'Live Chart', icon: '📈', route: '/dashboard/chart' },
    { id: 'signals', label: 'Signals', icon: '⚡', route: '/dashboard/signals' },
    { id: 'positions', label: 'Positions & Trades', icon: '🎯', route: '/dashboard/positions' },
    { id: 'performance', label: 'Performance', icon: '📉', route: '/dashboard/performance' },
    { id: 'market-pulse', label: 'Market Pulse', icon: '🌍', route: '/dashboard/market-pulse' },
  ]},
  { section: 'ACCOUNT', items: [
    { id: 'mt-setup', label: 'MT4 / MT5', icon: '🖥️', route: '/dashboard/mt-setup' },
    { id: 'subscription', label: 'Subscription & Plan', icon: '💳', route: '/dashboard/subscription' },
    { id: 'license', label: 'License & Devices', icon: '🔑', route: '/dashboard/license' },
    { id: 'referral', label: 'Referral & Growth', icon: '🌱', route: '/dashboard/referral' },
    { id: 'payouts', label: 'Payouts', icon: '💰', route: '/dashboard/payouts' },
    { id: 'notifications', label: 'Notifications', icon: '🔔', route: '/dashboard/notifications' },
    { id: 'security', label: 'Security', icon: '🔒', route: '/dashboard/security' },
    { id: 'support', label: 'Support', icon: '🛟', route: '/dashboard/support' },
  ]},
];

export default function UserLayout({ children }: { children: React.ReactNode }) {
  return <AppShell navItems={userNavItems} mode="user">{children}</AppShell>;
}
