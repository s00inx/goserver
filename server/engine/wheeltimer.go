package engine

import (
	"sync/atomic"
	"syscall"
)

// timer wheel for request timeout,
// mask should be timeout - 1 (no alignment bc struct created only at start)
type TimerWheel struct {
	slots  [1 << 9]*Session // (NOTE: power of 2 for using bitmask over %)
	cursor int              // cur wheel slot
	mask   int
}

func NewWheel() *TimerWheel {
	return &TimerWheel{
		slots: [1 << 9]*Session{},
		mask:  1<<9 - 1,
	}
}

// update timer wheel bucket with O(1)
func (tw *TimerWheel) Update(s *Session) {
	if s.tprev != nil {
		s.tprev.tnext = s.tnext
	} else {
		if tw.slots[s.slot] == s {
			tw.slots[s.slot] = s.tnext
		}
	}

	if s.tnext != nil {
		s.tnext.tprev = s.tprev
	}

	ns := (tw.cursor + 40) & tw.mask
	s.tnext = tw.slots[ns]
	s.tprev = nil
	s.slot = ns

	if s.tnext != nil {
		s.tnext.tprev = s
	}

	tw.slots[ns] = s
}

// start goroutine that kills processes with timeout
func (tw *TimerWheel) killSharded(ss []atomic.Pointer[Session]) {
	tw.cursor = (tw.cursor + 1) & tw.mask

	explisthead := tw.slots[tw.cursor]
	tw.slots[tw.cursor] = nil

	cur := explisthead
	for cur != nil {
		next := cur.tnext

		if cur.inWork.Load() {
			cur = next
			continue
		}

		// ATOMICALLY compare and swap
		if ss[cur.Fd].CompareAndSwap(cur, nil) {
			cur.tnext = nil
			cur.tprev = nil

			if cur.bufraw != nil {
				bufPool.Put(cur.bufraw)
				cur.Buf = nil
				cur.bufraw = nil
			}

			sessionPool.Put(cur.raw)

			syscall.Close(int(cur.Fd))
			cur.Reset()
		}

		cur = next
	}
}
