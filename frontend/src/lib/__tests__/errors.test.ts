import { getApiErrorMessage } from '@/lib/errors';

describe('getApiErrorMessage', () => {
  it('returns fallback for non-object errors', () => {
    expect(getApiErrorMessage('oops', 'fallback')).toBe('fallback');
  });

  it('returns message from error response', () => {
    const err = { response: { data: { message: 'Bad request' } } };
    expect(getApiErrorMessage(err, 'fallback')).toBe('Bad request');
  });

  it('returns fallback when message is missing', () => {
    expect(getApiErrorMessage({}, 'fallback')).toBe('fallback');
  });
});
