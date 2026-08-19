import { cn } from '@/lib/utils';

describe('cn utility', () => {
  it('merges classes', () => {
    expect(cn('a', 'b')).toBe('a b');
  });

  it('handles conditional classes', () => {
    expect(cn('a', false && 'b', 'c')).toBe('a c');
  });
});
