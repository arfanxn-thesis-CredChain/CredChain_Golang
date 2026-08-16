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
    name VARCHAR(256) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE credential_issuer_organizations (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(256) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE competencies (
    id CHAR(26) PRIMARY KEY,
    name VARCHAR(256) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE
);

CREATE TYPE credential_extract_status AS ENUM (
    'pending',
    'succeeded',
    'failed'
);

CREATE TABLE credentials (
    id CHAR(26) PRIMARY KEY,
    holder_user_id CHAR(26) NOT NULL,
    submitter_user_id CHAR(26) NOT NULL,
    issuer_user_id CHAR(26) NOT NULL,
    issuer_organization_id CHAR(26) NOT NULL,
    type_id CHAR(26) NOT NULL,
    number VARCHAR(256),
    name VARCHAR(256) NOT NULL,
    meta JSONB,
    token_id VARCHAR(256) UNIQUE,
    file_hash CHAR(66) NOT NULL,
    file_uri TEXT,
    extract_status credential_extract_status NOT NULL DEFAULT 'pending',
    extract_error TEXT,
    approver_user_id CHAR(26),
    rejecter_user_id CHAR(26),
    revoker_user_id CHAR(26),
    rejection_reason TEXT,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    extracted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_holder_user_id FOREIGN KEY (holder_user_id) REFERENCES users(id),
    CONSTRAINT fk_submitter_user_id FOREIGN KEY (submitter_user_id) REFERENCES users(id),
    CONSTRAINT fk_issuer_user_id FOREIGN KEY (issuer_user_id) REFERENCES users(id),
    CONSTRAINT fk_issuer_organization_id FOREIGN KEY (issuer_organization_id) REFERENCES credential_issuer_organizations(id),
    CONSTRAINT fk_type_id FOREIGN KEY (type_id) REFERENCES credential_types(id),
    CONSTRAINT fk_approver_user_id FOREIGN KEY (approver_user_id) REFERENCES users(id),
    CONSTRAINT fk_rejecter_user_id FOREIGN KEY (rejecter_user_id) REFERENCES users(id),
    CONSTRAINT fk_revoker_user_id FOREIGN KEY (revoker_user_id) REFERENCES users(id),
    CONSTRAINT chk_credentials_approved_xor_rejected CHECK (approved_at IS NULL OR rejected_at IS NULL),
    CONSTRAINT uq_credentials_issuer_org_number UNIQUE (issuer_organization_id, number)
);

CREATE INDEX idx_credentials_holder_user_id ON credentials(holder_user_id);
CREATE INDEX idx_credentials_issuer_user_id ON credentials(issuer_user_id);
CREATE INDEX idx_credentials_type_id        ON credentials(type_id);
CREATE INDEX idx_credentials_expires_at     ON credentials(expires_at);
CREATE INDEX idx_credentials_revoked_at     ON credentials(revoked_at);
CREATE INDEX idx_credentials_extract_status ON credentials(extract_status);
CREATE INDEX idx_credentials_file_hash      ON credentials(file_hash);
CREATE UNIQUE INDEX idx_credentials_file_hash_active ON credentials(file_hash) WHERE revoked_at IS NULL AND rejected_at IS NULL;

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
