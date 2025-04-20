// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesSecret{}
var _ resource.ResourceWithImportState = &KubernetesSecret{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		return &KubernetesSecret{
			tfTypeNameSuffix: "_secret",
		}
	})
}

// KubernetesSecret defines the resource implementation.
type KubernetesSecret struct {
	tfprovider.ResourceBase[*SecretModel]
	tfTypeNameSuffix string
}

// SecretModel describes the resource data model.
type SecretModel struct {
	Type          types.String                `tfsdk:"type"`
	MetaData      kresource.MetaData          `tfsdk:"metadata"`
	Immutable     types.Bool                  `tfsdk:"immutable"`
	Filenames     *tfprovider.FilesModel      `tfsdk:"file_data"`
	TextFilenames *tfprovider.FilesModel      `tfsdk:"text_file_data"`
	Data          types.Map                   `tfsdk:"data"`
	TextData      types.Map                   `tfsdk:"text_data"`
	ApiOptions    *tfprovider.APIOptionsModel `tfsdk:"api_options"`

	tfprovider.OutputMetadata
}

func (dmm *SecretModel) BuildManifest(manifest *unstructured.Unstructured) error {
	sm := &kresource.StringMap{}
	sm.SetBase64Encoded(true)

	err := dmm.Filenames.AddToStringMap(sm)
	if err != nil {
		return err
	}
	err = sm.AddMaps(dmm.TextData, dmm.Data)
	if err != nil {
		return err
	}

	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":        dmm.MetaData.Name,
			"labels":      dmm.MetaData.Labels,
			"annotations": dmm.MetaData.Annotations,
		},
		"data": sm.GetUnstructured(),
	})
	if dmm.MetaData.Namespace != nil {
		manifest.SetNamespace(*dmm.MetaData.Namespace)
	}
	immutable := dmm.Immutable.ValueBool()
	if immutable {
		manifest.Object["immutable"] = immutable
	}
	sType := dmm.Type.ValueString()
	if sType == "" {
		sType = "Opaque"
	}
	manifest.Object["type"] = sType
	return nil
}
func (dmm *SecretModel) FromManifest(manifest *unstructured.Unstructured) error {
	//	dmm.MetaData.SetFromActual(actual)
	s, _, _ := unstructured.NestedString(manifest.Object, "type")
	dmm.Type = basetypes.NewStringValue(s)
	dmm.OutputMetadata.FromManifest(manifest)
	return nil
}

func (dmm *SecretModel) GetResouceKey() (kresource.Key, error) {
	return kresource.Key{
		ApiVersion: "v1",
		Kind:       "Secret",
		MetaData:   dmm.MetaData,
	}, nil
}

func (dmm *SecretModel) GetApiOptions() *kresource.APIOptions {
	return dmm.ApiOptions.Options()
}

func (r *KubernetesSecret) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesSecret) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Kubernetes Secret resource. This resource manages the lifecycle of a Kubernetes Secret.",
		Attributes: map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"file_data":      tfprovider.DefineFileListSchema(false),
			"text_file_data": tfprovider.DefineFileListSchema(false),
			"data": schema.MapAttribute{
				MarkdownDescription: "Base64 encoded data to store in the secret",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"text_data": schema.MapAttribute{
				MarkdownDescription: "Plain text data to store in the secret ( will be base64 encoded )",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the secret",
				Optional:            true,
				Computed:            true,
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
		Blocks: map[string]schema.Block{
			"metadata": tfprovider.LongMetadataSchemaBlock(),
		},
	}
}

func (r *KubernetesSecret) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KubernetesSecret) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &SecretModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesSecret) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	plan := &SecretModel{}
	state := &SecretModel{}
	r.ResourceBase.Read(ctx, plan, state, req, resp)
}

func (r *KubernetesSecret) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &SecretModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *KubernetesSecret) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &SecretModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *KubernetesSecret) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
