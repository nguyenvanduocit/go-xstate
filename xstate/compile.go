package xstate

import (
	"fmt"
	"strings"
)

// ConfigError identifies a statically invalid statechart configuration.
// Error preserves the message used by CreateMachine; StateID identifies the
// node being configured or resolved. Unwrap exposes the underlying error.
type ConfigError struct {
	StateID string
	Err     error
}

func (e *ConfigError) Error() string { return e.Err.Error() }
func (e *ConfigError) Unwrap() error { return e.Err }

func invalidConfig(stateID string, err error) *ConfigError {
	return &ConfigError{StateID: stateID, Err: err}
}

// Compile constructs a machine and returns static configuration errors.
// Unlike CreateMachine, it also checks deferred initial-state errors before
// returning. It does not execute actions, guards, context factories or inputs,
// and does not validate dynamic expressions or late-bound implementations.
// Programmer panics are not converted into configuration errors.
func Compile[C any](config MachineConfig[C], impl ...Implementations) (machine *StateMachine[C], err error) {
	defer func() {
		if value := recover(); value != nil {
			configErr, ok := value.(*ConfigError)
			if !ok {
				panic(value)
			}
			machine, err = nil, configErr
		}
	}()
	rootID := config.ID
	if rootID == "" {
		rootID = "(machine)"
	}
	if err = validateConfigTree(StateConfig{ID: rootID, Type: config.Type, States: config.States}, rootID, nil, map[string]bool{}); err != nil {
		return nil, err
	}
	machine = CreateMachine(config, impl...)
	var validateInitial func(*StateNode) error
	validateInitial = func(n *StateNode) error {
		if n.initialErr != nil {
			configErr, ok := n.initialErr.(*ConfigError)
			if !ok {
				panic(n.initialErr)
			}
			return configErr
		}
		if n.Type == History {
			if n.Parent == nil {
				return invalidConfig(n.ID, fmt.Errorf("history state requires a parent"))
			}
			if n.Config.Target != "" {
				_ = resolveHistoryDefaultTransition(n)
			}
		}
		for _, key := range n.childOrder {
			if err := validateInitial(n.States[key]); err != nil {
				return err
			}
		}
		return nil
	}
	if err = validateInitial(machine.Root); err != nil {
		return nil, err
	}
	return machine, nil
}

func validateConfigTree(config StateConfig, rootID string, path []string, ids map[string]bool) error {
	id := config.ID
	if id == "" {
		id = strings.Join(append([]string{rootID}, path...), ".")
	}
	if ids[id] {
		return invalidConfig(id, fmt.Errorf("duplicate state id %q", id))
	}
	ids[id] = true
	switch config.Type {
	case "", Atomic, Compound, Parallel, Final, History:
	default:
		return invalidConfig(id, fmt.Errorf("invalid state type %q", config.Type))
	}
	switch config.History {
	case "", Shallow, Deep:
	default:
		return invalidConfig(id, fmt.Errorf("invalid history type %q", config.History))
	}
	keys := make(map[string]bool, len(config.States))
	for _, child := range config.States {
		if keys[child.Key] {
			return invalidConfig(id, fmt.Errorf("duplicate child state %q", child.Key))
		}
		keys[child.Key] = true
		childPath := append(append([]string(nil), path...), child.Key)
		if err := validateConfigTree(child, rootID, childPath, ids); err != nil {
			return err
		}
	}
	return nil
}
