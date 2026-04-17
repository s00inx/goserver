package engine

import (
	"sync/atomic"
	"time"
)

// i use global variable bc it is unefficient to store struct in session,
// i would use date only in response builder
var CurrentDate atomic.Pointer[[]byte]

// подход с двумя буферами и атомарной заменой указателя помогает избежать
// гонок данных (а также сохранить 0 аллокаций), потому что если бы я использовал 1 буфер, то адрес в памяти бы не менялся,
// а значит при изменении и чтении в 1 момент могла произойти гонка данных

var (
	buf1, buf2 [29]byte
	datetick   bool
)

func (e *Engine) UpdateDate() {
	var b []byte
	if datetick {
		b = buf1[:0]
	} else {
		b = buf2[:0]
	}

	res := time.Now().UTC().AppendFormat(b, time.RFC1123)

	CurrentDate.Store(&res)
	datetick = !datetick
}
