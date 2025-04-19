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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesConfigMap{}
var _ resource.ResourceWithImportState = &KubernetesConfigMap{}

func init() {
	// Register the resource with the provider.
	tfprovider.RegisterResource(func() resource.Resource {
		return &KubernetesConfigMap{
			tfTypeNameSuffix: "_config_map",
		}
	})
}

// KubernetesConfigMap defines the resource implementation.
type KubernetesConfigMap struct {
	resourceBase     *tfprovider.BaseResourceHandler[*ConfigMapModel]
	tfTypeNameSuffix string
}

// ConfigMapModel describes the resource data model.
type ConfigMapModel struct {
	MetaData   kresource.MetaData     `tfsdk:"metadata"`
	Immutable  types.Bool             `tfsdk:"immutable"`
	Filenames  *tfprovider.FilesModel `tfsdk:"file_data"`
	Data       types.Map              `tfsdk:"data"`
	BinaryData types.Map              `tfsdk:"binary_data"`
	ApiOptions *tfprovider.APIOptions `tfsdk:"api_options"`

	tfprovider.OutputMetadata
}

func (dmm *ConfigMapModel) BuildManifest(manifest *unstructured.Unstructured) error {
	sm := &tfprovider.StringMap{}
	sm.SetBase64Encoded(false)

	err := sm.AddFileModel(dmm.Filenames)
	if err != nil {
		return err
	}
	err = sm.AddMaps(dmm.Data, dmm.BinaryData)
	if err != nil {
		return err
	}

	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
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
	return nil
}
func (dmm *ConfigMapModel) FromManifest(manifest *unstructured.Unstructured) error {
	//dmm.MetaData.SetFromActual(actual)
	//b, _, _ := unstructured.NestedBool(actual.Object, "immutable")
	//dmm.Immutable = basetypes.NewBoolValue(b)
	dmm.OutputMetadata.FromManifest(manifest)
	return nil
}
func (dmm *ConfigMapModel) GetApiOptions() *tfprovider.APIOptions {
	return dmm.ApiOptions
}
func (dmm *ConfigMapModel) GetResouceKey() (kresource.Key, error) {
	return kresource.Key{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
		MetaData:   dmm.MetaData,
	}, nil
}

func (r *KubernetesConfigMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesConfigMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Kubernetes ConfigMap resource. This resource manages the lifecycle of a Kubernetes configmap.",
		Attributes: map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"api_options": tfprovider.ApiOptionsModelSchema(),
			"file_data":   tfprovider.DefineFileListSchema(false),
			"data": schema.MapAttribute{
				MarkdownDescription: "Data to store in the configmap",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"binary_data": schema.MapAttribute{
				MarkdownDescription: "Base64 encoded data to store in the configmap ( will be base64 decoded )",
				ElementType:         types.StringType,
				Optional:            true,
			},
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

func (r *KubernetesConfigMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.resourceBase = tfprovider.NewCommonHandler[*ConfigMapModel](ctx, req, resp)
}

func (r *KubernetesConfigMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.resourceBase.Create(ctx, req, resp)
}

func (r *KubernetesConfigMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.resourceBase.Read(ctx, req, resp)
}

func (r *KubernetesConfigMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.resourceBase.Update(ctx, req, resp)
}

func (r *KubernetesConfigMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.resourceBase.Delete(ctx, req, resp)
}

func (r *KubernetesConfigMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
