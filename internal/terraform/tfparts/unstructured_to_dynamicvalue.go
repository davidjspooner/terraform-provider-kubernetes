package tfparts

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func UnstructuredToDynamic(u unstructured.Unstructured) (basetypes.DynamicValue, error) {
	// Convert the unstructured object to a map[string]interface{}
	var dynamicValue basetypes.DynamicValue

	_, attrValue, err := anyToAttrValue(u.Object)
	if err != nil {
		return dynamicValue, err
	}
	// Create a new dynamic value
	dynamicValue = basetypes.NewDynamicValue(attrValue)

	return dynamicValue, nil
}

func DiagsToGoError(diags diag.Diagnostics) error {
	if diags.HasError() {
		msg := diags[0].Detail()
		if msg == "" {
			msg = diags[0].Summary()
		}

		return errors.New(msg)
	}
	return nil
}

func anyToAttrValue(v interface{}) (attr.Type, attr.Value, error) {
	switch v := v.(type) {
	case string:
		return types.StringType, basetypes.NewStringValue(v), nil
	case int64:
		return types.Int64Type, basetypes.NewInt64Value(v), nil
	case float64:
		return types.Float64Type, basetypes.NewFloat64Value(v), nil
	case bool:
		return types.BoolType, basetypes.NewBoolValue(v), nil
	default:
		rValue := reflect.ValueOf(v)
		for rValue.Kind() == reflect.Ptr || rValue.Kind() == reflect.Interface {
			if rValue.IsNil() {
				return types.DynamicType, basetypes.NewDynamicNull(), nil
			} else {
				rValue = rValue.Elem()
			}
		}
		switch rValue.Kind() {
		case reflect.Map:
			m := make(map[string]attr.Value)
			t := make(map[string]attr.Type)
			for _, key := range rValue.MapKeys() {
				keyValue := key.Interface()
				value := rValue.MapIndex(key)
				typeAttr, valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, nil, err
				}
				key := fmt.Sprintf("%v", keyValue)
				m[key] = valueAttr
				t[key] = typeAttr
			}
			obj, diags := basetypes.NewObjectValue(t, m)
			if diags.HasError() {
				return nil, nil, DiagsToGoError(diags)
			}
			return types.ObjectType{AttrTypes: t}, obj, nil
		case reflect.Slice, reflect.Array:
			s := make([]attr.Value, rValue.Len())
			t := make([]attr.Type, rValue.Len())
			for i := 0; i < rValue.Len(); i++ {
				value := rValue.Index(i)
				typeAttr, valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, nil, err
				}
				s[i] = valueAttr
				t[i] = typeAttr
			}
			tupple, diags := basetypes.NewTupleValue(t, s)
			if diags.HasError() {
				return nil, nil, DiagsToGoError(diags)
			}
			return types.TupleType{ElemTypes: t}, tupple, nil
		case reflect.Struct:
			s := make(map[string]attr.Value)
			t := make(map[string]attr.Type)
			for i := 0; i < rValue.NumField(); i++ {
				field := rValue.Type().Field(i)
				value := rValue.Field(i)
				tags := field.Tag.Get("yaml")
				parts := strings.Split(tags, ",")
				if strings.Contains(tags, "omitempty") {
					if value.IsZero() {
						continue
					}
				}
				typeAttr, valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, nil, err
				}
				if len(parts) > 0 && parts[0] != "" {
					s[parts[0]] = valueAttr
					t[parts[0]] = typeAttr
				} else {
					s[field.Name] = valueAttr
					t[field.Name] = typeAttr
				}
			}
			object, diags := basetypes.NewObjectValue(t, s)
			if diags.HasError() {
				return nil, nil, DiagsToGoError(diags)
			}
			return types.ObjectType{AttrTypes: t}, object, nil
		default:
			return nil, nil, fmt.Errorf("unsupported type: %s", rValue.Kind())
		}
	}
}
