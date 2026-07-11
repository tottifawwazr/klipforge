DROP INDEX IF EXISTS refresh_tokens_family_active_idx;
DROP INDEX IF EXISTS refresh_tokens_session_idx;

ALTER TABLE refresh_tokens
  DROP CONSTRAINT IF EXISTS refresh_tokens_revocation_reason_check,
  DROP CONSTRAINT IF EXISTS refresh_tokens_rotation_state_check,
  DROP CONSTRAINT IF EXISTS refresh_tokens_issued_expiry_check,
  DROP COLUMN IF EXISTS ip_address,
  DROP COLUMN IF EXISTS user_agent,
  DROP COLUMN IF EXISTS revocation_reason,
  DROP COLUMN IF EXISTS rotated_at,
  DROP COLUMN IF EXISTS issued_at,
  DROP COLUMN IF EXISTS session_id;
