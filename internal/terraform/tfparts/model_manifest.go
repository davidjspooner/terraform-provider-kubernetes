package tfparts

import (
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kresource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ManifestModel struct {
	Kind     string `tfsdk:"kind"`
	Name     string `tfsdk:"name"`
	Manifest string `tfsdk:"manifest"`
	Source   string `tfsdk:"source"`
}

// ReadManifest reads manifest text and returns a Manifest object.
func ReadManifest(text string) (*ManifestModel, error) {
	u, err := kresource.ParseSingleYamlManifest(text)
	if err != nil {
		return nil, err
	}
	manifest := &ManifestModel{}
	manifest.Kind, _, _ = unstructured.NestedString(u.Object, "kind")
	manifest.Name, _, _ = unstructured.NestedString(u.Object, "metadata", "name")
	manifest.Manifest = text
	return manifest, nil
}

func (m *ManifestModel) Key() string {
	return m.Kind + "/" + m.Name
}
