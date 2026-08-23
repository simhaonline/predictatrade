import { render, screen } from '@testing-library/react';

jest.mock('next/navigation', () => ({
  usePathname: () => '/dashboard/live',
}));

jest.mock('next-themes', () => ({
  useTheme: () => ({ theme: 'dark', setTheme: jest.fn() }),
}));

describe('User Sidebar Separation', () => {
  it('renders User Panel label for USER role', async () => {
    jest.doMock('@/providers/auth-provider', () => ({
      useAuth: () => ({
        user: { id: '2', email: 'user@test.com', role: 'USER', name: 'Test User' },
        sessionState: 'AUTHENTICATED',
        loading: false,
        login: jest.fn(),
        logout: jest.fn(),
        refreshUser: jest.fn(),
      }),
    }));
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.getByTestId('panel-label')).toHaveTextContent('User Panel');
  });

  it('shows user navigation items', async () => {
    jest.doMock('@/providers/auth-provider', () => ({
      useAuth: () => ({
        user: { id: '2', email: 'user@test.com', role: 'USER', name: 'Test User' },
        sessionState: 'AUTHENTICATED',
        loading: false,
        login: jest.fn(),
        logout: jest.fn(),
        refreshUser: jest.fn(),
      }),
    }));
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.getByText('MetaTrader Client')).toBeInTheDocument();
    expect(screen.getByText('Referral & Earnings')).toBeInTheDocument();
    expect(screen.getByText('Billing & Subscription')).toBeInTheDocument();
  });

  it('does NOT show admin-only items to user', async () => {
    jest.doMock('@/providers/auth-provider', () => ({
      useAuth: () => ({
        user: { id: '2', email: 'user@test.com', role: 'USER', name: 'Test User' },
        sessionState: 'AUTHENTICATED',
        loading: false,
        login: jest.fn(),
        logout: jest.fn(),
        refreshUser: jest.fn(),
      }),
    }));
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    expect(screen.queryByText('Signal Panel')).not.toBeInTheDocument();
    expect(screen.queryByText('Scoring Board')).not.toBeInTheDocument();
    expect(screen.queryByText('Platform Operations')).not.toBeInTheDocument();
    expect(screen.queryByText('System Health')).not.toBeInTheDocument();
    expect(screen.queryByText('License Management')).not.toBeInTheDocument();
  });
});
