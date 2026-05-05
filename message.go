package mvu

import (
	"sync"

	"gioui.org/op"
)

type Message = any

type MessageOp struct{ Message }

var (
	collectorMu sync.Mutex
	collectors  = make(map[*op.Ops]*[]MessageOp)
)

func registerCollector(o *op.Ops, msgs *[]MessageOp) {
	collectorMu.Lock()
	collectors[o] = msgs
	collectorMu.Unlock()
}

func unregisterCollector(o *op.Ops) {
	collectorMu.Lock()
	delete(collectors, o)
	collectorMu.Unlock()
}

func (msgOp MessageOp) Add(o *op.Ops) {
	collectorMu.Lock()
	msgs, ok := collectors[o]
	collectorMu.Unlock()
	if ok {
		*msgs = append(*msgs, msgOp)
	}
}
