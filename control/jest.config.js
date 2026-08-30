/**
 * Jest configuration for the Predict-A-Trade control plane.
 *
 * NestJS 12 ships ESM-only packages, and Jest 30 deliberately disables native
 * `require(esm)` inside its VM (jest-util overrides `process.features.require_module`
 * to false). The only reliable way to run the suite on Node 24 is ESM mode, driven
 * by ts-jest with `useESM`. Tests must therefore use `import` (not `require`).
 */
/** @type {import('jest').Config} */
module.exports = {
  moduleFileExtensions: ['js', 'json', 'ts'],
  rootDir: 'src',
  testRegex: '.*\\.spec\\.ts$',
  transform: {
    '^.+\\.tsx?$': [
      'ts-jest',
      {
        useESM: true,
        tsconfig: 'tsconfig.spec.json',
      },
    ],
  },
  extensionsToTreatAsEsm: ['.ts'],
  testEnvironment: 'node',
  moduleNameMapper: {
    // Allow extensionless/`.js` relative imports to resolve to `.ts` sources.
    '^(\\.{1,2}/.*)\\.js$': '$1',
  },
  collectCoverageFrom: ['**/*.(t|j)s'],
  coverageDirectory: '../coverage',
};
