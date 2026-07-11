-- No tables yet (#2 is scaffold-only). Enables pgcrypto now since every future
-- table (#4's applications, users) will use gen_random_uuid() for primary keys.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
