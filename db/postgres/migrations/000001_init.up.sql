create table reports (id text primary key, name text not null, labels json, created_by text not null, created_at timestamptz, updated_by text not null, updated_at timestamptz);
