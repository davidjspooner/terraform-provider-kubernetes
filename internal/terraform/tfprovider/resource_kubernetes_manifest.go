package tfprovider

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/vpath"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesManifest{}
var _ resource.ResourceWithImportState = &KubernetesManifest{}

func init() {
	// Register the resource with the provider.
	RegisterResource(func() resource.Resource {
		r := KubernetesManifest{
			tfTypeNameSuffix: "_manifest",
		}
		attr := map[string]schema.Attribute{
			"manifest_content": schema.StringAttribute{
				MarkdownDescription: "Manifest to apply",
				Optional:            true,
			},
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

// KubernetesManifest defines the resource implementation.
type KubernetesManifest struct {
	ResourceBase[*ManifestResourceModel]
	tfTypeNameSuffix string
}

// ManifestResourceModel describes the resource data model.
type ManifestResourceModel struct {
	ManifestString types.String  `tfsdk:"manifest_content"`
	Manifest       types.Dynamic `tfsdk:"manifest"`

	ApiOptions *tfparts.APIOptionsModel `tfsdk:"api_options"`
	tfparts.FetchMap
}

func (model *ManifestResourceModel) BuildManifest(manifest *unstructured.Unstructured) error {
	manifestStr := model.ManifestString.ValueString()
	var err error
	*manifest, err = kresource.ParseSingleYamlManifest(manifestStr)
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
	previousManifest, err := kresource.ParseSingleYamlManifest(model.ManifestString.ValueString())
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
		yamlData := []byte("changed....")

		model.ManifestString = basetypes.NewStringValue(string(yamlData))
	}

	return nil
}
func (model *ManifestResourceModel) GetResouceKey() (kresource.ResourceKey, error) {
	manifest, err := kresource.ParseSingleYamlManifest(model.ManifestString.ValueString())
	if err != nil {
		return kresource.ResourceKey{}, nil
	}

	namespace := manifest.GetNamespace()
	name := manifest.GetName()
	if name == "" {
		return kresource.ResourceKey{}, fmt.Errorf("name is empty")
	}
	k := kresource.ResourceKey{
		ApiVersion: manifest.GetAPIVersion(),
		Kind:       manifest.GetKind(),
	}
	k.MetaData.Name = name
	if namespace != "" {
		k.MetaData.Namespace = &namespace
	}
	return k, nil
}

func (r *KubernetesManifest) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesManifest) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *KubernetesManifest) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceBase.Configure(ctx, req, resp)
}

func (r *KubernetesManifest) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &ManifestResourceModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesManifest) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &ManifestResourceModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *KubernetesManifest) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &ManifestResourceModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *KubernetesManifest) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &ManifestResourceModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *KubernetesManifest) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
