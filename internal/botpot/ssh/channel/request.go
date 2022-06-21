package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
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

func (r *ptyReq) Insert(tx pgx.Tx) error {
	_, err := tx.Exec(context.TODO(), `
	INSERT INTO PTYRequest(session_id, channel_id, ts, term, columns, rows, width, height, modelist)
		SELECT MAX(Session.id), $1, $2, $3, $4, $5, $6, $7, $8
			FROM Session
`, r.chID, r.ts, r.term, r.columns, r.rows, r.width, r.height, []byte(r.modelist))
	return err
}

func newRequest(req *ssh.Request, fromClient bool, chID uint32, l zerolog.Logger) (request, error) {
	switch req.Type {
	case "pty-req": // RFC 4254 Section 6.2.
		r := struct {
			Term     string
			Columns  uint32
			Rows     uint32
			Width    uint32
			Height   uint32
			Modelist string
		}{}
		err := ssh.Unmarshal(req.Payload, &r)
		if err != nil {
			return nil, err
		}
		l.Info().
			Str("term", r.Term).
			Uint32("columns", r.Columns).
			Uint32("rows", r.Rows).
			Uint32("width", r.Width).
			Uint32("height", r.Height).
			Str("modeList", r.Modelist).
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

	default:
		return nil, fmt.Errorf("request %q not supported", req.Type)

	}
}
