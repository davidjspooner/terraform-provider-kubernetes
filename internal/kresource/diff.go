package kresource

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrAbortComparison = errors.New("abort comparison")

type DifferenceHandler interface {
	HandleDifference(path string, left, right interface{}) error
}

type DifferenceHandlerFunc func(path string, left, right interface{}) error

func (dh DifferenceHandlerFunc) HandleDifference(path string, left, right interface{}) error {
	return dh(path, left, right)
}

func FindDifferences(prefix string, left, right interface{}, handler DifferenceHandler) error {
	err := compareReflect(prefix, reflect.ValueOf(left), reflect.ValueOf(right), handler)
	if err == ErrAbortComparison {
		return nil
	}
	return err
}

var reflectType = reflect.TypeOf((*reflect.Value)(nil)).Elem()

func compareReflect(path string, left, right reflect.Value, handler DifferenceHandler) error {

	var err error

	// Dereference pointers

	for {
		if left.Kind() == reflect.Ptr && !left.IsNil() {
			left = left.Elem()
		} else if left.Type() == reflectType {
			left = left.Elem()
		} else {
			break
		}
	}
	for {
		if right.Kind() == reflect.Ptr && !right.IsNil() {
			right = right.Elem()
		} else if right.Type() == reflectType {
			right = right.Elem()
		} else {
			break
		}
	}

	// Check if kinds differ
	if left.Kind() != right.Kind() {
		err = handler.HandleDifference(path, safeInterface(left), safeInterface(right))
		if err != nil {
			return err
		}
		return nil
	}

	switch left.Kind() {
	case reflect.Struct:
		for i := 0; i < left.NumField(); i++ {
			fieldName := left.Type().Field(i).Name
			newPath := fmt.Sprintf("%s.%s", path, fieldName)
			err = compareReflect(newPath, left.Field(i), right.Field(i), handler)
			if err != nil {
				return err
			}
		}
	case reflect.Map:

		allKeys := make(map[string]reflect.Value)
		for _, key := range left.MapKeys() {
			keyAsString := fmt.Sprintf("%v", key)
			allKeys[keyAsString] = key
		}
		for _, key := range right.MapKeys() {
			keyAsString := fmt.Sprintf("%v", key)
			if _, ok := allKeys[keyAsString]; !ok {
				allKeys[keyAsString] = key
			}
		}

		// Compare keys in both maps
		for sKey, key := range allKeys {
			newPath := fmt.Sprintf("%s.%s", path, sKey)
			lValue := left.MapIndex(key)
			rValue := right.MapIndex(key)
			if !lValue.IsValid() {
				err = handler.HandleDifference(newPath, nil, safeInterface(rValue))
				if err != nil {
					return err
				}
				continue
			}
			if !rValue.IsValid() {
				err = handler.HandleDifference(newPath, safeInterface(lValue), nil)
				if err != nil {
					return err
				}
				continue
			}
			err = compareReflect(newPath, lValue, rValue, handler)
			if err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		lLen := left.Len()
		rLen := right.Len()
		minLen := min(lLen, rLen)
		for i := 0; i < minLen; i++ {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			lVal := left.Index(i)
			rVal := right.Index(i)
			err := compareReflect(newPath, lVal, rVal, handler)
			if err != nil {
				return err
			}
		}
		if lLen > rLen {
			for i := rLen; i < lLen; i++ {
				newPath := fmt.Sprintf("%s[%d]", path, i)
				lVal := left.Index(i)
				err = handler.HandleDifference(newPath, safeInterface(lVal), nil)
				if err != nil {
					return err
				}
			}
		} else if rLen > lLen {
			for i := lLen; i < rLen; i++ {
				newPath := fmt.Sprintf("%s[%d]", path, i)
				rVal := right.Index(i)
				err = handler.HandleDifference(newPath, nil, safeInterface(rVal))
				if err != nil {
					return err
				}
			}
		}
	default:
		if !reflect.DeepEqual(left.Interface(), right.Interface()) {
			err = handler.HandleDifference(path, safeInterface(left), safeInterface(right))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func safeInterface(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

//----

func DiffResources(left, right interface{}) ([]string, error) {
	var differences []string
	handleDifferences := DifferenceHandlerFunc(func(path string, left, right interface{}) error {
		differences = append(differences, path)
		return nil
	})
	err := compareReflect("", reflect.ValueOf(left), reflect.ValueOf(right), handleDifferences)
	return differences, err
}
