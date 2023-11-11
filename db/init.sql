CREATE ROLE readaccess;

GRANT USAGE ON SCHEMA public TO readaccess;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readaccess;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO readaccess;

CREATE USER grafana WITH PASSWORD 'example';
GRANT readaccess TO grafana;

\i schema.sql
