import { NextResponse } from 'next/server';
import { readFileSync } from 'fs';

// Production build-id endpoint — the service worker uses this to detect a
// new deploy and purge its caches + reload the page automatically. An old
// client therefore always converges to the newest build with no user action.
export const dynamic = 'force-dynamic';

export async function GET() {
  try {
    const buildId = readFileSync('.next/BUILD_ID', 'utf8').trim();
    return NextResponse.json({ buildId }, { headers: { 'Cache-Control': 'no-store' } });
  } catch {
    const buildId = process.env.BUILD_ID || 'unknown';
    return NextResponse.json({ buildId }, { headers: { 'Cache-Control': 'no-store' } });
  }
}
