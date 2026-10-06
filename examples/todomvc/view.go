package todomvc

import (
	"unicode/utf16"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// FilterTodos mirrors filterTodos (Todos.tsx:8-17). An unknown filter returns all todos.
func FilterTodos(filter string, todos []TodoItem) []TodoItem {
	switch filter {
	case FilterActive:
		return filterBy(todos, func(t TodoItem) bool { return !t.Completed })
	case FilterCompleted:
		return filterBy(todos, func(t TodoItem) bool { return t.Completed })
	}
	return append([]TodoItem{}, todos...)
}

// Derived holds the values Todos.tsx computes from the context on every render
// (Todos.tsx:53-56 and :148).
type Derived struct {
	Filtered              []TodoItem `json:"filtered"`
	NumActiveTodos        int        `json:"numActiveTodos"`
	AllCompleted          bool       `json:"allCompleted"`
	Mark                  string     `json:"mark"`
	ClearCompletedVisible bool       `json:"clearCompletedVisible"`
}

// Derive computes what the <Todos> component renders from filter and todos.
func Derive(filter string, todos []TodoItem) Derived {
	active := len(filterBy(todos, func(t TodoItem) bool { return !t.Completed }))
	allCompleted := len(todos) > 0 && active == 0
	mark := "completed"
	if allCompleted {
		mark = "active"
	}
	return Derived{
		Filtered:              FilterTodos(filter, todos),
		NumActiveTodos:        active,
		AllCompleted:          allCompleted,
		Mark:                  mark,
		ClearCompletedVisible: active < len(todos),
	}
}

// hashFilter mirrors `window.location.hash.slice(2)`; slice counts UTF-16 code units.
func hashFilter(hash string) string {
	units := utf16.Encode([]rune(hash))
	if len(units) <= 2 {
		return ""
	}
	return string(utf16.Decode(units[2:]))
}

// FilterFromHash mirrors `window.location.hash.slice(2) || 'all'` (Todos.tsx:40).
func FilterFromHash(hash string) string {
	if f := hashFilter(hash); f != "" {
		return f
	}
	return FilterAll
}

// InitialHashEvent mirrors the mount effect of Todos.tsx:45-51: the filter.change event
// sent for the initial URL hash, or false when the hash holds no filter.
func InitialHashEvent(hash string) (xs.E, bool) {
	f := hashFilter(hash)
	if f == "" {
		return nil, false
	}
	return xs.E{"type": "filter.change", "filter": f}, true
}
