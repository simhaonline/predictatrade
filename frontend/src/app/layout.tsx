import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Predict-A-Trade — XAUUSD Intelligence Platform',
  description: 'Production-grade XAUUSD prediction, signals, execution, and referral growth platform',
};

// No-FOUC inline script (SOW Section 187.3)
const themeScript = `
(function() {
  try {
    var stored = localStorage.getItem('pat-theme');
    var theme = stored || (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark');
    document.documentElement.setAttribute('data-theme', theme);
  } catch(e) {
    document.documentElement.setAttribute('data-theme', 'dark');
  }
})();
`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="dark" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body>{children}</body>
    </html>
  );
}
