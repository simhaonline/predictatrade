import {
  IconDashboard, IconChartLine, IconCategory, IconBolt, IconChartBar,
  IconKey, IconShield, IconUsers, IconReceipt, IconCoin,
  IconDeviceDesktop, IconFileAnalytics, IconTestPipe, IconClipboardList,
  IconTool, IconHeartbeat, IconSettings, IconActivity,
  IconAlertTriangle, IconListCheck, IconAdjustments, IconWallet,
  IconReportMoney, IconBrain, IconBroadcast, IconWorld, IconRocket,
  IconDatabase, IconFlag, IconBuildingBank, IconServer, IconDroplet, IconSparkles,
} from '@tabler/icons-react';

export interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  section?: string;
}

export const adminNavigation: NavItem[] = [
  // ── Trading Operations ──
  { label: 'Real-Time Console', href: '/admin/dashboard', icon: IconDashboard, section: 'Trading Operations' },
  { label: 'Signal Monitor', href: '/admin/signals', icon: IconChartLine, section: 'Trading Operations' },
  { label: 'Pipeline Monitor', href: '/admin/pipeline-monitor', icon: IconListCheck, section: 'Trading Operations' },
  { label: 'Signal Engine', href: '/admin/signal-engine', icon: IconBrain, section: 'Trading Operations' },
  { label: 'Agent Mesh', href: '/admin/agent-mesh', icon: IconBroadcast, section: 'Trading Operations' },
  { label: 'Scoring Board', href: '/admin/scoring-board', icon: IconChartBar, section: 'Trading Operations' },
  { label: 'Strategy Panel', href: '/admin/strategies', icon: IconBolt, section: 'Trading Operations' },
  { label: 'Devil Liquidity', href: '/admin/devil-liquidity', icon: IconDroplet, section: 'Trading Operations' },
  { label: 'Regime Diagnostics', href: '/admin/regime-diagnostics', icon: IconActivity, section: 'Trading Operations' },
  { label: 'Backtesting', href: '/admin/backtesting', icon: IconTestPipe, section: 'Trading Operations' },
  { label: 'Trading Reports', href: '/admin/trading-reports', icon: IconFileAnalytics, section: 'Trading Operations' },

  // ── Signal Quality ──
  { label: 'Indicator Monitor', href: '/admin/indicator-monitor', icon: IconActivity, section: 'Signal Quality' },
  { label: 'Signal Accuracy', href: '/admin/signal-accuracy', icon: IconListCheck, section: 'Signal Quality' },

  // ── Risk & Compliance ──
  { label: 'Risk Center', href: '/admin/risk-center', icon: IconAlertTriangle, section: 'Risk & Compliance' },

  // ── Customer Management ──
  { label: 'Users & Onboarding', href: '/admin/users', icon: IconUsers, section: 'Customer Management' },
  { label: 'Subscription Management', href: '/admin/subscriptions', icon: IconReceipt, section: 'Customer Management' },
  { label: 'License Management', href: '/admin/licenses', icon: IconShield, section: 'Customer Management' },
  { label: 'Activations', href: '/admin/activations', icon: IconKey, section: 'Customer Management' },

  // ── Devices & Infrastructure ──
  { label: 'Device Auth', href: '/admin/device-auth', icon: IconDeviceDesktop, section: 'Devices & Infrastructure' },
  { label: 'MT Accounts', href: '/admin/mt-accounts', icon: IconServer, section: 'Devices & Infrastructure' },

  // ── Finance ──
  { label: 'Plans & Entitlements', href: '/admin/plans-entitlements', icon: IconListCheck, section: 'Finance' },
  { label: 'Billing & Invoices', href: '/admin/billing', icon: IconCoin, section: 'Finance' },
  { label: 'Payments', href: '/admin/payments', icon: IconWallet, section: 'Finance' },
  { label: 'Payout Operations', href: '/admin/payout-operations', icon: IconReportMoney, section: 'Finance' },
  { label: 'Commission Operations', href: '/admin/commission-operations', icon: IconCoin, section: 'Finance' },
  { label: 'Referrals & Affiliates', href: '/admin/referrals', icon: IconUsers, section: 'Finance' },
  { label: 'Finance & Referral Reports', href: '/admin/finance-referral-reports', icon: IconFileAnalytics, section: 'Finance' },

  // ── Market Data & Intelligence ──
  { label: 'Market Data', href: '/admin/market-data', icon: IconBroadcast, section: 'Market Data & Intelligence' },
  { label: 'Macro Calendar', href: '/admin/macro-news', icon: IconWorld, section: 'Market Data & Intelligence' },
  { label: 'Macro Intelligence', href: '/admin/macro-intelligence', icon: IconChartBar, section: 'Market Data & Intelligence' },
  { label: 'Astro Intelligence', href: '/admin/astro', icon: IconSparkles, section: 'Market Data & Intelligence' },
  { label: 'AI Providers', href: '/admin/ai-providers', icon: IconBrain, section: 'Market Data & Intelligence' },
  { label: 'Broker Qualification', href: '/admin/broker-qualification', icon: IconBuildingBank, section: 'Market Data & Intelligence' },

  // ── System Operations ──
  { label: 'Platform Operations', href: '/admin/operations', icon: IconTool, section: 'System Operations' },
  { label: 'Logs & Audit', href: '/admin/logs', icon: IconClipboardList, section: 'System Operations' },
  { label: 'System Health', href: '/admin/health', icon: IconHeartbeat, section: 'System Operations' },
  { label: 'Feature Flags', href: '/admin/feature-flags', icon: IconFlag, section: 'System Operations' },
  { label: 'Backup & DR', href: '/admin/backup-dr', icon: IconDatabase, section: 'System Operations' },
  { label: 'Releases', href: '/admin/releases', icon: IconRocket, section: 'System Operations' },
  { label: 'Settings', href: '/admin/settings', icon: IconSettings, section: 'System Operations' },
];

