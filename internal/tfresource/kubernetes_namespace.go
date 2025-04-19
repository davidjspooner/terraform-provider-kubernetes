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
	resourceBase     *tfprovider.BaseResourceHandler[*NamespaceModel]
	tfTypeNameSuffix string
}

// NamespaceModel describes the resource data model.
type NamespaceModel struct {
	MetaData kresource.MetaData `tfsdk:"metadata"`

	ApiOptions *tfprovider.APIOptions `tfsdk:"api_options"`
	tfprovider.OutputMetadata
}

func (nsm *NamespaceModel) BuildManifest(manifest *unstructured.Unstructured) error {
	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]interface{}{
			"name": nsm.MetaData.Name,
		},
	})
	if nsm.MetaData.Namespace != nil {
		manifest.SetNamespace(*nsm.MetaData.Namespace)
	}
	labels := nsm.MetaData.Labels
	if labels != nil {
		manifest.SetLabels(labels)
	}
	annotations := nsm.MetaData.Annotations
	if annotations != nil {
		manifest.SetAnnotations(annotations)
	}

	return nil
}
func (nsm *NamespaceModel) FromManifest(manifest *unstructured.Unstructured) error {
	nsm.OutputMetadata.FromManifest(manifest)
	nsm.MetaData.FromManifest(manifest)
	return nil
}
func (nsm *NamespaceModel) GetApiOptions() *tfprovider.APIOptions {
	return nsm.ApiOptions
}

func (nsm *NamespaceModel) GetResouceKey() (kresource.Key, error) {
	return kresource.Key{
		ApiVersion: "v1",
		Kind:       "Namespace",
		MetaData:   nsm.MetaData,
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
	r.resourceBase = tfprovider.NewCommonHandler[*NamespaceModel](ctx, req, resp)
}

func (r *KubernetesNamespace) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.resourceBase.Create(ctx, req, resp)
}

func (r *KubernetesNamespace) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.resourceBase.Read(ctx, req, resp)
}

func (r *KubernetesNamespace) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.resourceBase.Update(ctx, req, resp)
}

func (r *KubernetesNamespace) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.resourceBase.Delete(ctx, req, resp)
}

func (r *KubernetesNamespace) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
