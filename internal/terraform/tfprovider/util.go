package tfprovider

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func MergeResourceAttributes(attrs ...map[string]rschema.Attribute) map[string]rschema.Attribute {
	merged := make(map[string]rschema.Attribute)
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}

func MergeDataAttributes(attrs ...map[string]dschema.Attribute) map[string]dschema.Attribute {
	merged := make(map[string]dschema.Attribute)
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}
