ALTER TABLE clip_submissions
  ADD CONSTRAINT clip_submissions_review_note_length_check
    CHECK (review_note IS NULL OR char_length(review_note) <= 1000),
  ADD CONSTRAINT clip_submissions_reviewer_timestamp_check
    CHECK (reviewed_by_user_id IS NULL OR reviewed_at IS NOT NULL),
  ADD CONSTRAINT clip_submissions_moderation_state_check
    CHECK (
      (status = 'PENDING' AND reviewed_at IS NULL AND reviewed_by_user_id IS NULL AND review_note IS NULL)
      OR (status = 'APPROVED' AND reviewed_at IS NOT NULL AND review_note IS NULL)
      OR (status IN ('REJECTED', 'FLAGGED') AND reviewed_at IS NOT NULL AND btrim(review_note) <> '')
    );

CREATE INDEX clip_submissions_moderation_queue_idx
  ON clip_submissions (status, submitted_at DESC, id);
