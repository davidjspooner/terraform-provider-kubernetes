package pmodel

import (
	"strings"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/vpath"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Kind    string `tfsdk:"kind"`
	Name    string `tfsdk:"name"`
	Content string `tfsdk:"content"`
}

func ManifestMapSchema() schema.Attribute {
	result := schema.MapNestedAttribute{
		MarkdownDescription: "A Kubernetes manifest. This resource manages the lifecycle of a Kubernetes manifest.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"kind": schema.StringAttribute{
					MarkdownDescription: "The kind of the resource .",
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "The name of the resource manifest.",
				},
				"content": schema.StringAttribute{
					MarkdownDescription: "The entire content of the resource manifest",
				},
			},
		},
		Computed: true,
	}
	return result
}

var kindPath = vpath.MustCompile("kind")
var namePath = vpath.MustCompile("metadata.name")

// ReadManifest reads manifest text and returns a Manifest object.
func ReadManifest(text string) (*Manifest, error) {
	var tmp map[string]interface{}
	r := strings.NewReader(text)
	decoder := yaml.NewDecoder(r)
	err := decoder.Decode(&tmp)
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{}
	manifest.Kind, err = vpath.Evaluate[string](namePath, tmp)
	if err != nil {
		return nil, err
	}
	manifest.Name, err = vpath.Evaluate[string](kindPath, tmp)
	if err != nil {
		return nil, err
	}
	manifest.Content = text
	return manifest, nil
}

func (m *Manifest) Key() string {
	return m.Kind + "/" + m.Name
}

var ManifestType = map[string]attr.Type{
	"kind":    types.StringType,
	"name":    types.StringType,
	"content": types.StringType,
}
