CREATE ROLE readaccess;

GRANT USAGE ON SCHEMA public TO readaccess;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readaccess;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO readaccess;

CREATE USER grafana WITH PASSWORD 'example';
GRANT readaccess TO grafana;

CREATE TABLE IP (
    ip_address inet NOT NULL,
    PRIMARY KEY (ip_address)
);
CREATE TABLE Session (
    id SERIAL NOT NULL,
    version TEXT NOT NULL,
    stdout TEXT NOT NULL, -- Related to script
    timing TEXT NOT NULL, -- Related to script
    src_ip inet NOT NULL,
    src_port INT NOT NULL,
    dst_ip inet NOT NULL,
    dst_port INT NOT NULL,
    start_ts timestamptz NOT NULL,
    end_ts timestamptz NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_ip FOREIGN KEY (src_ip) REFERENCES IP (ip_address) ON DELETE CASCADE,
    CONSTRAINT valid_port CHECK (
        src_port BETWEEN 0 AND 65535
        AND dst_port BETWEEN 0 AND 65535
    ),
    CONSTRAINT end_time_after_start_time CHECK (start_ts <= end_ts)
);
CREATE TABLE Channel (
    id INT NOT NULL,
    session_id INT NOT NULL,
    channel_type TEXT NOT NULL,
    recv BYTEA,
    recv_stderr BYTEA,
    start_ts timestamptz NOT NULL,
    end_ts timestamptz NOT NULL,
    PRIMARY KEY (id, session_id),
    CONSTRAINT fk_id FOREIGN KEY (session_id) REFERENCES Session (id) ON DELETE CASCADE,
    CONSTRAINT end_time_after_start_time CHECK (start_ts <= end_ts)
);
CREATE TABLE Request (
    id SERIAL NOT NULL,
    channel_id INT NOT NULL,
    session_id INT NOT NULL,
    ts timestamptz NOT NULL,
    from_client BOOLEAN NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_id FOREIGN KEY (channel_id, session_id) REFERENCES Channel (id, session_id) ON DELETE CASCADE
);
CREATE TABLE PTYRequest (
    request_id INT NOT NULL,
    term TEXT NOT NULL,
    columns INT NOT NULL,
    rows INT NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    modelist BYTEA NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE ExecRequest (
    request_id INT NOT NULL,
    command TEXT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE ExitStatusRequest (
    request_id INT NOT NULL,
    exit_status INT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE ExitSignalRequest (
    request_id INT NOT NULL,
    signal_name TEXT NOT NULL,
    core_dumped BOOLEAN NOT NULL,
    error_msg TEXT NOT NULL,
    language_tag TEXT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE ShellRequest (
    request_id INT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE WindowDimChangeRequest (
    request_id INT NOT NULL,
    columns INT NOT NULL,
    rows INT NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE EnvironmentRequest (
    request_id INT NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
CREATE TABLE SubSystemRequest (
    request_id INT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (request_id),
    CONSTRAINT fk_id FOREIGN KEY (request_id) REFERENCES Request (id) ON DELETE CASCADE
);
