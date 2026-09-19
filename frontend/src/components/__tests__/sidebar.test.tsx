import { render, screen } from '@testing-library/react';

jest.mock('@/providers/auth-provider', () => ({
  useAuth: () => ({
    user: { id: '1', email: 'admin@test.com', role: 'ADMIN', name: 'Test Admin' },
    sessionState: 'AUTHENTICATED',
    loading: false,
    login: jest.fn(),
    logout: jest.fn(),
    refreshUser: jest.fn(),
  }),
}));

jest.mock('next/navigation', () => ({
  usePathname: () => '/admin/dashboard',
}));

jest.mock('next-themes', () => ({
  useTheme: () => ({ theme: 'dark', setTheme: jest.fn() }),
}));

describe('Admin Sidebar', () => {
  it('renders no panel label in footer (labels removed)', async () => {
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.getByTestId('panel-label')).toHaveTextContent('');
  });

  it('shows admin navigation items including Signal Panel', async () => {
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.getByText('Signal Monitor')).toBeInTheDocument();
    expect(screen.getByText('Indicator Monitor')).toBeInTheDocument();
    expect(screen.getByText('Scoring Board')).toBeInTheDocument();
    // Platform Operations merged into Risk Center (2026-09-19) — the nav shows
    // the single merged entry; the old label must NOT reappear.
    expect(screen.getByText('Risk Center')).toBeInTheDocument();
    expect(screen.queryByText('Platform Operations')).not.toBeInTheDocument();
    expect(screen.getByText('System Health')).toBeInTheDocument();
  });

  it('does NOT show user-only items', async () => {
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.queryByText('MetaTrader Client')).not.toBeInTheDocument();
    expect(screen.queryByText('Referral & Earnings')).not.toBeInTheDocument();
    expect(screen.queryByText('Billing & Subscription')).not.toBeInTheDocument();
    expect(screen.queryByText('Strategy Preferences')).not.toBeInTheDocument();
  });
});
