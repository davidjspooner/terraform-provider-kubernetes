package tfprovider

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/vpath"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceKubeResource{}
var _ resource.ResourceWithImportState = &ResourceKubeResource{}

func init() {
	// Register the resource with the provider.
	RegisterResource(func() resource.Resource {
		r := ResourceKubeResource{}
		r.ResourceBase.tfTypeNameSuffix = "_applied_manifest"
		attr := map[string]schema.Attribute{
			"manifest": schema.DynamicAttribute{
				MarkdownDescription: "Manifest to apply",
				Optional:            true,
			},
		}

		r.schema = schema.Schema{
			// This description is used by the documentation generator and the language server.
			MarkdownDescription: "Generic Manifest resource. This resource allows you to apply any Kubernetes manifest to the cluster. ",

			Attributes: MergeResourceAttributes(
				attr,
				tfparts.FetchRequestAttributes(),
				tfparts.ApiOptionsResourceAttributes(),
			),
		}

		return &r
	})
}

// ResourceKubeResource defines the resource implementation.
type ResourceKubeResource struct {
	ResourceBase[*ManifestResourceModel]
}

// ManifestResourceModel describes the resource data model.
type ManifestResourceModel struct {
	Manifest types.Dynamic `tfsdk:"manifest"`

	tfparts.APIOptionsModel
	tfparts.FetchMap
}

func (model *ManifestResourceModel) BuildManifest(manifest *unstructured.Unstructured) error {
	var err error
	ctx := context.Background()
	*manifest, err = tfparts.DynamicValueToUnstructured(ctx, model.Manifest)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// This is a special case where the manifest is empty
			// and we want to return a nil error.
			*manifest = unstructured.Unstructured{}
			return nil
		}
		return err
	}
	return nil
}
func (model *ManifestResourceModel) UpdateFrom(manifest unstructured.Unstructured) error {
	ctx := context.Background()
	previousManifest, err := tfparts.DynamicValueToUnstructured(ctx, model.Manifest)
	if err != nil {
		return err
	}

	changed := false
	diffHandler := func(path string, left, right interface{}) error {
		if left != nil {
			changed = true
			return fmt.Errorf("left: %v, right: %v", left, right)
		}
		return nil
	}

	err = vpath.FindDifferences("", previousManifest.Object, manifest.Object, vpath.DifferenceHandlerFunc(diffHandler))
	_ = err

	if changed {
		// If the manifest has changed, we need to update the manifest string
		// to reflect the new state.
		// This is a bit of a hack, but we need to do this to ensure that the
		// resource is updated correctly.

		//yamlData, err := yaml.Marshal(manifest.Object)
		//if err != nil {
		//	return err
		//}
		_ = 1
		changed = false

	}

	return nil
}
func (model *ManifestResourceModel) GetResouceKey() (kube.ResourceKey, error) {
	ctx := context.Background()
	manifest, err := tfparts.DynamicValueToUnstructured(ctx, model.Manifest)
	if err != nil {
		return kube.ResourceKey{}, nil
	}

	namespace := manifest.GetNamespace()
	name := manifest.GetName()
	if name == "" {
		return kube.ResourceKey{}, fmt.Errorf("name is empty")
	}
	k := kube.ResourceKey{
		ApiVersion: manifest.GetAPIVersion(),
		Kind:       manifest.GetKind(),
	}
	k.Metadata.Name = name
	if namespace != "" {
		k.Metadata.Namespace = &namespace
	}
	return k, nil
}

func (r *ResourceKubeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	r.ResourceBase.Metadata(ctx, req, resp)
}

func (r *ResourceKubeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *ResourceKubeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceBase.Configure(ctx, req, resp)
}

func (r *ResourceKubeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &ManifestResourceModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *ResourceKubeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &ManifestResourceModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *ResourceKubeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &ManifestResourceModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *ResourceKubeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &ManifestResourceModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *ResourceKubeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
