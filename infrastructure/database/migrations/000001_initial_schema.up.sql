-- Trigram similarity backs the reviewer's fuzzy-match suggestions when
-- resolving a submitted free-text metadata name against the taxonomy.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE role AS ENUM (
    'super_admin',
    'admin',
    'issuer',
    'holder'
);

CREATE TYPE gender AS ENUM (
    'male',
    'female'
);

-- Organizational units (faculty / study program / department ...), self-referential tree
CREATE TABLE user_units (
    id CHAR(26) PRIMARY KEY,
    parent_id CHAR(26),
    name VARCHAR(256) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_user_units_parent_id FOREIGN KEY (parent_id) REFERENCES user_units(id)
);

CREATE INDEX idx_user_units_parent_id ON user_units(parent_id);

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY,
    unit_id CHAR(26),
    name VARCHAR(256),
    number VARCHAR(256) UNIQUE,
    email VARCHAR(256) UNIQUE NOT NULL,
    language VARCHAR(2) NOT NULL DEFAULT 'id' CHECK (language IN ('id', 'en')),
    gender gender,
    birth_date DATE,
    meta JSONB,
    role role NOT NULL,
    wallet_address CHAR(42) UNIQUE NOT NULL,
    encrypted_wallet_private_key VARCHAR(256) NOT NULL,
    joined_year INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_users_unit_id FOREIGN KEY (unit_id) REFERENCES user_units(id)
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_unit_id ON users(unit_id);
CREATE INDEX idx_users_joined_year ON users(joined_year);

CREATE TABLE credential_types (
    id CHAR(26) PRIMARY KEY,
    submitter_user_id CHAR(26),
    approver_user_id CHAR(26),
    rejecter_user_id CHAR(26),
    name VARCHAR(256) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    rejection_reason TEXT,
    activated_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_credential_types_submitter_user_id FOREIGN KEY (submitter_user_id) REFERENCES users(id),
    CONSTRAINT fk_credential_types_approver_user_id FOREIGN KEY (approver_user_id) REFERENCES users(id),
    CONSTRAINT fk_credential_types_rejecter_user_id FOREIGN KEY (rejecter_user_id) REFERENCES users(id),
    CONSTRAINT chk_credential_types_activated_xor_rejected CHECK (activated_at IS NULL OR rejected_at IS NULL)
);

CREATE TABLE credential_issuer_organizations (
    id CHAR(26) PRIMARY KEY,
    submitter_user_id CHAR(26),
    approver_user_id CHAR(26),
    rejecter_user_id CHAR(26),
    name VARCHAR(256) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    rejection_reason TEXT,
    activated_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_credential_issuer_organizations_submitter_user_id FOREIGN KEY (submitter_user_id) REFERENCES users(id),
    CONSTRAINT fk_credential_issuer_organizations_approver_user_id FOREIGN KEY (approver_user_id) REFERENCES users(id),
    CONSTRAINT fk_credential_issuer_organizations_rejecter_user_id FOREIGN KEY (rejecter_user_id) REFERENCES users(id),
    CONSTRAINT chk_credential_issuer_organizations_activated_xor_rejected CHECK (activated_at IS NULL OR rejected_at IS NULL)
);

CREATE TABLE competencies (
    id CHAR(26) PRIMARY KEY,
    submitter_user_id CHAR(26),
    approver_user_id CHAR(26),
    rejecter_user_id CHAR(26),
    name VARCHAR(256) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    rejection_reason TEXT,
    activated_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_competencies_submitter_user_id FOREIGN KEY (submitter_user_id) REFERENCES users(id),
    CONSTRAINT fk_competencies_approver_user_id FOREIGN KEY (approver_user_id) REFERENCES users(id),
    CONSTRAINT fk_competencies_rejecter_user_id FOREIGN KEY (rejecter_user_id) REFERENCES users(id),
    CONSTRAINT chk_competencies_activated_xor_rejected CHECK (activated_at IS NULL OR rejected_at IS NULL)
);

-- Case-insensitive uniqueness race backstop for unrejected reference names.
-- A rejected name does not permanently block future proposals.
CREATE UNIQUE INDEX uq_credential_types_lower_name
    ON credential_types (LOWER(name)) WHERE rejected_at IS NULL;
CREATE UNIQUE INDEX uq_credential_issuer_organizations_lower_name
    ON credential_issuer_organizations (LOWER(name)) WHERE rejected_at IS NULL;
CREATE UNIQUE INDEX uq_competencies_lower_name
    ON competencies (LOWER(name)) WHERE rejected_at IS NULL;

CREATE INDEX idx_credential_types_name_trgm
    ON credential_types USING GIN (name gin_trgm_ops);
CREATE INDEX idx_credential_issuer_organizations_name_trgm
    ON credential_issuer_organizations USING GIN (name gin_trgm_ops);
CREATE INDEX idx_competencies_name_trgm
    ON competencies USING GIN (name gin_trgm_ops);

CREATE INDEX idx_credential_types_submitter ON credential_types(submitter_user_id);
CREATE INDEX idx_credential_types_activated_at ON credential_types(activated_at);
CREATE INDEX idx_credential_issuer_organizations_submitter ON credential_issuer_organizations(submitter_user_id);
CREATE INDEX idx_credential_issuer_organizations_activated_at ON credential_issuer_organizations(activated_at);
CREATE INDEX idx_competencies_submitter ON competencies(submitter_user_id);
CREATE INDEX idx_competencies_activated_at ON competencies(activated_at);

CREATE TABLE credentials (
    id CHAR(26) PRIMARY KEY,
    holder_user_id CHAR(26) NOT NULL,
    submitter_user_id CHAR(26) NOT NULL,
    issuer_user_id CHAR(26),
    issuer_organization_id CHAR(26) NOT NULL,
    type_id CHAR(26) NOT NULL,
    number VARCHAR(256),
    name VARCHAR(256) NOT NULL,
    meta JSONB,
    token_id VARCHAR(256) UNIQUE,
    file_hash CHAR(66) NOT NULL,
    file_uri TEXT,
    extract_enqueued_at TIMESTAMP WITH TIME ZONE,
    extract_failed_at TIMESTAMP WITH TIME ZONE,
    extract_error TEXT,
    rejecter_user_id CHAR(26),
    revoker_user_id CHAR(26),
    rejection_reason TEXT,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    activated_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revocation_reason TEXT,
    extracted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_holder_user_id FOREIGN KEY (holder_user_id) REFERENCES users(id),
    CONSTRAINT fk_submitter_user_id FOREIGN KEY (submitter_user_id) REFERENCES users(id),
    CONSTRAINT fk_issuer_user_id FOREIGN KEY (issuer_user_id) REFERENCES users(id),
    CONSTRAINT fk_issuer_organization_id FOREIGN KEY (issuer_organization_id) REFERENCES credential_issuer_organizations(id),
    CONSTRAINT fk_type_id FOREIGN KEY (type_id) REFERENCES credential_types(id),
    CONSTRAINT fk_rejecter_user_id FOREIGN KEY (rejecter_user_id) REFERENCES users(id),
    CONSTRAINT fk_revoker_user_id FOREIGN KEY (revoker_user_id) REFERENCES users(id),
    CONSTRAINT chk_credentials_activated_xor_rejected CHECK (activated_at IS NULL OR rejected_at IS NULL)
);

CREATE INDEX idx_credentials_holder_user_id ON credentials(holder_user_id);
CREATE INDEX idx_credentials_issuer_user_id ON credentials(issuer_user_id);
CREATE INDEX idx_credentials_type_id        ON credentials(type_id);
CREATE INDEX idx_credentials_expires_at     ON credentials(expires_at);
CREATE INDEX idx_credentials_revoked_at     ON credentials(revoked_at);
CREATE INDEX idx_credentials_extract_enqueued_at ON credentials(extract_enqueued_at) WHERE extract_enqueued_at IS NOT NULL;
CREATE INDEX idx_credentials_extract_failed_at   ON credentials(extract_failed_at)   WHERE extract_failed_at IS NOT NULL;
CREATE INDEX idx_credentials_file_hash      ON credentials(file_hash);
CREATE UNIQUE INDEX idx_credentials_file_hash_active ON credentials(file_hash) WHERE revoked_at IS NULL AND rejected_at IS NULL;
CREATE UNIQUE INDEX uq_credentials_issuer_org_number
    ON credentials (issuer_organization_id, number);

-- Many-to-many: which competencies a credential attests
CREATE TABLE competency_credential (
    competency_id CHAR(26) NOT NULL,
    credential_id CHAR(26) NOT NULL,
    PRIMARY KEY (competency_id, credential_id),
    CONSTRAINT fk_competency_credential_competency_id FOREIGN KEY (competency_id) REFERENCES competencies(id),
    CONSTRAINT fk_competency_credential_credential_id FOREIGN KEY (credential_id) REFERENCES credentials(id)
);

-- Token types for user refresh tokens
CREATE TYPE user_token_type AS ENUM ('refresh');

-- User tokens table for managing refresh tokens
CREATE TABLE user_tokens (
    id CHAR(26) PRIMARY KEY,
    user_id CHAR(26) NOT NULL,
    type user_token_type NOT NULL,
    token VARCHAR(512) NOT NULL UNIQUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Index for fast user-based lookups
CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);

-- Index for token lookups (refresh token validation)
CREATE INDEX idx_user_tokens_token ON user_tokens(token);

CREATE TABLE notifications (
    id                  VARCHAR(26) PRIMARY KEY,
    notified_user_id     CHAR(26) NOT NULL REFERENCES users(id),
    subject_type        TEXT,
    subject_id          CHAR(26),
    type                TEXT NOT NULL,
    data                JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ,
    read_at             TIMESTAMPTZ,
    CONSTRAINT notifications_type_check CHECK (
        type IN ('credential_registered', 'credential_approved', 'credential_rejected', 'credential_revoked',
                 'credential_type_submitted', 'credential_issuer_organization_submitted', 'competency_submitted')
    ),
    CONSTRAINT notifications_subject_check CHECK (
        (subject_type IS NULL AND subject_id IS NULL) OR
        (subject_type IS NOT NULL AND subject_id IS NOT NULL AND (
            (subject_type = 'credential' AND type IN ('credential_registered', 'credential_approved', 'credential_rejected', 'credential_revoked')) OR
            (subject_type = 'credential_type' AND type = 'credential_type_submitted') OR
            (subject_type = 'credential_issuer_organization' AND type = 'credential_issuer_organization_submitted') OR
            (subject_type = 'competency' AND type = 'competency_submitted')
        ))
    ),
    CONSTRAINT notifications_user_subject_type_unique
        UNIQUE (notified_user_id, subject_type, subject_id, type)
);

CREATE INDEX notifications_user_feed_idx
    ON notifications (notified_user_id, created_at DESC, id DESC);

CREATE INDEX notifications_user_unread_idx
    ON notifications (notified_user_id)
    WHERE read_at IS NULL;
