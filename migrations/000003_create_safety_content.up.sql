CREATE TABLE IF NOT EXISTS safety_content (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category           TEXT NOT NULL,
    subcategory        TEXT NOT NULL,
    title              TEXT NOT NULL,
    body               TEXT NOT NULL,
    steps              JSONB DEFAULT '[]',
    tags               JSONB DEFAULT '[]',
    contextual_trigger TEXT,
    last_reviewed      TIMESTAMPTZ NOT NULL DEFAULT now()
);
