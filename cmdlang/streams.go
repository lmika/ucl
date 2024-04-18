package cmdlang

import (
	"errors"
	"fmt"
	"io"
)

// stream is an object which returns a collection of objects from a source.
// These are used to create pipelines
//
// The stream implementation can expect close to be called if at least one next() call is made.  Otherwise
// closableStream cannot assume that close will be called (the pipe may be left unconsumed, for example).
//
// It is the job of the final iterator to call close. Any steam that consumes from another stream must
// implement this, and call close on the parent stream.
type stream interface {
	object

	// next pulls the next object from the stream.  If an object is available, the result is the
	// object and a nil error.  If no more objects are available, error returns io.EOF.
	// Otherwise, an error is returned.
	next() (object, error)

	close() error
}

// forEach will iterate over all the items of a stream. The iterating function can return an error, which will
// be returned as is. A stream that has consumed every item will return nil. The stream will automatically be closed.
func forEach(s stream, f func(object, int) error) (err error) {
	defer s.close()

	var sv object
	i := 0
	for sv, err = s.next(); err == nil; sv, err = s.next() {
		if err := f(sv, i); err != nil {
			return err
		}
		i += 1
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// asStream converts an object to a stream.  If t is already a stream, it's returned as is.
// Otherwise, a singleton stream is returned.
func asStream(v object) stream {
	switch s := v.(type) {
	case stream:
		return s
	case listObject:
		return &listIterStream{list: s}
	}

	return &singletonStream{t: v}
}

type emptyStream struct{}

func (s *emptyStream) String() string {
	return "(nil)"
}

func (s emptyStream) next() (object, error) {
	return nil, io.EOF
}

func (s emptyStream) close() error { return nil }

type singletonStream struct {
	t        object
	consumed bool
}

func (s *singletonStream) String() string {
	return s.t.String()
}

func (s *singletonStream) Truthy() bool {
	return !s.consumed
}

func (s *singletonStream) next() (object, error) {
	if s.consumed {
		return nil, io.EOF
	}
	s.consumed = true
	return s.t, nil
}

func (s *singletonStream) close() error { return nil }

type listIterStream struct {
	list []object
	cusr int
}

func (s *listIterStream) String() string {
	return fmt.Sprintf("listIterStream{list: %v}", s.list)
}

func (s *listIterStream) Truthy() bool {
	return len(s.list) > s.cusr
}

func (s *listIterStream) next() (o object, err error) {
	if s.cusr >= len(s.list) {
		return nil, io.EOF
	}

	o = s.list[s.cusr]
	s.cusr += 1

	return o, nil
}

func (s *listIterStream) close() error { return nil }

type mapFilterStream struct {
	in    stream
	mapFn func(x object) (object, bool, error)
}

func (ms mapFilterStream) String() string {
	return fmt.Sprintf("mapFilterStream{in: %v}", ms.in)
}

func (ms mapFilterStream) Truthy() bool {
	return true // ???
}

func (ms mapFilterStream) next() (object, error) {
	for {
		u, err := ms.in.next()
		if err != nil {
			return nil, err
		}

		t, ok, err := ms.mapFn(u)
		if err != nil {
			return nil, err
		} else if ok {
			return t, nil
		}
	}
}

func (ms mapFilterStream) close() error {
	return ms.in.close()
}
