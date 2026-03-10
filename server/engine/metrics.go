package engine

import (
	"os"
	"sync/atomic"
)

type Metrics struct {
	// router metrics
	ReqTotal                  uint64
	LatencyB                  [5]uint64
	Resp2xx, Resp4xx, Resp5xx uint64

	// epoll metrics
	ActiveConn  int64
	BytesSent   uint64
	EpollErrors uint64
}

var Stats Metrics

var statsBuf [512]byte

func (e *Engine) PrintStats() {
	active := atomic.LoadInt64(&Stats.ActiveConn)

	epollErr := atomic.LoadUint64(&Stats.EpollErrors)
	r2xx := atomic.LoadUint64(&Stats.Resp2xx)
	r4xx := atomic.LoadUint64(&Stats.Resp4xx)
	r5xx := atomic.LoadUint64(&Stats.Resp5xx)

	pos := 0

	pos += copy(statsBuf[pos:], "\r[GENERAL]")
	pos += copy(statsBuf[pos:], " Active: ")
	pos = writeUint(statsBuf[:], pos, uint64(active), 6)
	pos += copy(statsBuf[pos:], "\n")
	pos += copy(statsBuf[pos:], "[LATENCY] <1ms: ")
	pos = writeUint(statsBuf[:], pos, atomic.LoadUint64(&Stats.LatencyB[0]), 6)
	pos += copy(statsBuf[pos:], " | <5ms: ")
	pos = writeUint(statsBuf[:], pos, atomic.LoadUint64(&Stats.LatencyB[1]), 6)
	pos += copy(statsBuf[pos:], " | >10ms: ")
	pos = writeUint(statsBuf[:], pos, atomic.LoadUint64(&Stats.LatencyB[4]), 6)
	pos += copy(statsBuf[pos:], "\n")

	pos += copy(statsBuf[pos:], "[STATUS] 2xx: ")
	pos = writeUint(statsBuf[:], pos, r2xx, 7)
	pos += copy(statsBuf[pos:], " | 4xx/5xx: ")
	pos = writeUint(statsBuf[:], pos, r4xx+r5xx, 6)
	pos += copy(statsBuf[pos:], " | EpollErr: ")
	pos = writeUint(statsBuf[:], pos, epollErr, 5)

	pos += copy(statsBuf[pos:], "\033[F\033[F")
	os.Stdout.Write(statsBuf[:pos])
}

func writeUint(buf []byte, pos int, val uint64, width int) int {
	var tmp [20]byte
	i := 0
	if val == 0 {
		tmp[i] = '0'
		i++
	} else {
		for val > 0 {
			tmp[i] = byte('0' + (val % 10))
			val /= 10
			i++
		}
	}
	for k := 0; k < width-i; k++ {
		buf[pos] = ' '
		pos++
	}
	for j := i - 1; j >= 0; j-- {
		buf[pos] = tmp[j]
		pos++
	}
	return pos
}
