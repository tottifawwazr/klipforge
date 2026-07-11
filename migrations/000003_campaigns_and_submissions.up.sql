CREATE TABLE campaigns (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  brand_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  name text NOT NULL,
  description text,
  status text NOT NULL DEFAULT 'DRAFT',
  budget_amount numeric(14, 2) NOT NULL DEFAULT 0,
  remaining_budget_amount numeric(14, 2) NOT NULL DEFAULT 0,
  cpm_amount numeric(14, 4),
  max_payout_per_clip numeric(14, 2) NOT NULL DEFAULT 0,
  currency_code text NOT NULL DEFAULT 'USD',
  start_date date NOT NULL,
  end_date date NOT NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT campaigns_name_not_blank_check CHECK (btrim(name) <> ''),
  CONSTRAINT campaigns_status_check CHECK (status IN ('DRAFT', 'ACTIVE', 'PAUSED', 'COMPLETED', 'CANCELLED')),
  CONSTRAINT campaigns_budget_non_negative_check CHECK (budget_amount >= 0),
  CONSTRAINT campaigns_remaining_budget_non_negative_check CHECK (remaining_budget_amount >= 0),
  CONSTRAINT campaigns_remaining_budget_limit_check CHECK (remaining_budget_amount <= budget_amount),
  CONSTRAINT campaigns_cpm_non_negative_check CHECK (cpm_amount IS NULL OR cpm_amount >= 0),
  CONSTRAINT campaigns_max_payout_non_negative_check CHECK (max_payout_per_clip >= 0),
  CONSTRAINT campaigns_currency_code_check CHECK (currency_code ~ '^[A-Z]{3}$'),
  CONSTRAINT campaigns_date_range_check CHECK (end_date > start_date)
);

CREATE TABLE campaign_platforms (
  campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  platform text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (campaign_id, platform),
  CONSTRAINT campaign_platforms_platform_check CHECK (platform IN ('TIKTOK', 'INSTAGRAM', 'YOUTUBE'))
);

CREATE TABLE campaign_requirements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  requirement_type text NOT NULL,
  description text NOT NULL,
  position smallint NOT NULL DEFAULT 1,
  is_required boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT campaign_requirements_type_check CHECK (requirement_type IN ('CONTENT', 'HASHTAG', 'MENTION', 'DISCLOSURE', 'OTHER')),
  CONSTRAINT campaign_requirements_description_not_blank_check CHECK (btrim(description) <> ''),
  CONSTRAINT campaign_requirements_position_check CHECK (position > 0),
  CONSTRAINT campaign_requirements_campaign_position_key UNIQUE (campaign_id, position)
);

CREATE TABLE campaign_participants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  clipper_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status text NOT NULL DEFAULT 'PENDING',
  joined_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT campaign_participants_status_check CHECK (status IN ('PENDING', 'ACCEPTED', 'DECLINED', 'REMOVED')),
  CONSTRAINT campaign_participants_campaign_clipper_key UNIQUE (campaign_id, clipper_id),
  CONSTRAINT campaign_participants_id_campaign_key UNIQUE (id, campaign_id)
);

CREATE TABLE clip_submissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE RESTRICT,
  participant_id uuid NOT NULL,
  content_url text NOT NULL UNIQUE,
  platform text NOT NULL,
  status text NOT NULL DEFAULT 'PENDING',
  submitted_at timestamptz NOT NULL DEFAULT NOW(),
  reviewed_at timestamptz,
  reviewed_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  review_note text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT clip_submissions_content_url_not_blank_check CHECK (btrim(content_url) <> ''),
  CONSTRAINT clip_submissions_content_url_normalized_check CHECK (content_url = btrim(content_url)),
  CONSTRAINT clip_submissions_platform_check CHECK (platform IN ('TIKTOK', 'INSTAGRAM', 'YOUTUBE')),
  CONSTRAINT clip_submissions_status_check CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'FLAGGED')),
  CONSTRAINT clip_submissions_review_timestamp_check CHECK (reviewed_at IS NULL OR reviewed_at >= submitted_at),
  CONSTRAINT clip_submissions_participant_campaign_fk FOREIGN KEY (participant_id, campaign_id)
    REFERENCES campaign_participants(id, campaign_id) ON DELETE RESTRICT,
  CONSTRAINT clip_submissions_campaign_platform_fk FOREIGN KEY (campaign_id, platform)
    REFERENCES campaign_platforms(campaign_id, platform) ON DELETE RESTRICT,
  CONSTRAINT clip_submissions_id_campaign_key UNIQUE (id, campaign_id),
  CONSTRAINT clip_submissions_id_campaign_participant_key UNIQUE (id, campaign_id, participant_id)
);

CREATE INDEX campaigns_brand_status_dates_idx ON campaigns (brand_id, status, start_date DESC);
CREATE INDEX campaigns_status_dates_idx ON campaigns (status, start_date DESC);
CREATE INDEX campaign_participants_clipper_status_idx ON campaign_participants (clipper_id, status, created_at DESC);
CREATE INDEX clip_submissions_campaign_status_submitted_idx ON clip_submissions (campaign_id, status, submitted_at DESC);
CREATE INDEX clip_submissions_participant_submitted_idx ON clip_submissions (participant_id, submitted_at DESC);

CREATE TRIGGER campaigns_set_updated_at
BEFORE UPDATE ON campaigns
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER campaign_requirements_set_updated_at
BEFORE UPDATE ON campaign_requirements
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER campaign_participants_set_updated_at
BEFORE UPDATE ON campaign_participants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER clip_submissions_set_updated_at
BEFORE UPDATE ON clip_submissions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
