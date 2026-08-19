import { getApiErrorMessage } from '@/lib/errors';

describe('API Error Normalization', () => {
  it('should extract message from axios error', () => {
    const err = { response: { data: { message: 'Invalid credentials' } } };
    expect(getApiErrorMessage(err, 'fallback')).toBe('Invalid credentials');
  });

  it('should use fallback for unknown error shape', () => {
    expect(getApiErrorMessage(null, 'fallback')).toBe('fallback');
    expect(getApiErrorMessage(undefined, 'fallback')).toBe('fallback');
    expect(getApiErrorMessage('string', 'fallback')).toBe('fallback');
  });

  it('should use fallback when message missing', () => {
    const err = { response: { data: {} } };
    expect(getApiErrorMessage(err, 'fallback')).toBe('fallback');
  });
});
