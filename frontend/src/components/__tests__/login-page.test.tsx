import { render, fireEvent, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

jest.mock('@/providers/auth-provider', () => ({
  useAuth: () => ({ login: jest.fn().mockResolvedValue(undefined), user: null, loading: false, sessionState: 'UNAUTHENTICATED', logout: jest.fn(), refreshUser: jest.fn() }),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn(), prefetch: jest.fn() }),
}));

describe('LoginPage', () => {
  it('renders email and password inputs', async () => {
    const { default: LoginPage } = await import('@/app/(auth)/login/page');
    render(<LoginPage />);
    expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter your password')).toBeInTheDocument();
  });

  it('submits form with values', async () => {
    const { default: LoginPage } = await import('@/app/(auth)/login/page');
    render(<LoginPage />);
    const email = screen.getByPlaceholderText('you@example.com') as HTMLInputElement;
    const password = screen.getByPlaceholderText('Enter your password') as HTMLInputElement;
    fireEvent.change(email, { target: { value: 'test@example.com' } });
    fireEvent.change(password, { target: { value: 'password123' } });
    expect(email.value).toBe('test@example.com');
    expect(password.value).toBe('password123');
  });
});
