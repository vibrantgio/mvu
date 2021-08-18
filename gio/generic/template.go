package rx

import (
	"github.com/reactivego/scheduler"
	"github.com/reactivego/subscriber"
	"github.com/reactivego/vibrant/gio"
)

//jig:template TypeGio<Foo>

type Foo = gio.Foo

//jig:template ExtendGioObservable<Foo>
//jig:needs TypeGio<Foo>

// ExtendGioObservableFoo converts a gio Observable to a local type so operators can be generated for it.
func ExtendGioObservableFoo(o gio.ObservableFoo) ObservableFoo {
	observable := func(observe FooObserver, scheduler scheduler.Scheduler, subscriber subscriber.Subscriber) {
		o(gio.FooObserver(observe), scheduler, subscriber)
	}
	return observable
}

//jig:template AliasGioObservable<Foo>
//jig:needs TypeGio<Foo>

type FooObserver = gio.FooObserver
type ObservableFoo = gio.ObservableFoo

// AliasGioObservableFoo forces generation type alias ObservableFoo for gio.ObservableFoo.
func AliasGioObservableFoo() {
	var o ObservableFoo
	_ = gio.ObservableFoo(o)
}

//jig:template AsGioObservable<Foo>
//jig:needs AliasGioObservable<Foo>

// AsGioObservableFoo forces ObservableFoo to actually alias the gio.ObservableFoo type.
func AsGioObservableFoo(o ObservableFoo) gio.ObservableFoo {
	return gio.ObservableFoo(o)
}
