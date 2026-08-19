import { RingBuffer } from '@/lib/performance';

describe('RingBuffer', () => {
  it('pushes items and returns them in order', () => {
    const buf = new RingBuffer<number>(3);
    buf.push(1);
    buf.push(2);
    expect(buf.toArray()).toEqual([1, 2]);
  });

  it('overwrites old items when full', () => {
    const buf = new RingBuffer<number>(2);
    buf.push(1);
    buf.push(2);
    buf.push(3);
    expect(buf.toArray()).toEqual([2, 3]);
  });
});
