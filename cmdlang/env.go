package cmdlang

type evalCtx struct {
	root     *evalCtx
	parent   *evalCtx
	commands map[string]invokable
	macros   map[string]macroable
	vars     map[string]object
}

func (ec *evalCtx) fork() *evalCtx {
	return &evalCtx{parent: ec, root: ec.root}
}

func (ec *evalCtx) addCmd(name string, inv invokable) {
	if ec.commands == nil {
		ec.commands = make(map[string]invokable)
	}

	ec.commands[name] = inv
}

func (ec *evalCtx) addMacro(name string, inv macroable) {
	if ec.macros == nil {
		ec.macros = make(map[string]macroable)
	}

	ec.macros[name] = inv
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

func (ec *evalCtx) lookupInvokable(name string) invokable {
	if ec == nil {
		return nil
	}

	for e := ec; e != nil; e = e.parent {
		if cmd, ok := e.commands[name]; ok {
			return cmd
		}
	}

	return ec.parent.lookupInvokable(name)
}

func (ec *evalCtx) lookupMacro(name string) macroable {
	if ec == nil {
		return nil
	}

	for e := ec; e != nil; e = e.parent {
		if cmd, ok := e.macros[name]; ok {
			return cmd
		}
	}

	return ec.parent.lookupMacro(name)
}
