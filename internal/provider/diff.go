package provider

import (
	"reflect"

	"github.com/davidjspooner/dsvalue/pkg/path"
	"github.com/davidjspooner/dsvalue/pkg/reflected"
	"github.com/davidjspooner/dsvalue/pkg/value"
)

func diffResources(left, right interface{}) ([]string, error) {
	var diffs []string
	leftRoot, err := reflected.NewReflectedObject(reflect.ValueOf(left), nil)
	if err != nil {
		return nil, err
	}
	rightRoot, err := reflected.NewReflectedObject(reflect.ValueOf(right), nil)
	if err != nil {
		return nil, err
	}
	err = path.Diff(leftRoot, rightRoot, func(p path.Path, left, right value.Value) error {
		pathString := p.String()
		diffs = append(diffs, pathString)
		return nil
	})
	return diffs, err
}
