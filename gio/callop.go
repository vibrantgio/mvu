package gio

import "gioui.org/op"

func (o ObservableCallOp) Paint(other ...ObservableCallOp) ObservableCallOp {
	observable := o.CombineLatestWith(other...).MapCallOp(func(callops []CallOp) CallOp {
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
	return observable
}
