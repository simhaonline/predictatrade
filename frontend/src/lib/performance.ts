export function rafBatch(callback: () => void) {
  if (typeof window === 'undefined') return;
  requestAnimationFrame(callback);
}

export class RingBuffer<T> {
  private buffer: T[];
  private size: number;
  private index = 0;
  private count = 0;

  constructor(size: number) {
    this.size = size;
    this.buffer = new Array(size);
  }

  push(item: T) {
    this.buffer[this.index] = item;
    this.index = (this.index + 1) % this.size;
    if (this.count < this.size) this.count++;
  }

  toArray(): T[] {
    const result: T[] = [];
    for (let i = 0; i < this.count; i++) {
      result.push(this.buffer[(this.index - this.count + i + this.size) % this.size]);
    }
    return result;
  }

  length() {
    return this.count;
  }
}
