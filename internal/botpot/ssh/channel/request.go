package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Request types used in sessions - RFC 4254 6.X
const (
	SessionRequest               = "session"       // RFC 4254 6.1
	PTYRequest                   = "pty-req"       // RFC 4254 6.2
	X11Request                   = "x11-req"       // RFC 4254 6.3.1
	X11ChannelRequest            = "x11"           // RFC 4254 6.3.2
	EnvironmentRequest           = "env"           // RFC 4254 6.4
	ShellRequest                 = "shell"         // RFC 4254 6.5
	ExecRequest                  = "exec"          // RFC 4254 6.5
	SubsystemRequest             = "subsystem"     // RFC 4254 6.5
	WindowDimensionChangeRequest = "window-change" // RFC 4254 6.7
	FlowControlRequest           = "xon-off"       // RFC 4254 6.8
	SignalRequest                = "signal"        // RFC 4254 6.9
	ExitStatusRequest            = "exit-status"   // RFC 4254 6.10
	ExitSignalRequest            = "exit-signal"   // RFC 4254 6.10
)

type request interface {
	Insert(tx pgx.Tx) error
}

type ptyReq struct {
	term     string
	modelist string
	columns  uint32
	rows     uint32
	width    uint32
	height   uint32

	ts         time.Time
	chID       uint32
	fromClient bool
}

type execReq struct {
	command string

	ts         time.Time
	chID       uint32
	fromClient bool
}

func (r *ptyReq) Insert(tx pgx.Tx) error {
	_, err := tx.Exec(context.TODO(), `
	INSERT INTO PTYRequest(session_id, channel_id, ts, term, columns, rows, width, height, modelist, from_client)
		SELECT MAX(Session.id), $1, $2, $3, $4, $5, $6, $7, $8, $9
			FROM Session
`, r.chID, r.ts, r.term, r.columns, r.rows, r.width, r.height, []byte(r.modelist), r.fromClient)
	return err
}

func (r *execReq) Insert(tx pgx.Tx) error {
	_, err := tx.Exec(context.TODO(), `
	INSERT INTO ExecRequest(session_id, channel_id, ts, command, from_client)
		SELECT MAX(Session.id), $1, $2, $3, $4
			FROM Session
`, r.chID, r.ts, r.command, r.fromClient)
	return err
}

func newRequest(req *ssh.Request, fromClient bool, chID uint32, l zerolog.Logger) (request, error) {
	switch req.Type {
	case PTYRequest:
		r := struct {
			Term     string
			Columns  uint32
			Rows     uint32
			Width    uint32
			Height   uint32
			Modelist string
		}{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Str("term", r.Term).
			Uint32("columns", r.Columns).
			Uint32("rows", r.Rows).
			Uint32("width", r.Width).
			Uint32("height", r.Height).
			Str("modeList", r.Modelist).
			Str("type", req.Type).
			Msg("Got channel request")
		return &ptyReq{
			term:       r.Term,
			modelist:   r.Modelist,
			columns:    r.Columns,
			rows:       r.Rows,
			width:      r.Width,
			height:     r.Height,
			fromClient: fromClient,
			ts:         time.Now(),
			chID:       chID,
		}, nil
	case ExecRequest:
		r := struct{ Command string }{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Str("command", r.Command).
			Str("type", req.Type).
			Msg("Got channel request")
		return &execReq{
			command:    r.Command,
			ts:         time.Now(),
			chID:       chID,
			fromClient: fromClient,
		}, nil
	default:
		return nil, fmt.Errorf("request %q not supported", req.Type)

	}
}
