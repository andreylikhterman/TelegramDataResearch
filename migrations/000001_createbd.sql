-- +goose Up
create table channels (
                      id text primary key,
                      title text not null,
                      type char not null,
                      subscribers_counter int not null
);

create table users (
                          id text primary key,
                          name text not null
);

create table posts (
                       id text primary key,
                       channel_id text not null references channels(id),
                       post_id_into_channel text not null,
                       timestamp time not null,
                       value text
);

create table comments    (
                       id text primary key,
                       post_id text not null references posts(id),
                       replied_to text not null,
                       user_id text not null,
                       timestamp time not null,
                       value text
);

-- +goose Down
drop table auth;