create table if not exists ads (
    id bigserial primary key,
    created_at timestamp(0) not null default now(),
    title text not null,
    description text not null,
    price integer not null,
    categories text[] not null,
    version integer not null default 1
);