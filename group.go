package vibrant

import (
	"gioui.org/op"

	"github.com/reactivego/x"
)

func Group(layers ...x.Observable[op.CallOp]) x.Observable[op.CallOp] {
	return x.Map(x.Combine(layers...), func(callops []op.CallOp) op.CallOp {
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
