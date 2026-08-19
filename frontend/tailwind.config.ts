import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: 'class',
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        /* Semantic PAT theme tokens */
        'pat-bg-page': 'hsl(var(--pat-bg-page))',
        'pat-bg-surface': 'hsl(var(--pat-bg-surface))',
        'pat-bg-surface-secondary': 'hsl(var(--pat-bg-surface-secondary))',
        'pat-bg-header': 'hsl(var(--pat-bg-header))',

        'pat-bg-sidebar': 'hsl(var(--pat-bg-sidebar))',
        'pat-bg-sidebar-hover': 'hsl(var(--pat-bg-sidebar-hover))',
        'pat-bg-sidebar-active': 'hsl(var(--pat-bg-sidebar-active))',
        'pat-text-sidebar': 'hsl(var(--pat-text-sidebar))',
        'pat-text-sidebar-active': 'hsl(var(--pat-text-sidebar-active))',
        'pat-text-sidebar-muted': 'hsl(var(--pat-text-sidebar-muted))',
        'pat-border-sidebar': 'hsl(var(--pat-border-sidebar))',

        'pat-text-primary': 'hsl(var(--pat-text-primary))',
        'pat-text-secondary': 'hsl(var(--pat-text-secondary))',
        'pat-text-muted': 'hsl(var(--pat-text-muted))',
        'pat-text-inverse': 'hsl(var(--pat-text-inverse))',

        'pat-border': 'hsl(var(--pat-border))',
        'pat-border-strong': 'hsl(var(--pat-border-strong))',

        primary: {
          DEFAULT: 'hsl(var(--pat-primary))',
          foreground: 'hsl(var(--pat-primary-foreground))',
        },
        'pat-primary-hover': 'hsl(var(--pat-primary-hover))',

        'pat-success': 'hsl(var(--pat-success))',
        'pat-danger': 'hsl(var(--pat-danger))',
        'pat-warning': 'hsl(var(--pat-warning))',
        'pat-info': 'hsl(var(--pat-info))',

        'pat-card-bg': 'hsl(var(--pat-card-bg))',
        'pat-card-border': 'hsl(var(--pat-card-border))',

        'pat-table-bg': 'hsl(var(--pat-table-bg))',
        'pat-table-header': 'hsl(var(--pat-table-header))',
        'pat-table-hover': 'hsl(var(--pat-table-hover))',
        'pat-table-border': 'hsl(var(--pat-table-border))',

        'pat-input-bg': 'hsl(var(--pat-input-bg))',
        'pat-input-border': 'hsl(var(--pat-input-border))',
        'pat-input-text': 'hsl(var(--pat-input-text))',

        /* Badge surfaces */
        'pat-badge-success-bg': 'hsl(var(--pat-badge-success-bg))',
        'pat-badge-success-text': 'hsl(var(--pat-badge-success-text))',
        'pat-badge-danger-bg': 'hsl(var(--pat-badge-danger-bg))',
        'pat-badge-danger-text': 'hsl(var(--pat-badge-danger-text))',
        'pat-badge-warning-bg': 'hsl(var(--pat-badge-warning-bg))',
        'pat-badge-warning-text': 'hsl(var(--pat-badge-warning-text))',
        'pat-badge-info-bg': 'hsl(var(--pat-badge-info-bg))',
        'pat-badge-info-text': 'hsl(var(--pat-badge-info-text))',
        'pat-badge-neutral-bg': 'hsl(var(--pat-badge-neutral-bg))',
        'pat-badge-neutral-text': 'hsl(var(--pat-badge-neutral-text))',

        /* Trading-specific semantic colors (theme-independent) */
        'pat-price-bid': 'hsl(var(--pat-price-bid))',      /* #10B981 */
        'pat-price-ask': 'hsl(var(--pat-price-ask))',      /* #EF4444 */
        'pat-value-tp': 'hsl(var(--pat-value-tp))',        /* #10B981 */
        'pat-value-sl': 'hsl(var(--pat-value-sl))',        /* #EF4444 */
        'pat-session': 'hsl(var(--pat-session-highlight))', /* #EAB308 */
        'pat-candidate-buy': 'hsl(var(--pat-candidate-buy))',  /* #F59E0B */
        'pat-candidate-sell': 'hsl(var(--pat-candidate-sell))', /* #FB923C */
      },
      borderRadius: {
        lg: 'var(--pat-radius)',
        md: 'calc(var(--pat-radius) - 2px)',
        sm: 'calc(var(--pat-radius) - 4px)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [require('tailwindcss-animate')] // eslint-disable-line @typescript-eslint/no-require-imports,
};

export default config;
