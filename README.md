# Botpot

Botpot is an interactive SSH honeypot that supports all requests defined in [RFC 4254](https://www.rfc-editor.org/rfc/rfc4254). 
It works by acting as a proxy between the attacker-initiated SSH connection and a Docker container, 
parsing all the data sent between the two connections. When either the Docker container or the attacker 
disconnects, the session data is saved in a PostgreSQL database.

**NOTE:** The project is in active development and changes to the database may occur

```mermaid
flowchart BT
  subgraph net[Docker Network]
    direction LR
    subgraph internal[Internal]
        Botpot -- SSH --> Honeypot
    end
    Botpot --> postgres[(PostgreSQL)]
    Grafana --> postgres
  end
  A[Attacker] -- SSH --> Botpot
```

## Features

- Supports all SSH requests defined in [RFC 4254](https://www.rfc-editor.org/rfc/rfc4254)
- Does not do any emulation, making it indistinguishable from a real SSH connection
- Keeps a buffer of honeypot containers running, minimizing delay for attackers
- Logs all data collected during the session and saves it in a PostgreSQL database
- Provides visualizations of the collected data through Grafana.
- Built on top of a [distroless image](https://github.com/GoogleContainerTools/distroless)

## Sequence Diagram

```mermaid
sequenceDiagram
    participant A as Attacker
    participant B as Botpot
    participant H as Honeypot
    participant D as Docker
    participant P as PostgreSQL

    loop forever
        alt len(Available Honeypots) < buffer
            B->>D: Start Honeypot
        end
    end

    loop for each attacker
        A->>B: SSH Connect
        B->>H: SSH Connect
        H-->>B: SSH Established
        B-->>A: SSH Established
    
        loop until disconnected
            critical proxy data
                A->>B: SSH request
                B-->B: Parse request & store data
                B->>H: SSH request
                H-->>B: SSH response
                B-->B: Parse request & store data
                B-->>A: SSH response
            end
        end

        A->>B: SSH Disconnect
        B->>H: SSH Disconnect
        H-->>B: SSH Disconnected
        B-->>A: SSH Disconnected
        
        B->>P: Store data
    end
```

## Database ER Diagram

```mermaid
erDiagram
    IP {
        ip_address INET
    }
    SESSION {
        id SERIAL
        version TEXT
        stdout TEXT
        timing TEXT
        src_ip IP
        src_port INT
        dst_ip INET
        dst_port INT
        start_ts TIMESTAMPZ
        end_ts TIMESTAMPZ
    }
    CHANNEL {
        id INT
        session_id INT
        channel_type TEXT
        recv BYTEA
        recv_stderr BYTEA
        start_ts TIMESTAMPZ
        end_ts TIMESTAMPZ
    }
    REQUEST {
        id SERIAL
        session_id INT
        ts TIMESTAMPZ
        from_client BOOLEAN
    }
    PTYREQUEST {
        request_id serial
        term TEXT
        colulmns INT
        rows INT
        width INT
        height INT
        modelist BYTEA
    }
    EXECREQUEST {
        request_id SERIAL
        command TEXT
    }
    EXITSTATUSREQUEST {
        request_id SERIAL
        exit_status INT
    }
    EXITSIGNALREQUEST {
        request_id SERIAL
        signal_name TEXT
        core_dumped BOOLEAN
        error_msg TEXT
        language_tag TEXT
    }
    SHELLREQUEST {
        request_id SERIAL
    }
    WINDOWDIMCHANGEREQUEST {
        request_id SERIAL
        columns INT
        rows INT
        width INT
        height INT
    }
    ENVIRONMENTREQUEST {
        request_id SERIAL
        name TEXT
        value TEXT
    }
    SUBSYSTEMREQUEST {
        request_id SERIAL
        name TEXT
    }

    SESSION }|--|| IP : contains
    SESSION ||--o{ CHANNEL : has
    SESSION ||--o{ REQUEST : has
    CHANNEL ||--o{ REQUEST : has
    REQUEST ||--o{ PTYREQUEST : has
    REQUEST ||--o{ EXECREQUEST : has
    REQUEST ||--o{ EXITSTATUSREQUEST : has
    REQUEST ||--o{ EXITSIGNALREQUEST : has
    REQUEST ||--o{ SHELLREQUEST : has
    REQUEST ||--o{ WINDOWDIMCHANGEREQUEST : has
    REQUEST ||--o{ ENVIRONMENTREQUEST : has
    REQUEST ||--o{ SUBSYSTEMREQUEST : has
```
