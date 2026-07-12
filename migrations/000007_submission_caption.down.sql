ALTER TABLE clip_submissions
  DROP CONSTRAINT IF EXISTS clip_submissions_caption_length_check,
  DROP COLUMN IF EXISTS caption;
