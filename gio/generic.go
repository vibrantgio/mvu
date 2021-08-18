package gio

import (
	"github.com/reactivego/scheduler"
	"github.com/reactivego/subscriber"
)

type Foo int

type FooObserver func(next Foo, err error, done bool)

type ObservableFoo func(FooObserver, scheduler.Scheduler, subscriber.Subscriber)
