DROP INDEX IF EXISTS campaigns_slug_key;

ALTER TABLE campaigns
  DROP CONSTRAINT IF EXISTS campaigns_thumbnail_url_check,
  DROP CONSTRAINT IF EXISTS campaigns_brief_length_check,
  DROP CONSTRAINT IF EXISTS campaigns_slug_length_check,
  DROP CONSTRAINT IF EXISTS campaigns_slug_format_check,
  DROP COLUMN IF EXISTS thumbnail_url,
  DROP COLUMN IF EXISTS brief,
  DROP COLUMN IF EXISTS slug;
