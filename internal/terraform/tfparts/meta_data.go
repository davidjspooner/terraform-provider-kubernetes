package tfparts

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type ResourceMetaData struct {
	Name        string            `yaml:"name" tfsdk:"name"`
	Namespace   *string           `yaml:"namespace,omitempty" tfsdk:"namespace"`
	Labels      map[string]string `yaml:"labels,omitempty" tfsdk:"labels"`
	Annotations map[string]string `yaml:"annotations,omitempty" tfsdk:"annotations"`
}

func (m *ResourceMetaData) UpdateFrom(manifest unstructured.Unstructured) {
	m.Name = manifest.GetName()
	s, _, _ := unstructured.NestedString(manifest.Object, "metadata", "namespace")
	if s != "" {
		m.Namespace = &s
	} else {
		m.Namespace = nil
	}
	m.Labels = manifest.GetLabels()
	m.Annotations = manifest.GetAnnotations()
}
