DROP INDEX IF EXISTS clip_submissions_moderation_queue_idx;

ALTER TABLE clip_submissions
  DROP CONSTRAINT IF EXISTS clip_submissions_moderation_state_check,
  DROP CONSTRAINT IF EXISTS clip_submissions_reviewer_timestamp_check,
  DROP CONSTRAINT IF EXISTS clip_submissions_review_note_length_check;
