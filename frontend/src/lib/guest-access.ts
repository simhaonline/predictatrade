// live.predictatrade.com access gate per plan policy:
// FREE / anonymous visitors see the live dashboard only 11:00–13:00 GMT+3
// (= 08:00–10:00 UTC). Paid / active subscribers get 24/7 access.
import { SubscriptionContext, isActiveSubscription } from "./subscription-access";

const OPEN_UTC_HOURS = [8, 9];  // 11,12 in GMT+3 (08:00–09:59 UTC)

export function isLivePreviewOpen(
  now = new Date(),
  ctx?: SubscriptionContext,
): boolean {
  // Active paid subscribers bypass the time window entirely (24/7 access).
  if (isActiveSubscription(ctx)) return true;
  const h = now.getUTCHours();
  return OPEN_UTC_HOURS.includes(h);
}

export function nextLiveOpenUTC(
  now = new Date(),
  ctx?: SubscriptionContext,
): string | null {
  if (isLivePreviewOpen(now, ctx)) return null;
  const d = new Date(now);
  // advance until we hit an open hour boundary at next hour start
  d.setUTCMinutes(0, 0, 0);
  do {
    d.setUTCHours(d.getUTCHours() + 1);
    if (isLivePreviewOpen(d, ctx)) return d.toISOString();
  } while (d.getTime() - now.getTime() < 1000 * 60 * 60 * 24 * 7);
  return null;
}
