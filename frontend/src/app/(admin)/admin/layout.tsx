import { AppShell } from '@/components/ui/app-shell';

const adminNavItems = [
  { section: 'OPERATIONS', items: [
    { id: 'dashboard', label: 'Dashboard', icon: '📊', route: '/admin' },
    { id: 'users', label: 'Users', icon: '👤', route: '/admin/users' },
    { id: 'billing', label: 'Subscriptions & Billing', icon: '💳', route: '/admin/billing' },
    { id: 'plans', label: 'Plans & Pricing', icon: '📋', route: '/admin/plans' },
    { id: 'referrals', label: 'Referral Network', icon: '🌳', route: '/admin/referrals' },
    { id: 'commissions', label: 'Commission Control', icon: '⚖️', route: '/admin/commissions' },
    { id: 'payouts', label: 'Payouts', icon: '💰', route: '/admin/payouts' },
  ]},
  { section: 'LICENSING', items: [
    { id: 'licenses', label: 'Licenses', icon: '🔑', route: '/admin/licenses' },
    { id: 'devices', label: 'Devices & MT Accounts', icon: '🖥️', route: '/admin/devices' },
    { id: 'releases', label: 'Client Releases', icon: '📦', route: '/admin/releases' },
  ]},
  { section: 'SYSTEM', items: [
    { id: 'strategies', label: 'Strategies', icon: '⚡', route: '/admin/strategies' },
    { id: 'risk', label: 'Risk Controls', icon: '🚨', route: '/admin/risk' },
    { id: 'market-data', label: 'Market Data Health', icon: '📡', route: '/admin/market-data' },
    { id: 'ai', label: 'AI Providers', icon: '🤖', route: '/admin/ai' },
    { id: 'infra', label: 'Infrastructure', icon: '⚙️', route: '/admin/infrastructure' },
    { id: 'audit', label: 'Audit & Security', icon: '🔒', route: '/admin/audit' },
    { id: 'finance', label: 'Finance Reports', icon: '📈', route: '/admin/finance' },
    { id: 'support', label: 'Support Queue', icon: '🛟', route: '/admin/support' },
  ]},
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AppShell navItems={adminNavItems} mode="admin">{children}</AppShell>;
}
