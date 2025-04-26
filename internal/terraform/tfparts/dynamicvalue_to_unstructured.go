package tfparts

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func DynamicValueToUnstructured(ctx context.Context, value types.Dynamic) (unstructured.Unstructured, error) {
	if value.IsNull() || value.IsUnknown() {
		return unstructured.Unstructured{}, nil
	}
	var u unstructured.Unstructured
	innerValue := value.UnderlyingValue()
	switch innerValue := innerValue.(type) {
	case basetypes.ObjectValue:
		// Convert the object value to a map[string]interface{}
		valueMap := innerValue.Attributes()
		anyMap, err := convertAttrObjectToAnyMap(ctx, valueMap)
		if err != nil {
			return u, fmt.Errorf("failed to convert object value to map: %w", err)
		}
		u.Object = anyMap
	default:
		return u, fmt.Errorf("unsupported type %T", innerValue)
	}
	_ = innerValue

	return u, nil
}

func convertAttrValueToAny(ctx context.Context, value attr.Value) (interface{}, error) {
	switch value := value.(type) {
	case basetypes.StringValue:
		s := value.ValueString()
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			return n, nil
		}
		b, err := strconv.ParseBool(s)
		if err == nil {
			return b, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f, nil
		}
		return s, nil
	case basetypes.BoolValue:
		return value.ValueBool(), nil
	case basetypes.NumberValue:
		bf := value.ValueBigFloat()
		f, _ := bf.Float64()
		return f, nil
	case basetypes.TupleValue:
		typeList := value.ElementTypes(ctx)
		valueList := value.Elements()
		return convertAttrTupleToAnyList(typeList, valueList)
	case basetypes.ObjectValue:
		elements := value.Attributes()
		return convertAttrObjectToAnyMap(ctx, elements)
	default:
		return nil, fmt.Errorf("unsupported type %T", value)
	}
}

func convertAttrObjectToAnyMap(ctx context.Context, attrMap map[string]attr.Value) (map[string]interface{}, error) {
	anyMap := make(map[string]interface{}, len(attrMap))
	for key, value := range attrMap {
		anyValue, err := convertAttrValueToAny(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value for key %s: %w", key, err)
		}
		anyMap[key] = anyValue
	}
	return anyMap, nil
}

func convertAttrTupleToAnyList(typeList []attr.Type, valueList []attr.Value) ([]interface{}, error) {
	if len(typeList) != len(valueList) {
		return nil, fmt.Errorf("typeList and valueList must be the same length")
	}
	anyList := make([]interface{}, len(typeList))
	for i, value := range valueList {
		anyValue, err := convertAttrValueToAny(context.Background(), value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value at index %d: %w", i, err)
		}
		anyList[i] = anyValue
	}
	return anyList, nil
}
