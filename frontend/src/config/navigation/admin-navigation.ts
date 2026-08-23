import {
  IconDashboard, IconChartLine, IconCategory, IconBolt, IconChartBar,
  IconKey, IconShield, IconUsers, IconReceipt, IconCoin,
  IconDeviceDesktop, IconFileAnalytics, IconTestPipe, IconClipboardList,
  IconTool, IconHeartbeat, IconSettings, IconActivity,
  IconAlertTriangle, IconListCheck, IconAdjustments, IconWallet,
  IconReportMoney, IconBrain, IconBroadcast, IconWorld, IconRocket,
  IconDatabase, IconFlag, IconBuildingBank, IconServer,
} from '@tabler/icons-react';

export interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
}

export const adminNavigation: NavItem[] = [
  { label: 'Live Dashboard', href: '/admin/dashboard', icon: IconDashboard },
  { label: 'Signal Panel', href: '/admin/signals', icon: IconChartLine },
  { label: 'Indicator Panel', href: '/admin/indicators', icon: IconCategory },
  { label: 'Indicator Monitor', href: '/admin/indicator-monitor', icon: IconActivity },
  { label: 'Strategy Panel', href: '/admin/strategies', icon: IconBolt },
  { label: 'Regime Diagnostics', href: '/admin/regime-diagnostics', icon: IconChartBar },
  { label: 'Scoring Board', href: '/admin/scoring-board', icon: IconChartBar },
  { label: 'Activations', href: '/admin/activations', icon: IconKey },
  { label: 'License Management', href: '/admin/licenses', icon: IconShield },
  { label: 'User Onboarding', href: '/admin/users', icon: IconUsers },
  { label: 'Subscription Management', href: '/admin/subscriptions', icon: IconReceipt },
  { label: 'Plans & Entitlements', href: '/admin/plans-entitlements', icon: IconListCheck },
  { label: 'Billing & Payouts', href: '/admin/billing', icon: IconCoin },
  { label: 'Commission Control', href: '/admin/commission-control-center', icon: IconAdjustments },
  { label: 'Commission Operations', href: '/admin/commission-operations', icon: IconCoin },
  { label: 'Payout Operations', href: '/admin/payout-operations', icon: IconWallet },
  { label: 'Finance & Referral Reports', href: '/admin/finance-referral-reports', icon: IconReportMoney },
  { label: 'Risk Center', href: '/admin/risk-center', icon: IconAlertTriangle },
  { label: 'MT Accounts', href: '/admin/mt-accounts', icon: IconServer },
  { label: 'AI Providers', href: '/admin/ai-providers', icon: IconBrain },
  { label: 'Market Data', href: '/admin/market-data', icon: IconBroadcast },
  { label: 'Macro & News', href: '/admin/macro-news', icon: IconWorld },
  { label: 'Releases', href: '/admin/releases', icon: IconRocket },
  { label: 'Backup & DR', href: '/admin/backup-dr', icon: IconDatabase },
  { label: 'Feature Flags', href: '/admin/feature-flags', icon: IconFlag },
  { label: 'Broker Qualification', href: '/admin/broker-qualification', icon: IconBuildingBank },
  { label: 'Referral & Commissions', href: '/admin/referrals', icon: IconUsers },
  { label: 'Device Auth', href: '/admin/device-auth', icon: IconDeviceDesktop },
  { label: 'Trading Reports', href: '/admin/trading-reports', icon: IconFileAnalytics },
  { label: 'Backtesting Reports', href: '/admin/backtesting', icon: IconTestPipe },
  { label: 'Logs & Audit', href: '/admin/logs', icon: IconClipboardList },
  { label: 'Platform Operations', href: '/admin/operations', icon: IconTool },
  { label: 'System Health', href: '/admin/health', icon: IconHeartbeat },
  { label: 'Settings', href: '/admin/settings', icon: IconSettings },
];
