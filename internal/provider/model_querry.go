package provider

import (
	"fmt"
	"regexp"

	"github.com/davidjspooner/dsvalue/pkg/path"
	"github.com/davidjspooner/dsvalue/pkg/value"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type QuerryModel struct {
	Select  types.String `tfsdk:"select"`
	Match   types.String `tfsdk:"match"`
	Capture types.String `tfsdk:"capture"`

	path   path.Path
	regexp *regexp.Regexp
}

func QuerrySchemaList(required bool) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{

		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"capture": schema.StringAttribute{
					MarkdownDescription: "named value in output",
					Optional:            true,
				},
				"select": schema.StringAttribute{
					MarkdownDescription: "field to check in resource",
					Optional:            true,
				},
				"match": schema.StringAttribute{
					MarkdownDescription: "regex to match",
					Optional:            true,
				},
			},
		},
		Required: required,
	}
}

func (w *QuerryModel) Check(object value.Value) (interface{}, error) {
	var err error

	if w.path == nil {
		w.path, err = path.CompilePath(w.Select.ValueString())
		if err != nil {
			return nil, err
		}

	}
	rx := w.Match.ValueString()
	if w.regexp == nil && rx != "" {
		w.regexp, err = regexp.Compile(rx)
		if err != nil {
			return nil, err
		}
	}

	leaf, err := w.path.EvaluateFor(object)
	if err != nil {
		return nil, err
	}

	v := leaf.WithoutSource()
	if w.regexp != nil {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", v)
		}
		if !w.regexp.MatchString(s) {
			return nil, nil
		}
	}
	return v, nil
}

type QuerryList []*QuerryModel

type CaptureMap map[string]any

func (ql QuerryList) Check(object value.Value) (CaptureMap, error) {
	captured := make(map[string]any)
	for _, w := range ql {
		value, err := w.Check(object)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		name := w.Capture.ValueString()
		if name != "" {
			captured[name] = value
		}
	}
	return captured, nil
}
