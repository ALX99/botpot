CREATE TABLE IP (
    ip_address inet NOT NULL,
    PRIMARY KEY (ip_address)
);
CREATE TABLE Session (
    id serial NOT NULL,
    src_ip inet NOT NULL,
    src_port int NOT NULL,
    dst_ip inet NOT NULL,
    dst_port int NOT NULL,
    start_timestamp timestamp NOT NULL,
    end_timestamp timestamp NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_ip FOREIGN KEY (src_ip) REFERENCES IP (ip_address),
    CONSTRAINT valid_port CHECK (
        src_port BETWEEN 0 AND 65535
        AND dst_port BETWEEN 0 AND 65535
    ),
    CONSTRAINT end_time_after_start_time CHECK (start_timestamp <= end_timestamp)
);
CREATE TABLE SSHSession (
    session_id int NOT NULL,
    version text NOT NULL,
    PRIMARY KEY (session_id),
    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES Session (id)
);