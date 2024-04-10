package cmdlang

import (
	"errors"
)

type evalCtx struct {
	parent   *evalCtx
	commands map[string]invokable
}

func (ec *evalCtx) addCmd(name string, inv invokable) {
	if ec.commands == nil {
		ec.commands = make(map[string]invokable)
	}

	ec.commands[name] = inv
}

func (ec *evalCtx) lookupCmd(name string) (invokable, error) {
	for e := ec; e != nil; e = e.parent {
		if ec.commands == nil {
			continue
		}

		if cmd, ok := ec.commands[name]; ok {
			return cmd, nil
		}

	}
	return nil, errors.New("name " + name + " not found")
}
