-- 096_ai_providers.sql — AI provider registry (check.md #17)
CREATE TABLE IF NOT EXISTS ai.ai_providers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(80) NOT NULL UNIQUE,
    provider     VARCHAR(40) NOT NULL,        -- ollama | openai | custom
    base_url     VARCHAR(255) NOT NULL,
    api_key_ref  VARCHAR(120),                -- ref to secret manager key (never store raw)
    model        VARCHAR(120),
    enabled      BOOLEAN NOT NULL DEFAULT true,
    health_check BOOLEAN NOT NULL DEFAULT false, -- last health check ok?
    last_checked TIMESTAMPTZ,
    created_by   UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ai.ai_providers (name, provider, base_url, model, enabled, created_by)
SELECT 'Ollama Local', 'ollama', 'http://ollama:11434', '', false, NULL
WHERE NOT EXISTS (SELECT 1 FROM ai.ai_providers WHERE name='Ollama Local')
  AND EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='ai' LIMIT 1);
