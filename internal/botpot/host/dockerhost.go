package host

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type DHost struct {
	id       string
	running  bool
	occupied bool
	sync.RWMutex
}

func NewDHost(id string) *DHost {
	return &DHost{id: id}
}

func (h *DHost) SetRunning(val bool) {
	h.Lock()
	h.running = val
	h.Unlock()
}

func (h *DHost) SetOccupied(val bool) {
	log.Debug().Str("ID", h.id).Bool("occupied", val).Send()
	h.Lock()
	h.occupied = val
	h.Unlock()
}

func (h *DHost) Running() bool {
	h.RLock()
	defer h.RUnlock()
	return h.running
}

func (h *DHost) Occupied() bool {
	h.RLock()
	defer h.RUnlock()
	return h.occupied
}

func (h *DHost) ID() string {
	return h.id
}
