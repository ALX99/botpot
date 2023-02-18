package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

type commonReq struct {
	ts         time.Time
	chID       uint32
	fromClient bool
}

func (r commonReq) Insert(tx pgx.Tx) (int, error) {
	row := tx.QueryRow(context.TODO(), `
	INSERT INTO Request(session_id, channel_id, ts, from_client)
		SELECT MAX(Session.id), $1, $2, $3
			FROM Session
    RETURNING id
`, r.chID, r.ts, r.fromClient)

	var id int
	err := row.Scan(&id)
	return id, err
}

type ptyReq struct {
	term     string
	modelist string
	columns  uint32
	rows     uint32
	width    uint32
	height   uint32

	c commonReq
}

func (r *ptyReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO PTYRequest(request_id, term, columns, rows, width, height, modelist)
		VALUES($1, $2, $3, $4, $5, $6, $7)
`, id, r.term, r.columns, r.rows, r.width, r.height, []byte(r.modelist))
	return err
}

type execReq struct {
	command string

	c commonReq
}

func (r *execReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO ExecRequest(request_id, command)
		VALUES($1, $2)
`, id, r.command)
	return err
}

type exitStatusReq struct {
	exitStatus uint32

	c commonReq
}

func (r *exitStatusReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO ExitStatusRequest(request_id, exit_status)
		VALUES($1, $2)
`, id, r.exitStatus)
	return err
}

type exitSignalReq struct {
	signalName string
	coreDumped bool
	errorMsg   string
	langTag    string

	c commonReq
}

func (r *exitSignalReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO ExitSignalRequest(request_id, signal_name, core_dumped, error_msg, language_tag)
		VALUES($1, $2, $3, $4, $5)
`, id, r.signalName, r.coreDumped, r.errorMsg, r.langTag)
	return err
}

type shellReq struct {
	c commonReq
}

func (r *shellReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO ShellRequest(request_id)
		VALUES($1)
`, id)
	return err
}

type windowDimChangeReq struct {
	columns uint32
	rows    uint32
	width   uint32
	height  uint32

	c commonReq
}

func (r *windowDimChangeReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO WindowDimChangeRequest(request_id, columns, rows, width, height)
		VALUES($1, $2, $3, $4, $5)
`, id, r.columns, r.rows, r.width, r.height)
	return err
}

type envReq struct {
	Name  string
	Value string

	c commonReq
}

func (r *envReq) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO EnvironmentRequest(request_id, name, value)
		VALUES($1, $2, $3)
`, id, r.Name, r.Value)
	return err
}

type subSystemRequest struct {
	Name string

	c commonReq
}

func (r *subSystemRequest) Insert(tx pgx.Tx) error {
	id, err := r.c.Insert(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(), `
	INSERT INTO SubSystemRequest(request_id, name)
		VALUES($1, $2)
`, id, r.Name)
	return err
}

// nolint:ireturn // needs to return interfae since is returns a bunch of different types
func newRequest(req *ssh.Request, fromClient bool, chID uint32, l zerolog.Logger) (request, error) {
	c := commonReq{ts: time.Now(), chID: chID, fromClient: fromClient}
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
			term:     r.Term,
			modelist: r.Modelist,
			columns:  r.Columns,
			rows:     r.Rows,
			width:    r.Width,
			height:   r.Height,
			c:        c,
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
			command: r.Command,
			c:       c,
		}, nil
	case ExitStatusRequest:
		r := struct{ ExitStatus uint32 }{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Uint32("exitStatus", r.ExitStatus).
			Str("type", req.Type).
			Msg("Got channel request")
		return &exitStatusReq{
			exitStatus: r.ExitStatus,
			c:          c,
		}, nil
	case ExitSignalRequest:
		r := struct {
			SignalName string
			CoreDumped bool
			ErrorMsg   string
			LangTag    string
		}{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Str("type", req.Type).
			Str("signalName", r.SignalName).
			Bool("coreDumped", r.CoreDumped).
			Str("errorMsg", r.ErrorMsg).
			Str("langTag", r.LangTag).
			Msg("Got channel request")
		return &exitSignalReq{
			signalName: r.SignalName,
			coreDumped: r.CoreDumped,
			errorMsg:   r.ErrorMsg,
			langTag:    r.LangTag,
			c:          c,
		}, nil
	case ShellRequest:
		l.Info().
			Str("type", req.Type).
			Msg("Got channel request")
		return &shellReq{
			c: c,
		}, nil
	case WindowDimensionChangeRequest:
		r := struct {
			Columns uint32
			Rows    uint32
			Width   uint32
			Height  uint32
		}{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Uint32("columns", r.Columns).
			Uint32("rows", r.Rows).
			Uint32("width", r.Width).
			Uint32("height", r.Height).
			Msg("Got channel request")
		return &windowDimChangeReq{
			columns: r.Columns,
			rows:    r.Rows,
			width:   r.Width,
			height:  r.Height,
			c:       c,
		}, nil
	case EnvironmentRequest:
		r := struct {
			Name  string
			Value string
		}{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Str("name", r.Name).
			Str("value", r.Value).
			Msg("Got channel request")
		return &envReq{
			Name:  r.Name,
			Value: r.Value,
			c:     c,
		}, nil
	case SubsystemRequest:
		r := struct {
			Name string
		}{}
		if err := ssh.Unmarshal(req.Payload, &r); err != nil {
			return nil, err
		}
		l.Info().
			Str("name", r.Name).
			Msg("Got channel request")
		return &subSystemRequest{
			Name: r.Name,
			c:    c,
		}, nil
	default:
		return nil, fmt.Errorf("request %q not supported", req.Type)
	}
}
