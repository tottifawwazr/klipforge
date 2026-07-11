CREATE TABLE metric_snapshots (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id uuid NOT NULL REFERENCES clip_submissions(id) ON DELETE CASCADE,
  captured_at timestamptz NOT NULL,
  views_count bigint NOT NULL DEFAULT 0,
  likes_count bigint NOT NULL DEFAULT 0,
  comments_count bigint NOT NULL DEFAULT 0,
  shares_count bigint NOT NULL DEFAULT 0,
  saves_count bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT metric_snapshots_submission_captured_key UNIQUE (submission_id, captured_at),
  CONSTRAINT metric_snapshots_values_non_negative_check CHECK (
    views_count >= 0
    AND likes_count >= 0
    AND comments_count >= 0
    AND shares_count >= 0
    AND saves_count >= 0
  )
);

CREATE TABLE idempotency_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  scope text NOT NULL,
  idempotency_key text NOT NULL,
  request_hash text NOT NULL,
  status text NOT NULL DEFAULT 'IN_PROGRESS',
  response_code integer,
  response_body jsonb,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT idempotency_keys_scope_not_blank_check CHECK (btrim(scope) <> ''),
  CONSTRAINT idempotency_keys_key_not_blank_check CHECK (btrim(idempotency_key) <> ''),
  CONSTRAINT idempotency_keys_request_hash_not_blank_check CHECK (btrim(request_hash) <> ''),
  CONSTRAINT idempotency_keys_status_check CHECK (status IN ('IN_PROGRESS', 'COMPLETED', 'FAILED')),
  CONSTRAINT idempotency_keys_response_body_object_check CHECK (response_body IS NULL OR jsonb_typeof(response_body) = 'object'),
  CONSTRAINT idempotency_keys_expiry_check CHECK (expires_at > created_at),
  CONSTRAINT idempotency_keys_actor_scope_key UNIQUE (actor_user_id, scope, idempotency_key)
);

CREATE TABLE payouts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id uuid NOT NULL REFERENCES campaigns(id) ON DELETE RESTRICT,
  submission_id uuid NOT NULL,
  participant_id uuid NOT NULL,
  idempotency_key_id uuid NOT NULL UNIQUE REFERENCES idempotency_keys(id) ON DELETE RESTRICT,
  status text NOT NULL DEFAULT 'PENDING',
  amount numeric(14, 2) NOT NULL,
  currency_code text NOT NULL DEFAULT 'USD',
  processor_reference text UNIQUE,
  approved_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  approved_at timestamptz,
  processed_at timestamptz,
  failure_reason text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT payouts_status_check CHECK (status IN ('PENDING', 'APPROVED', 'PROCESSED', 'FAILED')),
  CONSTRAINT payouts_amount_non_negative_check CHECK (amount >= 0),
  CONSTRAINT payouts_currency_code_check CHECK (currency_code ~ '^[A-Z]{3}$'),
  CONSTRAINT payouts_approval_timestamp_check CHECK (approved_at IS NULL OR approved_at >= created_at),
  CONSTRAINT payouts_processed_timestamp_check CHECK (processed_at IS NULL OR processed_at >= created_at),
  CONSTRAINT payouts_submission_once_key UNIQUE (submission_id),
  CONSTRAINT payouts_submission_campaign_participant_fk FOREIGN KEY (submission_id, campaign_id, participant_id)
    REFERENCES clip_submissions(id, campaign_id, participant_id) ON DELETE RESTRICT
);

CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  notification_type text NOT NULL,
  title text NOT NULL,
  body text NOT NULL,
  data jsonb NOT NULL DEFAULT '{}'::jsonb,
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT notifications_type_not_blank_check CHECK (btrim(notification_type) <> ''),
  CONSTRAINT notifications_title_not_blank_check CHECK (btrim(title) <> ''),
  CONSTRAINT notifications_body_not_blank_check CHECK (btrim(body) <> ''),
  CONSTRAINT notifications_data_object_check CHECK (jsonb_typeof(data) = 'object'),
  CONSTRAINT notifications_read_timestamp_check CHECK (read_at IS NULL OR read_at >= created_at)
);

CREATE INDEX metric_snapshots_submission_captured_desc_idx ON metric_snapshots (submission_id, captured_at DESC);
CREATE INDEX idempotency_keys_expiry_idx ON idempotency_keys (expires_at);
CREATE INDEX payouts_campaign_status_idx ON payouts (campaign_id, status, created_at DESC);
CREATE INDEX payouts_participant_status_idx ON payouts (participant_id, status, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);

CREATE TRIGGER idempotency_keys_set_updated_at
BEFORE UPDATE ON idempotency_keys
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER payouts_set_updated_at
BEFORE UPDATE ON payouts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER notifications_set_updated_at
BEFORE UPDATE ON notifications
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
