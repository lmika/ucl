package cmdlang

import (
	"errors"
)

type evalCtx struct {
	parent        *evalCtx
	currentStream stream
	commands      map[string]invokable
	vars          map[string]object
}

func (ec *evalCtx) withCurrentStream(s stream) *evalCtx {
	return &evalCtx{
		parent:        ec,
		currentStream: s,
	}
}

func (ec *evalCtx) addCmd(name string, inv invokable) {
	if ec.commands == nil {
		ec.commands = make(map[string]invokable)
	}

	ec.commands[name] = inv
}

func (ec *evalCtx) setVar(name string, val object) {
	if ec.vars == nil {
		ec.vars = make(map[string]object)
	}
	ec.vars[name] = val
}

func (ec *evalCtx) getVar(name string) (object, bool) {
	if ec.vars == nil {
		return nil, false
	}

	if v, ok := ec.vars[name]; ok {
		return v, true
	} else if ec.parent != nil {
		return ec.parent.getVar(name)
	}

	return nil, false
}

func (ec *evalCtx) lookupCmd(name string) (invokable, error) {
	for e := ec; e != nil; e = e.parent {
		if e.commands == nil {
			continue
		}

		if cmd, ok := e.commands[name]; ok {
			return cmd, nil
		}

	}
	return nil, errors.New("name " + name + " not found")
}
