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
