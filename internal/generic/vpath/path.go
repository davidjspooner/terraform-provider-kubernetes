package vpath

import "strings"

type Element interface {
	String() string
	EvaluateFor(interface{}) (interface{}, error)
}

type Path []Element

func (c Path) String() string {
	s := strings.Builder{}
	for _, e := range c {
		s.WriteString(e.String())
	}
	return s.String()
}

func (c Path) EvaluateFor(object interface{}) (interface{}, error) {
	at := strings.Builder{}
	var err error
	for _, e := range c {
		object, err = e.EvaluateFor(object)
		if err != nil {
			//do something with 'at'
			return nil, err
		}
		at.WriteString(e.String())
	}
	return object, nil
}

func Evaluate[T any](c Path, object interface{}) (T, error) {
	r, err := c.EvaluateFor(object)
	if err != nil {
		return *new(T), err
	}
	if object == nil {
		return *new(T), nil
	}
	v, ok := r.(T)
	if !ok {
		return *new(T), nil
	}
	return v, nil
}
