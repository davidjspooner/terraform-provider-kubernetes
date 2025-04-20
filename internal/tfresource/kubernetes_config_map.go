// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfresource

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	tfTypeNameSuffix string
	tfprovider.ResourceBase[*ConfigMapModel]
}

// ConfigMapModel describes the resource data model.
type ConfigMapModel struct {
	MetaData   kresource.ResourceMetaData  `tfsdk:"metadata"`
	Immutable  types.Bool                  `tfsdk:"immutable"`
	Files      *tfprovider.FilesModel      `tfsdk:"file_data"`
	Data       types.Map                   `tfsdk:"data"`
	ApiOptions *tfprovider.APIOptionsModel `tfsdk:"api_options"`
	Hashes     types.Map                   `tfsdk:"hashes"`

	tfprovider.OutputMetadata
	values kresource.StringMap
}

func (model *ConfigMapModel) BuildManifest(manifest *unstructured.Unstructured) error {

	err := model.Files.AddToStringMap(&model.values)
	if err != nil {
		return err
	}
	err = tfprovider.AddMapsToStringMap(&model.values, &model.Data, nil)
	if err != nil {
		return err
	}

	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":        model.MetaData.Name,
			"labels":      model.MetaData.Labels,
			"annotations": model.MetaData.Annotations,
		},
		"data": model.values.GetUnstructuredText(),
	})
	if model.MetaData.Namespace != nil {
		manifest.SetNamespace(*model.MetaData.Namespace)
	}
	immutable := model.Immutable.ValueBool()
	if immutable {
		manifest.Object["immutable"] = immutable
	}
	return nil
}

func GetHashes(sm *kresource.StringMap) types.Map {
	hashes := make(map[string]attr.Value)
	if sm == nil {
		return types.MapValueMust(types.StringType, hashes)
	}
	sm.ForEachTextContent(func(k, v string) error {
		hash := md5.Sum([]byte(v))
		s := hex.EncodeToString(hash[:])
		hashes[k] = basetypes.NewStringValue(s)
		return nil
	})
	sm.ForEachBase64Content(func(k, v string) error {
		hash := md5.Sum([]byte(v))
		s := hex.EncodeToString(hash[:])
		hashes[k] = basetypes.NewStringValue(s)
		return nil
	})
	return types.MapValueMust(types.StringType, hashes)
}

func (model *ConfigMapModel) FromManifest(manifest *unstructured.Unstructured) error {
	model.OutputMetadata.FromManifest(manifest)
	model.values.Clear()
	if manifest.Object["data"] != nil {
		for k, v := range manifest.Object["data"].(map[string]interface{}) {
			s := v.(string)
			model.values.AddText(k, s)
		}
	}
	if manifest.Object["binaryData"] != nil {
		for k, v := range manifest.Object["binaryData"].(map[string]interface{}) {
			s := v.(string)
			model.values.AddBase64(k, s)
		}
	}
	model.Hashes = GetHashes(&model.values)

	return nil
}
func (model *ConfigMapModel) GetApiOptions() *kresource.APIClientOptions {
	return model.ApiOptions.Options()
}
func (model *ConfigMapModel) GetResouceKey() (kresource.ResourceKey, error) {
	return kresource.ResourceKey{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
		MetaData:   model.MetaData,
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
			"hashes": schema.MapAttribute{
				MarkdownDescription: "A map of hashes of the data in the secret",
				ElementType:         types.StringType,
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
	provider, ok := req.ProviderData.(*tfprovider.KubernetesResourceProvider)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Type", "Expected provider data to be of type *tfprovider.KubernetesResourceProvider")
		return
	}
	r.Provider = provider
}

func (r *KubernetesConfigMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &ConfigMapModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesConfigMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &ConfigMapModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *KubernetesConfigMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &ConfigMapModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *KubernetesConfigMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &ConfigMapModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *KubernetesConfigMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
