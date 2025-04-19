// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"
	"errors"
	"io"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	resourceBase     *tfprovider.BaseResourceHandler[*GenericResourceModel]
	tfTypeNameSuffix string
}

// GenericResourceModel describes the resource data model.
type GenericResourceModel struct {
	ManifestString types.String `tfsdk:"manifest"`

	ApiOptions *tfprovider.APIOptions `tfsdk:"api_options"`
	tfprovider.OutputMetadata
}

func (grm *GenericResourceModel) BuildManifest(manifest *unstructured.Unstructured) error {
	manifestStr := grm.ManifestString.ValueString()
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
func (grm *GenericResourceModel) FromManifest(manifest *unstructured.Unstructured) error {
	grm.OutputMetadata.FromManifest(manifest)
	//s, _ := kresource.FormatYaml(actual)
	//grm.ManifestString = basetypes.NewStringValue(s)
	return nil
}
func (grm *GenericResourceModel) GetApiOptions() *tfprovider.APIOptions {
	return grm.ApiOptions
}
func (grm *GenericResourceModel) GetResouceKey() (kresource.Key, error) {
	manifest, err := kresource.ParseSingleYamlManifest(grm.ManifestString.ValueString())
	if err != nil {
		return kresource.Key{}, nil
	}

	namespace := manifest.GetNamespace()
	return kresource.Key{
		ApiVersion: manifest.GetAPIVersion(),
		Kind:       manifest.GetKind(),
		MetaData: kresource.MetaData{
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
				Optional:            true,
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
	r.resourceBase = tfprovider.NewCommonHandler[*GenericResourceModel](ctx, req, resp)
}

func (r *KubernetesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.resourceBase.Create(ctx, req, resp)
}

func (r *KubernetesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.resourceBase.Read(ctx, req, resp)
}

func (r *KubernetesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.resourceBase.Update(ctx, req, resp)
}

func (r *KubernetesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.resourceBase.Delete(ctx, req, resp)
}

func (r *KubernetesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
