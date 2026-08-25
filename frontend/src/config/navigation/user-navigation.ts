import {
  IconDashboard, IconChartLine, IconChartBar, IconDeviceDesktop, IconBolt,
  IconFileAnalytics, IconTestPipe, IconUsers, IconCreditCard, IconSettings,
  IconShieldLock, IconBell, IconLifebuoy, IconWallet, IconCertificate,
  IconHistory,
} from '@tabler/icons-react';

export interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  section?: string;
}

export const userNavigation: NavItem[] = [
  // ── Trading ──
  { label: 'Real-Time Console', href: '/dashboard/live', icon: IconDashboard, section: 'Trading' },
  { label: 'Signal Accuracy', href: '/dashboard/signal-accuracy', icon: IconChartBar, section: 'Trading' },
  { label: 'Signals', href: '/dashboard/signals', icon: IconChartLine, section: 'Trading' },
  { label: 'MetaTrader Client', href: '/dashboard/mt4-mt5-client', icon: IconDeviceDesktop, section: 'Trading' },
  { label: 'Strategy Preferences', href: '/dashboard/strategies', icon: IconBolt, section: 'Trading' },
  { label: 'Trading Reports', href: '/dashboard/trading-reports', icon: IconFileAnalytics, section: 'Trading' },
  { label: 'Backtest', href: '/dashboard/backtest', icon: IconTestPipe, section: 'Trading' },

  // ── Growth ──
  { label: 'Referral & Earnings', href: '/dashboard/referrals', icon: IconUsers, section: 'Growth' },
  { label: 'Billing & Subscription', href: '/dashboard/billing', icon: IconCreditCard, section: 'Growth' },
  { label: 'Payouts', href: '/dashboard/payouts', icon: IconWallet, section: 'Growth' },

  // ── Account ──
  { label: 'License', href: '/dashboard/license', icon: IconCertificate, section: 'Account' },
  { label: 'Security', href: '/dashboard/security', icon: IconShieldLock, section: 'Account' },
  { label: 'Activity Log', href: '/dashboard/activity-log', icon: IconHistory, section: 'Account' },
  { label: 'Notifications', href: '/dashboard/notifications', icon: IconBell, section: 'Account' },
  { label: 'Settings', href: '/dashboard/settings', icon: IconSettings, section: 'Account' },

  // ── Help ──
  { label: 'Support', href: '/dashboard/support', icon: IconLifebuoy, section: 'Help' },
];
