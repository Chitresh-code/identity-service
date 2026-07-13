ALTER TABLE users ADD COLUMN role text CHECK (role IN ('member', 'lead'));

CREATE TABLE login_handoff_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_handoff_codes_expires_at_idx ON login_handoff_codes (expires_at);

CREATE TABLE user_refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX user_refresh_tokens_expires_at_idx ON user_refresh_tokens (expires_at);
