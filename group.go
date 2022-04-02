package vibrant

import (
	"gioui.org/op"

	rx "github.com/reactivego/observable"
)

func Group(layers ...rx.Observable[op.CallOp]) rx.Observable[op.CallOp] {
	return rx.Map(rx.Combine(layers...), func(callops []op.CallOp) op.CallOp {
		if len(callops) == 1 {
			return callops[0]
		}
		ops := &op.Ops{}
		m := op.Record(ops)
		for _, co := range callops {
			co.Add(ops)
		}
		return m.Stop()
	})
}
