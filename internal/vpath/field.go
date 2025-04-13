package vpath

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

type Field string

var identifier = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)

func (f Field) String() string {
	s := string(f)
	if identifier.MatchString(s) {
		s = fmt.Sprintf(".%s", s)
	} else {
		s = fmt.Sprintf("[%q]", s)
	}
	return s
}

func (f Field) EvaluateFor(object interface{}) (interface{}, error) {
	if object == nil {
		return nil, nil
	}
	switch o := object.(type) {
	case map[string]interface{}:
		if v, ok := o[string(f)]; ok {
			return v, nil
		}
	case []interface{}:
		i, err := strconv.Atoi(string(f))
		if err == nil && i >= 0 && i < len(o) {
			return o[i], nil
		}
	}
	rObject := reflect.ValueOf(object)
	for rObject.Kind() == reflect.Ptr && !rObject.IsNil() {
		rObject = rObject.Elem()
	}
	switch rObject.Kind() {
	case reflect.Map:
		if rObject.Type().Key().Kind() == reflect.String {
			if v := rObject.MapIndex(reflect.ValueOf(string(f))); v.IsValid() {
				return v.Interface(), nil
			}
		}
	case reflect.Array, reflect.Slice:
		i, err := strconv.Atoi(string(f))
		if i < 0 {
			i = rObject.Len() + i
		}
		if err == nil {
			if i >= 0 && i < rObject.Len() {
				return rObject.Index(i).Interface(), nil
			}
		}
	case reflect.Struct:
		child := rObject.FieldByName(string(f)).Interface()
		return child, nil
	}
	return nil, fmt.Errorf("cannot extract %q from %T", f, object)
}
