package provider

import (
	"fmt"
	"regexp"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/vpath"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type QuerryModel struct {
	Select  types.String `tfsdk:"select"`
	Match   types.String `tfsdk:"match"`
	Capture types.String `tfsdk:"capture"`

	path   vpath.Path
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

func (w *QuerryModel) Check(object interface{}) (interface{}, error) {
	var err error

	if w.path == nil {
		w.path, err = vpath.Compile(w.Select.ValueString())
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

	v, err := w.path.EvaluateFor(object)
	if err != nil {
		return nil, err
	}

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

func (ql QuerryList) Check(object interface{}) (CaptureMap, error) {
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
