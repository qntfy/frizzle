package common

import (
	"sync"

	"github.com/qntfy/frizzle"
)

// UnAcked handles common tracking of UnAcked Msgs
type UnAcked struct {
	unAcked map[string]frizzle.Msg
	sync.RWMutex
}

// NewUnAcked returns an initialized struct
func NewUnAcked() *UnAcked {
	return &UnAcked{
		unAcked: make(map[string]frizzle.Msg),
	}
}

// Add a Msg to the UnAcked tracking
func (u *UnAcked) Add(m frizzle.Msg) {
	u.Lock()
	defer u.Unlock()
	u.unAcked[m.ID()] = m
}

// Remove a Msg from UnAcked tracking
func (u *UnAcked) Remove(m frizzle.Msg) error {
	u.Lock()
	defer u.Unlock()
	if _, ok := u.unAcked[m.ID()]; !ok {
		return frizzle.ErrAlreadyAcked
	}
	delete(u.unAcked, m.ID())
	return nil
}

// List current UnAcked Msgs
func (u *UnAcked) List() []frizzle.Msg {
	u.RLock()
	defer u.RUnlock()
	msgs := make([]frizzle.Msg, len(u.unAcked))
	i := 0
	for _, m := range u.unAcked {
		msgs[i] = m
		i++
	}
	return msgs
}

// Count of current UnAcked Msgs
func (u *UnAcked) Count() int {
	u.RLock()
	count := len(u.unAcked)
	u.RUnlock()
	return count
}
