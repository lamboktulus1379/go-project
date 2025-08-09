--liquibase formatted sql

--changeset lamboktulus1379:1 labels:my_project-label context:my_project-context
--preconditions onFail:WARN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'user';
--comment: my_project comment
create table public.user (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name varchar(50) not null,
    user_name varchar(50),
    password varchar(50),
    created_by varchar(50),
    updated_by varchar(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
--rollback DROP TABLE public.user;

--changeset lamboktulus1379:2 labels:initialize context:development
--preconditions onFail:WARN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'user';
--comment: creating project table
CREATE TABLE public.project
(
    id serial PRIMARY KEY,
    name character varying NOT NULL,
    description character varying,
    created_at time with time zone DEFAULT NOW(),
    updated_at time with time zone DEFAULT CURRENT_TIMESTAMP
);
--rollback DROP TABLE public.project;

--changeset lamboktulus1379:3 labels:share-feature context:share
--comment: create share tracking table (video_share_records) for social platforms
CREATE TABLE IF NOT EXISTS public.video_share_records (
    id BIGSERIAL PRIMARY KEY,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | success | failed
    error_message TEXT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_video_platform_user UNIQUE (video_id, platform, user_id)
);
--rollback DROP TABLE IF EXISTS public.video_share_records;

--changeset lamboktulus1379:4 labels:share-feature context:share
--comment: append-only audit log of share attempts
CREATE TABLE IF NOT EXISTS public.video_share_audit (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES public.video_share_records(id) ON DELETE CASCADE,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
--rollback DROP TABLE IF EXISTS public.video_share_audit;

--changeset lamboktulus1379:5 labels:share-feature context:share
--comment: job queue for server_post processing
CREATE TABLE IF NOT EXISTS public.share_jobs (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES public.video_share_records(id) ON DELETE CASCADE,
    platform VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | running | success | failed
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
--rollback DROP TABLE IF EXISTS public.share_jobs;

--changeset lamboktulus1379:6 labels:share-feature context:share
--comment: oauth tokens per user/platform for future automatic posting
CREATE TABLE IF NOT EXISTS public.oauth_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NULL,
    expires_at TIMESTAMPTZ NULL DEFAULT NULL,
    scopes TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_platform UNIQUE (user_id, platform)
);
--rollback DROP TABLE IF EXISTS public.oauth_tokens;

--changeset lamboktulus1379:7 labels:share-feature context:share
--comment: add page and token type fields for facebook page posting
ALTER TABLE public.oauth_tokens
    ADD COLUMN IF NOT EXISTS page_id VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS page_name VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS token_type VARCHAR(32) NULL; -- user | page
--rollback ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS token_type; ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS page_name; ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS page_id;
