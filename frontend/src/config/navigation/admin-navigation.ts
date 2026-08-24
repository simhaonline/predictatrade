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
  section?: string;
}

export const adminNavigation: NavItem[] = [
  // ── Real-Time Operations ──
  { label: 'Real-Time Console', href: '/admin/dashboard', icon: IconDashboard, section: 'Real-Time Operations' },
  { label: 'Signal Panel', href: '/admin/signals', icon: IconChartLine, section: 'Real-Time Operations' },
  { label: 'Indicator Monitor', href: '/admin/indicator-monitor', icon: IconActivity, section: 'Real-Time Operations' },
  { label: 'Strategy Panel', href: '/admin/strategies', icon: IconBolt, section: 'Real-Time Operations' },
  { label: 'Regime Diagnostics', href: '/admin/regime-diagnostics', icon: IconChartBar, section: 'Real-Time Operations' },
  { label: 'Scoring Board', href: '/admin/scoring-board', icon: IconChartBar, section: 'Real-Time Operations' },

  // ── Risk & Safety ──
  { label: 'Risk Center', href: '/admin/risk-center', icon: IconAlertTriangle, section: 'Risk & Safety' },
  { label: 'MT Accounts', href: '/admin/mt-accounts', icon: IconServer, section: 'Risk & Safety' },
  { label: 'Device Auth', href: '/admin/device-auth', icon: IconDeviceDesktop, section: 'Risk & Safety' },
  { label: 'License Management', href: '/admin/licenses', icon: IconShield, section: 'Risk & Safety' },
  { label: 'Activations', href: '/admin/activations', icon: IconKey, section: 'Risk & Safety' },

  // ── Customers & Billing ──
  { label: 'Users & Onboarding', href: '/admin/users', icon: IconUsers, section: 'Customers & Billing' },
  { label: 'Subscription Management', href: '/admin/subscriptions', icon: IconReceipt, section: 'Customers & Billing' },
  { label: 'Plans & Entitlements', href: '/admin/plans-entitlements', icon: IconListCheck, section: 'Customers & Billing' },
  { label: 'Billing & Invoices', href: '/admin/billing', icon: IconCoin, section: 'Customers & Billing' },
  { label: 'Commission Operations', href: '/admin/commission-operations', icon: IconCoin, section: 'Customers & Billing' },
  { label: 'Payout Operations', href: '/admin/payout-operations', icon: IconWallet, section: 'Customers & Billing' },
  { label: 'Referrals & Affiliates', href: '/admin/referrals', icon: IconUsers, section: 'Customers & Billing' },
  { label: 'Finance & Referral Reports', href: '/admin/finance-referral-reports', icon: IconReportMoney, section: 'Customers & Billing' },

  // ── Market & Intelligence ──
  { label: 'Market Data', href: '/admin/market-data', icon: IconBroadcast, section: 'Market & Intelligence' },
  { label: 'Macro & News', href: '/admin/macro-news', icon: IconWorld, section: 'Market & Intelligence' },
  { label: 'AI Providers', href: '/admin/ai-providers', icon: IconBrain, section: 'Market & Intelligence' },
  { label: 'Broker Qualification', href: '/admin/broker-qualification', icon: IconBuildingBank, section: 'Market & Intelligence' },

  // ── Platform & System ──
  { label: 'Signal Accuracy', href: '/admin/signal-accuracy', icon: IconChartBar, section: 'Market & Intelligence' },
  { label: 'Releases', href: '/admin/releases', icon: IconRocket, section: 'Platform & System' },
  { label: 'Backup & DR', href: '/admin/backup-dr', icon: IconDatabase, section: 'Platform & System' },
  { label: 'Feature Flags', href: '/admin/feature-flags', icon: IconFlag, section: 'Platform & System' },
  { label: 'Trading Reports', href: '/admin/trading-reports', icon: IconFileAnalytics, section: 'Platform & System' },
  { label: 'Backtesting', href: '/admin/backtesting', icon: IconTestPipe, section: 'Platform & System' },
  { label: 'Platform Operations', href: '/admin/operations', icon: IconTool, section: 'Platform & System' },
  { label: 'Logs & Audit', href: '/admin/logs', icon: IconClipboardList, section: 'Platform & System' },
  { label: 'System Health', href: '/admin/health', icon: IconHeartbeat, section: 'Platform & System' },
  { label: 'Settings', href: '/admin/settings', icon: IconSettings, section: 'Platform & System' },
];
