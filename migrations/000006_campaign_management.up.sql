ALTER TABLE campaigns
  ADD COLUMN slug text,
  ADD COLUMN brief text,
  ADD COLUMN thumbnail_url text;

UPDATE campaigns
SET slug = regexp_replace(lower(btrim(name)), '[^a-z0-9]+', '-', 'g'),
    brief = description;

ALTER TABLE campaigns
  ALTER COLUMN slug SET NOT NULL,
  ALTER COLUMN brief SET NOT NULL,
  ADD CONSTRAINT campaigns_slug_format_check CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  ADD CONSTRAINT campaigns_slug_length_check CHECK (char_length(slug) <= 140),
  ADD CONSTRAINT campaigns_brief_length_check CHECK (char_length(brief) BETWEEN 10 AND 1000),
  ADD CONSTRAINT campaigns_thumbnail_url_check CHECK (thumbnail_url IS NULL OR thumbnail_url ~ '^https?://');

CREATE UNIQUE INDEX campaigns_slug_key ON campaigns (slug);

