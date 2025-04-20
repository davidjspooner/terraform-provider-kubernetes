// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesNamespace{}
var _ resource.ResourceWithImportState = &KubernetesNamespace{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		return &KubernetesNamespace{
			tfTypeNameSuffix: "_namespace",
		}
	})
}

// KubernetesNamespace defines the resource implementation.
type KubernetesNamespace struct {
	tfprovider.ResourceBase[*NamespaceModel]
	tfTypeNameSuffix string
}

// NamespaceModel describes the resource data model.
type NamespaceModel struct {
	MetaData kresource.ResourceMetaData `tfsdk:"metadata"`

	ApiOptions *tfprovider.APIOptionsModel `tfsdk:"api_options"`
	tfprovider.OutputMetadata
}

func (model *NamespaceModel) BuildManifest(manifest *unstructured.Unstructured) error {
	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name": model.MetaData.Name,
		},
	})
	if model.MetaData.Namespace != nil {
		manifest.SetNamespace(*model.MetaData.Namespace)
	}
	labels := model.MetaData.Labels
	if labels != nil {
		manifest.SetLabels(labels)
	}
	annotations := model.MetaData.Annotations
	if annotations != nil {
		manifest.SetAnnotations(annotations)
	}

	return nil
}

func (model *NamespaceModel) FromManifest(manifest *unstructured.Unstructured) error {
	model.OutputMetadata.FromManifest(manifest)
	model.MetaData.FromManifest(manifest)
	return nil
}

func (model *NamespaceModel) GetApiOptions() *kresource.APIClientOptions {
	return model.ApiOptions.Options()
}

func (model *NamespaceModel) GetResouceKey() (kresource.ResourceKey, error) {
	return kresource.ResourceKey{
		ApiVersion: "v1",
		Kind:       "Namespace",
		MetaData:   model.MetaData,
	}, nil
}

func (r *KubernetesNamespace) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesNamespace) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Kubernetes Namespace resource. This resource manages the lifecycle of a Kubernetes namespace.",
		Attributes: map[string]schema.Attribute{
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
		Blocks: map[string]schema.Block{
			"metadata": tfprovider.LongMetadataSchemaBlock(),
		},
	}
}

func (r *KubernetesNamespace) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KubernetesNamespace) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &NamespaceModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesNamespace) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &NamespaceModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *KubernetesNamespace) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &NamespaceModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *KubernetesNamespace) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &NamespaceModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *KubernetesNamespace) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
