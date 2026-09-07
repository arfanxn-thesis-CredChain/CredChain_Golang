-- Drop join table first (FKs to competencies + credentials)
DROP TABLE IF EXISTS competency_credential;

-- Drop credentials (FKs reference users, credential_issuer_organizations, credential_types)
DROP INDEX IF EXISTS idx_credentials_file_hash_active;
DROP INDEX IF EXISTS uq_credentials_issuer_org_number;
DROP INDEX IF EXISTS idx_credentials_file_hash;
DROP INDEX IF EXISTS idx_credentials_extract_enqueued_at;
DROP INDEX IF EXISTS idx_credentials_extract_failed_at;
DROP INDEX IF EXISTS idx_credentials_revoked_at;
DROP INDEX IF EXISTS idx_credentials_expires_at;
DROP INDEX IF EXISTS idx_credentials_type_id;
DROP INDEX IF EXISTS idx_credentials_issuer_user_id;
DROP INDEX IF EXISTS idx_credentials_holder_user_id;
DROP TABLE IF EXISTS credentials;

-- Drop credential taxonomy tables
DROP TABLE IF EXISTS competencies;
DROP TABLE IF EXISTS credential_issuer_organizations;
DROP TABLE IF EXISTS credential_types;

-- Drop user_tokens before users (FK references users)
DROP INDEX IF EXISTS idx_user_tokens_token;
DROP INDEX IF EXISTS idx_user_tokens_user_id;
DROP TABLE IF EXISTS user_tokens;
DROP TYPE IF EXISTS user_token_type;

-- Drop users before user_units (FK fk_users_unit_id references user_units)
DROP INDEX IF EXISTS idx_users_joined_year;
DROP INDEX IF EXISTS idx_users_unit_id;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP TABLE IF EXISTS users;

-- Drop user_units last (self-referential)
DROP INDEX IF EXISTS idx_user_units_parent_id;
DROP TABLE IF EXISTS user_units;

DROP TYPE IF EXISTS role;
DROP TYPE IF EXISTS gender;

DROP EXTENSION IF EXISTS pg_trgm;
