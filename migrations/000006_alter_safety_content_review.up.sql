-- last_reviewed shipped as NOT NULL DEFAULT now(), which made every inserted
-- row claim it had been reviewed the instant it was written. Same false claim
-- as the hardcoded UI badge, encoded where it looks authoritative.
ALTER TABLE safety_content
    ADD COLUMN slug                  TEXT,
    ADD COLUMN summary               TEXT NOT NULL DEFAULT '',
    ADD COLUMN sources               JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN content_hash          TEXT NOT NULL DEFAULT '',
    ADD COLUMN review_state          TEXT NOT NULL DEFAULT 'pending_review',
    ADD COLUMN reviewer_name         TEXT,
    ADD COLUMN reviewer_credential   TEXT,
    ADD COLUMN reviewed_content_hash TEXT,
    ALTER COLUMN last_reviewed DROP NOT NULL,
    ALTER COLUMN last_reviewed DROP DEFAULT;

UPDATE safety_content SET slug = id::text WHERE slug IS NULL;

ALTER TABLE safety_content
    ALTER COLUMN slug SET NOT NULL,
    ADD CONSTRAINT safety_content_slug_unique UNIQUE (slug),
    ADD CONSTRAINT safety_content_review_state_valid
        CHECK (review_state IN ('draft', 'pending_review', 'reviewed', 'withdrawn')),
    -- A row cannot claim review without a named reviewer, a date, and the
    -- hash of the exact text that was approved.
    ADD CONSTRAINT safety_content_reviewed_requires_provenance
        CHECK (review_state <> 'reviewed' OR (
            reviewer_name         IS NOT NULL AND
            last_reviewed         IS NOT NULL AND
            reviewed_content_hash IS NOT NULL));

CREATE INDEX idx_safety_content_category ON safety_content (category, subcategory);
