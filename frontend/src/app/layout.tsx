import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import '@/styles/globals.css';
import { ThemeProvider } from '@/providers/theme-provider';
import { AuthProvider } from '@/providers/auth-provider';
import { ReactQueryProvider } from '@/providers/query-provider';
import { CookieConsentProvider } from '@/providers/cookie-consent-provider';
import { ThemedToaster } from '@/components/themed-toaster';

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
        <meta httpEquiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
        <meta httpEquiv="Pragma" content="no-cache" />
        <meta httpEquiv="Expires" content="0" />
      </head>
      <body className={inter.className} suppressHydrationWarning>
        <ThemeProvider attribute="class" defaultTheme="light" enableSystem disableTransitionOnChange>
          <CookieConsentProvider>
            <ReactQueryProvider>
              <AuthProvider>
                {children}
                <ThemedToaster />
              </AuthProvider>
            </ReactQueryProvider>
          </CookieConsentProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
