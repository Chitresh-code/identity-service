ALTER TABLE users ADD COLUMN is_admin boolean NOT NULL DEFAULT false;

-- Bootstrap: if any users already exist from before this migration (i.e. this
-- isn't a fresh database), promote the earliest-created one to admin, since
-- otherwise no one could ever reach the now admin-gated /admin/* routes. Fresh
-- databases don't need this -- UserStore.UpsertByAuth0Sub grants admin to
-- whichever user logs in first once the users table is empty.
UPDATE users SET is_admin = true
WHERE id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1);

CREATE TABLE applications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id uuid NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
    prefix text NOT NULL UNIQUE,
    secret_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
