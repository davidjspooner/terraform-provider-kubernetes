package tfparts

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func UnstructuredToDynamic(u unstructured.Unstructured) (basetypes.DynamicType, error) {
	// Convert the unstructured object to a map[string]interface{}
	var dynamicValue basetypes.DynamicType

	return dynamicValue, fmt.Errorf("not implemented yet")
}

func anyToAttrValue(v interface{}) (attr.Value, error) {
	switch v := v.(type) {
	case string:
		return basetypes.NewStringValue(v), nil
	case int64:
		return basetypes.NewNumberValue(big.NewFloat(float64(v))), nil
	case float64:
		return basetypes.NewNumberValue(big.NewFloat(v)), nil
	case bool:
		return basetypes.NewBoolValue(v), nil
	default:
		rValue := reflect.ValueOf(v)
		for rValue.Kind() == reflect.Ptr || rValue.Kind() == reflect.Interface {
			if rValue.IsNil() {
				return basetypes.NewDynamicNull(), nil
			} else {
				rValue = rValue.Elem()
			}
		}
		switch rValue.Kind() {
		case reflect.Map:
			m := make(map[string]attr.Value)
			for _, key := range rValue.MapKeys() {
				keyValue := key.Interface()
				value := rValue.MapIndex(key)
				valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, err
				}
				m[fmt.Sprintf("%v", keyValue)] = valueAttr
			}
			return basetypes.NewObjectValue()
		case reflect.Slice, reflect.Array:
			s := make([]attr.Value, rValue.Len())
			for i := 0; i < rValue.Len(); i++ {
				value := rValue.Index(i)
				valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, err
				}
				s[i] = valueAttr
			}
			return basetypes.NewTupleValue()
		case reflect.Struct:
			s := make(map[string]attr.Value)
			for i := 0; i < rValue.NumField(); i++ {
				field := rValue.Type().Field(i)
				value := rValue.Field(i)
				if err != nil {
					return nil, err
				}
				tags := field.Tag.Get("yaml")
				parts := strings.Split(tags, ",")
				if strings.Contains(tags, "omitempty") {
					if value.IsZero() {
						continue
					}
				}
				valueAttr, err := anyToAttrValue(value.Interface())
				if err != nil {
					return nil, err
				}
				if len(parts) > 0 && parts[0] != "" {
					s[parts[0]] = valueAttr
				} else {
					s[field.Name] = valueAttr
				}
			}
			return basetypes.NewObjectValue()
		default:
			return nil, fmt.Errorf("unsupported type: %s", rValue.Kind())
		}
	}
}
