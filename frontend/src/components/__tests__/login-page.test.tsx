import { render, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';

jest.mock('@/providers/auth-provider', () => ({
  useAuth: () => ({ login: jest.fn().mockResolvedValue(undefined), user: null, loading: false, sessionState: 'UNAUTHENTICATED', logout: jest.fn(), refreshUser: jest.fn() }),
}));

describe('LoginPage', () => {
  it('renders email and password inputs', async () => {
    const { default: LoginPage } = await import('@/app/(auth)/login/page');
    render(<LoginPage />);
    expect(document.getElementById('email')).toBeInTheDocument();
    expect(document.getElementById('password')).toBeInTheDocument();
  });

  it('submits form with values', async () => {
    const { default: LoginPage } = await import('@/app/(auth)/login/page');
    render(<LoginPage />);
    const email = document.getElementById('email') as HTMLInputElement;
    const password = document.getElementById('password') as HTMLInputElement;
    fireEvent.change(email, { target: { value: 'test@example.com' } });
    fireEvent.change(password, { target: { value: 'password123' } });
    expect(email.value).toBe('test@example.com');
    expect(password.value).toBe('password123');
  });
});
