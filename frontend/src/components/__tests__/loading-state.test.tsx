import { render, screen } from '@testing-library/react';

jest.mock('next/navigation', () => ({
  usePathname: () => '/admin/dashboard',
}));

jest.mock('next-themes', () => ({
  useTheme: () => ({ theme: 'dark', setTheme: jest.fn() }),
}));

describe('Session Loading State', () => {
  it('shows loading state (not User Panel) while session is unresolved', async () => {
    jest.doMock('@/providers/auth-provider', () => ({
      useAuth: () => ({
        user: null,
        sessionState: 'LOADING',
        loading: true,
        login: jest.fn(),
        logout: jest.fn(),
        refreshUser: jest.fn(),
      }),
    }));
    const { default: Sidebar } = await import('@/components/layout/sidebar');
    render(<Sidebar />);
    // Should show "Loading…" NOT "User Panel"
    expect(screen.getByTestId('panel-label')).toHaveTextContent('Loading');
    expect(screen.queryByText('Live Dashboard')).not.toBeInTheDocument();
  });
});
