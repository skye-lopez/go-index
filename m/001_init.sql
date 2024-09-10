create table packages (
    url text primary key,
    host text not null,
    path text not null, 
    owner text
);

create index trgm_idx on packages using gin (url gin_tgrm_ops);

create table package_version (
    version text primary key,
    owner text references packages(url),
    update_time text not null
);

create table log (
    id text primary key,
    last_write text not null
);
