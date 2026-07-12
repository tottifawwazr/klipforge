ALTER TABLE clip_submissions
  ADD COLUMN caption text NOT NULL DEFAULT '';

ALTER TABLE clip_submissions
  ADD CONSTRAINT clip_submissions_caption_length_check CHECK (char_length(caption) <= 2200);

