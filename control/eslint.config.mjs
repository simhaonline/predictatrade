import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';

export default [
  {
    files: ['src/**/*.ts', 'test/**/*.ts'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        project: './tsconfig.json',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'no-console': 'off',
    },
    ignores: ['dist/**', 'node_modules/**'],
  },
  {
    // Spec files live outside tsconfig.json's project (excluded there so Jest
    // type-checks them via tsconfig.spec.json). Flat config merges ALL matching
    // elements in order, so this block MUST come after the generic one above —
    // otherwise the generic src/**/*.ts project (tsconfig.json) wins and every
    // *.spec.ts throws a "file was not found in any of the provided project(s)"
    // parse error (13 lint errors after the NestJS 10→12 upgrade).
    files: ['src/**/*.spec.ts', 'test/**/*.spec.ts', 'src/**/*.e2e-spec.ts'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        project: './tsconfig.spec.json',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'no-console': 'off',
    },
    ignores: ['dist/**', 'node_modules/**'],
  },
  {
    ignores: ['dist/**', 'node_modules/**', 'coverage/**'],
  },
];