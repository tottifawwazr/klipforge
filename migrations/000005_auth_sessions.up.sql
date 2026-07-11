ALTER TABLE refresh_tokens
  ADD COLUMN session_id uuid,
  ADD COLUMN issued_at timestamptz,
  ADD COLUMN rotated_at timestamptz,
  ADD COLUMN revocation_reason text,
  ADD COLUMN user_agent text,
  ADD COLUMN ip_address inet;

UPDATE refresh_tokens
SET session_id = family_id,
    issued_at = created_at,
    rotated_at = CASE
      WHEN replaced_by_token_id IS NOT NULL THEN COALESCE(revoked_at, updated_at)
      ELSE NULL
    END;

ALTER TABLE refresh_tokens
  ALTER COLUMN session_id SET NOT NULL,
  ALTER COLUMN issued_at SET NOT NULL,
  ALTER COLUMN issued_at SET DEFAULT NOW(),
  ADD CONSTRAINT refresh_tokens_issued_expiry_check CHECK (expires_at > issued_at),
  ADD CONSTRAINT refresh_tokens_rotation_state_check CHECK (
    (rotated_at IS NULL AND replaced_by_token_id IS NULL)
    OR (rotated_at IS NOT NULL AND replaced_by_token_id IS NOT NULL)
  ),
  ADD CONSTRAINT refresh_tokens_revocation_reason_check CHECK (
    revocation_reason IS NULL OR btrim(revocation_reason) <> ''
  );

CREATE INDEX refresh_tokens_session_idx ON refresh_tokens (session_id, issued_at DESC);
CREATE INDEX refresh_tokens_family_active_idx
  ON refresh_tokens (family_id, expires_at)
  WHERE revoked_at IS NULL;
