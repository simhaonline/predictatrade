import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import '@/styles/globals.css';
import { ThemeProvider } from '@/providers/theme-provider';
import { AuthProvider } from '@/providers/auth-provider';
import { ReactQueryProvider } from '@/providers/query-provider';
import { CookieConsentProvider } from '@/providers/cookie-consent-provider';
import { Toaster } from 'sonner';

const inter = Inter({ subsets: ['latin'], display: 'swap' });

export const metadata: Metadata = {
  title: 'Predict-A-Trade XAUUSD',
  description: 'Nano-Scope Market Analysis & Probability Decision Engine',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="icon" href="/predict-a-trade_favicon.svg" type="image/svg+xml" />
        <link rel="apple-touch-icon" href="/predict-a-trade_app-icon-512x512.png" />
        {/* Force browser to never cache HTML pages — prevents stale JS chunks */}
        <meta httpEquiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
        <meta httpEquiv="Pragma" content="no-cache" />
        <meta httpEquiv="Expires" content="0" />
      </head>
      {/* suppressHydrationWarning on body to prevent #418 from next-themes class injection and browser extensions */}
      <body className={inter.className} suppressHydrationWarning>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false} disableTransitionOnChange>
          <CookieConsentProvider>
            <ReactQueryProvider>
              <AuthProvider>
                {children}
                {/* theme="dark" matches ThemeProvider defaultTheme — prevents system-theme detection mismatch */}
                <Toaster position="top-right" richColors theme="dark" />
              </AuthProvider>
            </ReactQueryProvider>
          </CookieConsentProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
