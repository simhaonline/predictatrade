export function getApiErrorMessage(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err !== null) {
    const e = err as { response?: { data?: { message?: string } } };
    return e.response?.data?.message || fallback;
  }
  return fallback;
}
