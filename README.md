# mvu

`mvu` is a small Go library for building [Gio](https://gioui.org/) applications with a Model-View-Update architecture and reactive rendering primitives.

It wraps Gio's window/event loop in a lightweight runtime that lets you describe your application as:

- a **model**: the current application state,
- **messages**: values that describe something that happened,
- an **update function**: pure-ish state transition logic that reacts to messages,
- a **view function**: a Gio widget derived from the current model, and
- **commands**: asynchronous or deferred work that can emit more messages.

The library is intentionally minimal. It does not prescribe a widget toolkit, styling system, or message hierarchy. A message is simply `any`, a view is a standard `layout.Widget`, and rendered layers are reactive `rx.Observable[layout.Widget]` values.

## Why use it?

Gio gives you immediate-mode UI primitives and direct access to the application event loop. `mvu` adds a thin coordination layer for applications that benefit from unidirectional data flow:

1. the runtime receives a message,
2. `Update` produces the next model and an optional command,
3. the model is mapped to a `View`,
4. the window is invalidated and rendered,
5. commands can emit more messages back into the loop.

This keeps state changes explicit while preserving the flexibility of Gio widgets and operations.

## Features

- Generic `Runtime[Model]` for typed application state.
- Elm-style `Init`, `Update`, and `View` functions.
- `Command` abstraction backed by `github.com/reactivego/rx` observables.
- Helpers for no-op, sequential, and concurrent commands.
- `MessageOp` for emitting messages from inside Gio layout code.
- Reactive window renderer that composes one or more observable widget layers.
- Direct access to the underlying `*app.Window` when needed.
- Compatible with Gio `v0.9.0`.

## Installation

```sh
go get github.com/vibrantgio/mvu
```

## Quick start

The following example creates a simple counter. Clicking the button emits an `Increment` message from the view. The runtime passes that message to `Update`, which returns the next model.

```go
package main

import (
	"fmt"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/vibrantgio/mvu"
)

type Model struct {
	Count int
}

type Increment struct{}

var button widget.Clickable
var theme = material.NewTheme()

func main() {
	go run()
	app.Main()
}

func run() {
	runtime := mvu.NewRuntime[Model](app.Title("MVU Counter"))

	runtime.Init = func() (Model, mvu.Command) {
		return Model{}, mvu.DoNothing()
	}

	runtime.Update = func(model Model, message mvu.Message) (Model, mvu.Command) {
		switch message.(type) {
		case Increment:
			model.Count++
		}
		return model, mvu.DoNothing()
	}

	runtime.View = func(model Model) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			for button.Clicked(gtx) {
				mvu.MessageOp{Message: Increment{}}.Add(gtx.Ops)
			}

			return layout.Center.Layout(gtx,
				material.Button(theme, &button, fmt.Sprintf("Count: %d", model.Count)).Layout,
			)
		}
	}

	if err := runtime.Run(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
```

Run it with:

```sh
go run .
```

## Core concepts

### Messages

A message is any Go value:

```go
type Message = any
```

Applications usually define small concrete message types:

```go
type Loaded struct {
	Items []Item
}

type Failed struct {
	Err error
}
```

Messages can come from several places:

- Gio layout code, via `mvu.MessageOp{Message: ...}.Add(gtx.Ops)`.
- Commands, by returning a non-nil message from `mvu.Do`.
- The window event stream exposed by `Window.Messages()`.

### Model

Your model is the typed application state managed by `Runtime[Model]`. It can be a struct, primitive value, pointer, or any other Go type.

```go
type Model struct {
	Loading bool
	Items   []Item
	Err     error
}
```

### Init

`Init` creates the initial model and an initial command. Use `mvu.DoNothing()` when there is no startup work.

```go
runtime.Init = func() (Model, mvu.Command) {
	return Model{Loading: true}, loadItems()
}
```

### Update

`Update` receives the current model and the next message. It returns the new model and a command to execute.

```go
runtime.Update = func(model Model, message mvu.Message) (Model, mvu.Command) {
	switch msg := message.(type) {
	case Loaded:
		model.Loading = false
		model.Items = msg.Items
	case Failed:
		model.Loading = false
		model.Err = msg.Err
	}
	return model, mvu.DoNothing()
}
```

### View

`View` maps the model to a Gio `layout.Widget`.

```go
runtime.View = func(model Model) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		// Draw using regular Gio operations and widgets.
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
```

When the view needs to notify the runtime about user interaction, add a `MessageOp` to the current `op.Ops`:

```go
mvu.MessageOp{Message: UserClicked{}}.Add(gtx.Ops)
```

The window renderer collects message operations during the frame and sends them into the runtime after the frame is submitted.

### Commands

A `Command` represents work that can emit messages. Commands are implemented as reactive observables, so they can be combined or sequenced.

Create a command with `mvu.Do`:

```go
func loadItems() mvu.Command {
	return mvu.Do(func() (mvu.Message, error) {
		items, err := fetchItems()
		if err != nil {
			return Failed{Err: err}, nil
		}
		return Loaded{Items: items}, nil
	})
}
```

Available helpers:

- `mvu.Do(fn)` runs a function and emits its returned message when non-nil.
- `mvu.DoNothing()` returns a command that completes without emitting a message.
- `mvu.DoConcurrent(cmds...)` merges multiple commands so they can run concurrently.
- `mvu.DoSequence(cmds...)` concatenates commands so they run in order.
- `cmd.Trace(name)` logs command start, completion, and failure information.

Command errors are caught by the runtime and printed as `Command Error: ...`. If you want errors to affect the model, return an error message value instead of returning a non-nil Go error.

## Rendering without `Runtime`

You can also use `Window` directly when you only need reactive rendering and do not need the full MVU loop.

```go
package main

import (
	"os"

	"gioui.org/app"
	"gioui.org/layout"

	"github.com/reactivego/rx"
	"github.com/vibrantgio/mvu"
)

func main() {
	go func() {
		window := mvu.NewWindow(app.Title("MVU - Minimal"))
		layer := rx.Of[layout.Widget](func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})

		window.Render(layer).Wait()
		os.Exit(0)
	}()
	app.Main()
}
```

`Window.Render` accepts any number of `rx.Observable[layout.Widget]` layers. The current value of each layer is rendered every frame, in the order passed to `Render`.

## API overview

### `Runtime[Model]`

```go
type Runtime[Model any] struct {
	Window *Window
	Init   func() (Model, Command)
	Update func(model Model, message Message) (Model, Command)
	View   func(model Model) layout.Widget
}
```

- `NewRuntime[Model](options ...app.Option) *Runtime[Model]` creates a runtime and its window.
- `Run(layers ...rx.Observable[layout.Widget]) error` starts the MVU loop and renders the runtime view after any additional layers.

Additional layers are useful for persistent backgrounds, overlays, animations, or debug UI that are driven by independent observables.

### `Window`

- `NewWindow(options ...app.Option) *Window` creates a Gio window wrapper.
- `Window() *app.Window` returns the underlying Gio window.
- `Messages() rx.Observable[Message]` exposes messages emitted by `MessageOp` during frames.
- `Render(layers ...rx.Observable[layout.Widget]) rx.Subscription` drives the Gio event loop until the window is destroyed.

### `MessageOp`

```go
type MessageOp struct{ Message }
```

Call `Add(gtx.Ops)` during layout to enqueue a message for the runtime/window message stream.

```go
mvu.MessageOp{Message: SomeMessage{}}.Add(gtx.Ops)
```

## Examples

The `example` module contains small Gio programs demonstrating direct window rendering and reactive layers, including:

- `example/01-minimal` — minimal window setup.
- `example/04-hello` — layered rendering with a backdrop and text.
- `example/edit` — Gio editor widgets inside reactive layers.
- `example/tweening` — animated color transitions driven by observables.

To run an example:

```sh
cd example/04-hello
go run .
```

## Design notes

- Gio's frame protocol is handled on a single goroutine inside `Window.Render`, which avoids deadlocks around `FrameEvent` handling.
- Layer observables are subscribed concurrently and stored as an atomic snapshot. Updating a layer invalidates the window so Gio schedules a new frame.
- `MessageOp` collection is scoped to the frame's `op.Ops`, allowing view code to emit messages without direct access to the runtime.
- `mvu` deliberately keeps messages untyped at the boundary. Use concrete message structs and type switches in your application for clarity.

## Requirements

- Go `1.23.8` or newer for the root module.
- Gio `v0.9.0`.
- `github.com/reactivego/rx` `v0.2.2`.

## License

No license file is currently included in this repository.
