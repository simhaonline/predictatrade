import {
  IconDashboard, IconChartLine, IconDeviceDesktop, IconBolt,
  IconFileAnalytics, IconTestPipe, IconUsers, IconCreditCard, IconSettings,
} from '@tabler/icons-react';

export interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
}

export const userNavigation: NavItem[] = [
  { label: 'Live Dashboard', href: '/dashboard/live', icon: IconDashboard },
  { label: 'Signals', href: '/dashboard/signals', icon: IconChartLine },
  { label: 'MT4/MT5 Client', href: '/dashboard/mt4-mt5-client', icon: IconDeviceDesktop },
  { label: 'Strategy Preferences', href: '/dashboard/strategies', icon: IconBolt },
  { label: 'Trading Reports', href: '/dashboard/trading-reports', icon: IconFileAnalytics },
  { label: 'Backtest', href: '/dashboard/backtest', icon: IconTestPipe },
  { label: 'Referral & Earnings', href: '/dashboard/referrals', icon: IconUsers },
  { label: 'Billing & Subscription', href: '/dashboard/billing', icon: IconCreditCard },
  { label: 'Settings', href: '/dashboard/settings', icon: IconSettings },
];
