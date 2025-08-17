package handlers

import (
	"fmt"
	"log/slog"
	"reflect"
)

// Handlers holds all the registered handlers
type Handlers struct {
	all map[string]*RegisteredHandler
}

// New creates and initializes a new HandlersRegistry instance.
func New() *Handlers {
	return &Handlers{
		all: make(map[string]*RegisteredHandler),
	}
}

// RegisteredHandler holds the compiled Go parts of a runner's lifecycle function.
type RegisteredHandler struct {
	Input     func() any
	InputType reflect.Type
	Deps      func() any
	Fn        any
}

// RegisterHandler registers a Go function for a runner's lifecycle event.
func (r *Handlers) RegisterHandler(name string, handler *RegisteredHandler) {
	if _, exists := r.all[name]; exists {
		panic(fmt.Sprintf("runner handler with name '%s' already registered", name))
	}
	slog.Debug("Registering runner handler.", "name", name)
	r.all[name] = handler
}
