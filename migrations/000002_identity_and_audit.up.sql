CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  role text NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_normalized_check CHECK (email = lower(btrim(email))),
  CONSTRAINT users_email_format_check CHECK (position('@' IN email) > 1),
  CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'BRAND', 'CLIPPER'))
);

CREATE TABLE profiles (
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  display_name text NOT NULL,
  avatar_url text,
  bio text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT profiles_display_name_not_blank_check CHECK (btrim(display_name) <> ''),
  CONSTRAINT profiles_bio_length_check CHECK (bio IS NULL OR char_length(bio) <= 500)
);

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  family_id uuid NOT NULL DEFAULT gen_random_uuid(),
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  replaced_by_token_id uuid REFERENCES refresh_tokens(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT refresh_tokens_hash_not_blank_check CHECK (btrim(token_hash) <> ''),
  CONSTRAINT refresh_tokens_expiry_check CHECK (expires_at > created_at)
);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  actor_email text NOT NULL,
  actor_role text NOT NULL,
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id uuid,
  request_id text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  ip_address inet,
  user_agent text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  CONSTRAINT audit_logs_actor_email_not_blank_check CHECK (btrim(actor_email) <> ''),
  CONSTRAINT audit_logs_action_not_blank_check CHECK (btrim(action) <> ''),
  CONSTRAINT audit_logs_entity_type_not_blank_check CHECK (btrim(entity_type) <> ''),
  CONSTRAINT audit_logs_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX users_role_active_idx ON users (role) WHERE is_active;
CREATE INDEX refresh_tokens_user_active_idx ON refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);
CREATE INDEX audit_logs_entity_created_at_idx ON audit_logs (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_logs_actor_created_at_idx ON audit_logs (actor_user_id, created_at DESC);
CREATE INDEX audit_logs_action_created_at_idx ON audit_logs (action, created_at DESC);

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER profiles_set_updated_at
BEFORE UPDATE ON profiles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER refresh_tokens_set_updated_at
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
