-- +goose Up
create table channels (
    id bigint primary key,
    title text not null,
    type char not null,
    subscribers_counter int not null
);

create table users (id bigint primary key, name text not null);

create table posts (
    id serial primary key,
    channel_id bigint not null,
    post_id_into_channel bigint not null,
    timestamp time not null,
    value text
);

create table comments (
    id bigint primary key,
    post_id bigint references posts (id),
    replied_to bigint not null,
    user_id bigint not null,
    timestamp time not null,
    value text
);

-- +goose Down
drop table auth;
