CREATE TABLE signing_keys (
    id text PRIMARY KEY,
    private_key_pem text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
