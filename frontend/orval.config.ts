import { defineConfig } from 'orval';

export default defineConfig({
  predictatrade: {
    input: '../control/openapi.json',
    output: {
      target: './src/generated/api.ts',
      client: 'react-query',
      mode: 'tags-split',
      schemas: './src/generated/models',
      override: {
        mutator: {
          path: './src/lib/axios-instance.ts',
          name: 'customInstance',
        },
      },
    },
  },
});
