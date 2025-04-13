package provider

import (
	"strings"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/vpath"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type ManifestModel struct {
	Kind     string `tfsdk:"kind"`
	Name     string `tfsdk:"name"`
	Manifest string `tfsdk:"manifest"`
	Source   string `tfsdk:"source"`
}

func ManifestMapSchema() schema.Attribute {
	result := schema.MapNestedAttribute{
		MarkdownDescription: "A Kubernetes manifest. This resource manages the lifecycle of a Kubernetes manifest.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"kind": schema.StringAttribute{
					MarkdownDescription: "The extracted kind of the resource .",
					Computed:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "The extracted name of the resource manifest.",
					Computed:            true,
				},
				"manifest": schema.StringAttribute{
					MarkdownDescription: "The entire manifest ( as yaml text ) of the resource.",
					Computed:            true,
				},
				"source": schema.StringAttribute{
					MarkdownDescription: "The source of the resource manifest.",
					Computed:            true,
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
func ReadManifest(text string) (*ManifestModel, error) {
	var tmp map[string]interface{}
	r := strings.NewReader(text)
	decoder := yaml.NewDecoder(r)
	err := decoder.Decode(&tmp)
	if err != nil {
		return nil, err
	}
	manifest := &ManifestModel{}
	manifest.Kind, err = vpath.Evaluate[string](kindPath, tmp)
	if err != nil {
		return nil, err
	}
	manifest.Name, err = vpath.Evaluate[string](namePath, tmp)
	if err != nil {
		return nil, err
	}
	manifest.Manifest = text
	return manifest, nil
}

func (m *ManifestModel) Key() string {
	return m.Kind + "/" + m.Name
}

var ManifestType = map[string]attr.Type{
	"kind":     types.StringType,
	"name":     types.StringType,
	"manifest": types.StringType,
	"source":   types.StringType,
}
