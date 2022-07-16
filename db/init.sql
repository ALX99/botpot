CREATE TABLE IP (
    ip_address inet NOT NULL,
    PRIMARY KEY (ip_address)
);
CREATE TABLE Session (
    id serial NOT NULL,
    version text NOT NULL,
    stdout text NOT NULL, -- Related to script
    timing text NOT NULL, -- Related to script
    src_ip inet NOT NULL,
    src_port int NOT NULL,
    dst_ip inet NOT NULL,
    dst_port int NOT NULL,
    start_ts timestamptz NOT NULL,
    end_ts timestamptz NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_ip FOREIGN KEY (src_ip) REFERENCES IP (ip_address),
    CONSTRAINT valid_port CHECK (
        src_port BETWEEN 0 AND 65535
        AND dst_port BETWEEN 0 AND 65535
    ),
    CONSTRAINT end_time_after_start_time CHECK (start_ts <= end_ts)
);
CREATE TABLE Channel (
    id int NOT NULL,
    session_id int NOT NULL,
    channel_type text NOT NULL,
    start_ts timestamptz NOT NULL,
    end_ts timestamptz NOT NULL,
    PRIMARY KEY (id, session_id),
    CONSTRAINT fk_id FOREIGN KEY (session_id) REFERENCES Session (id),
    CONSTRAINT end_time_after_start_time CHECK (start_ts <= end_ts)
);
CREATE TABLE PTYRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    term text NOT NULL,
    columns int NOT NULL,
    rows int NOT NULL,
    width int NOT NULL,
    height int NOT NULL,
    modelist bytea NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE ExecRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    command text NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE ExitStatusRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    exit_status int NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE ShellRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE WindowDimensionChangeRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    columns int NOT NULL,
    rows int NOT NULL,
    width int NOT NULL,
    height int NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE EnvironmentRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    name text NOT NULL,
    value text NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);
CREATE TABLE SubSystemRequest (
    id serial NOT NULL,
    session_id int NOT NULL,
    channel_id int NOT NULL,
    ts timestamptz NOT NULL,
    from_client boolean NOT NULL,
    name text NOT NULL,
    PRIMARY KEY (session_id, channel_id, id),
    CONSTRAINT fk_id FOREIGN KEY (session_id, channel_id) REFERENCES Channel (session_id, id)
);