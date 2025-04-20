// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/vpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesResource{}
var _ resource.ResourceWithImportState = &KubernetesResource{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		return &KubernetesResource{
			tfTypeNameSuffix: "_resource",
		}
	})
}

// KubernetesResource defines the resource implementation.
type KubernetesResource struct {
	tfprovider.ResourceBase[*GenericResourceModel]
	tfTypeNameSuffix string
}

// GenericResourceModel describes the resource data model.
type GenericResourceModel struct {
	ManifestString types.String `tfsdk:"manifest"`

	ApiOptions *tfprovider.APIOptionsModel `tfsdk:"api_options"`
	tfprovider.OutputMetadata
}

func (model *GenericResourceModel) BuildManifest(manifest *unstructured.Unstructured) error {
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
func (model *GenericResourceModel) FromManifest(manifest *unstructured.Unstructured) error {
	model.OutputMetadata.FromManifest(manifest)
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
func (model *GenericResourceModel) GetApiOptions() *kresource.APIClientOptions {
	return model.ApiOptions.Options()
}
func (model *GenericResourceModel) GetResouceKey() (kresource.ResourceKey, error) {
	manifest, err := kresource.ParseSingleYamlManifest(model.ManifestString.ValueString())
	if err != nil {
		return kresource.ResourceKey{}, nil
	}

	namespace := manifest.GetNamespace()
	return kresource.ResourceKey{
		ApiVersion: manifest.GetAPIVersion(),
		Kind:       manifest.GetKind(),
		MetaData: kresource.ResourceMetaData{
			Name:        manifest.GetName(),
			Labels:      manifest.GetLabels(),
			Annotations: manifest.GetAnnotations(),
			Namespace:   &namespace,
		},
	}, nil
}

func (r *KubernetesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"manifest": schema.StringAttribute{
				MarkdownDescription: "Manifest to apply",
				Required:            true,
			},
			"api_options": tfprovider.ApiOptionsModelSchema(),
			"resource_version": schema.StringAttribute{
				MarkdownDescription: "The resource version.",
				Computed:            true,
			},
			"uid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the resource.",
				Computed:            true,
			},
			"generation": schema.Int64Attribute{
				MarkdownDescription: "The generation of the resource.",
				Computed:            true,
			},
		},
	}
}

func (r *KubernetesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	provider, ok := req.ProviderData.(*tfprovider.KubernetesResourceProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *tfprovider.KubernetesResourceProvider")
		return
	}
	r.Provider = provider
}

func (r *KubernetesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &GenericResourceModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &GenericResourceModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *KubernetesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &GenericResourceModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *KubernetesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &GenericResourceModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *KubernetesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
