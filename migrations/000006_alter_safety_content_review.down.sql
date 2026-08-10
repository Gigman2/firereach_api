DROP INDEX IF EXISTS idx_safety_content_category;

ALTER TABLE safety_content
    DROP CONSTRAINT IF EXISTS safety_content_reviewed_requires_provenance,
    DROP CONSTRAINT IF EXISTS safety_content_review_state_valid,
    DROP CONSTRAINT IF EXISTS safety_content_slug_unique,
    DROP COLUMN IF EXISTS reviewed_content_hash,
    DROP COLUMN IF EXISTS reviewer_credential,
    DROP COLUMN IF EXISTS reviewer_name,
    DROP COLUMN IF EXISTS review_state,
    DROP COLUMN IF EXISTS content_hash,
    DROP COLUMN IF EXISTS sources,
    DROP COLUMN IF EXISTS summary,
    DROP COLUMN IF EXISTS slug;

UPDATE safety_content SET last_reviewed = now() WHERE last_reviewed IS NULL;

ALTER TABLE safety_content
    ALTER COLUMN last_reviewed SET DEFAULT now(),
    ALTER COLUMN last_reviewed SET NOT NULL;
