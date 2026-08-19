import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

jest.mock('next-themes', () => ({
  useTheme: () => ({ theme: 'dark', setTheme: jest.fn() }),
}));

describe('AccessibilitySettings', () => {
  it('renders theme section', async () => {
    const { default: AccessibilitySettings } = await import('@/components/accessibility-settings');
    render(<AccessibilitySettings />);
    expect(screen.getByText('Display Theme')).toBeInTheDocument();
  });

  it('renders accessibility controls', async () => {
    const { default: AccessibilitySettings } = await import('@/components/accessibility-settings');
    render(<AccessibilitySettings />);
    expect(screen.getByText(/Font Scale/)).toBeInTheDocument();
    expect(screen.getByText('High Contrast')).toBeInTheDocument();
    expect(screen.getByText('Reduce Motion')).toBeInTheDocument();
    expect(screen.getByText('Keyboard Navigation')).toBeInTheDocument();
  });
});
