import { render, screen } from '@testing-library/react';

describe('Error Pages', () => {
  it('should render 404 not-found page', async () => {
    const { default: NotFound } = await import('@/app/not-found');
    render(<NotFound />);
    expect(screen.getByText('404')).toBeInTheDocument();
  });

  it('should render 403 forbidden page', async () => {
    const { default: Forbidden } = await import('@/app/forbidden/page');
    render(<Forbidden />);
    expect(screen.getByText('403')).toBeInTheDocument();
  });

  it('should render error boundary', async () => {
    const { default: ErrorPage } = await import('@/app/error');
    const reset = jest.fn();
    render(<ErrorPage error={new Error('Test error')} reset={reset} />);
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });
});
