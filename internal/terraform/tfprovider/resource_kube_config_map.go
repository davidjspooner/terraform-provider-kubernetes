package tfprovider

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceKubeConfigMap{}
var _ resource.ResourceWithImportState = &ResourceKubeConfigMap{}

func init() {
	// Register the resource with the provider.
	RegisterResource(func() resource.Resource {
		r := &ResourceKubeConfigMap{}
		r.tfTypeNameSuffix = "_config_map"
		attr := map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"file_data": tfparts.DefineFileListSchema(false),
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
			"hashes": schema.MapAttribute{
				MarkdownDescription: "A map of hashes of the data in the secret",
				ElementType:         types.StringType,
				Computed:            true,
			},
		}
		r.schema = schema.Schema{
			Description: "A Kubernetes ConfigMap resource. This resource manages the lifecycle of a Kubernetes configmap.",
			Attributes: MergeResourceAttributes(
				attr,
				tfparts.FetchRequestAttributes(),
				tfparts.ApiOptionsResourceAttributes(),
			),
			Blocks: map[string]schema.Block{
				"metadata": tfparts.LongMetadataSchemaBlock(),
			},
		}
		return r
	})
}

// ResourceKubeConfigMap defines the resource implementation.
type ResourceKubeConfigMap struct {
	ResourceBase[*ConfigMapModel]
}

// ConfigMapModel describes the resource data model.
type ConfigMapModel struct {
	MetaData   tfparts.ResourceMetaData `tfsdk:"metadata"`
	Immutable  types.Bool               `tfsdk:"immutable"`
	Files      *tfparts.FilesModel      `tfsdk:"file_data"`
	Data       types.Map                `tfsdk:"data"`
	ApiOptions *tfparts.APIOptionsModel `tfsdk:"api_options"`
	Hashes     types.Map                `tfsdk:"hashes"`

	tfparts.FetchMap
	values kube.StringMap
}

func (model *ConfigMapModel) BuildManifest(manifest *unstructured.Unstructured) error {

	err := model.Files.AddToStringMap(&model.values)
	if err != nil {
		return err
	}
	err = tfparts.AddMapsToStringMap(&model.values, &model.Data, nil)
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

func GetHashes(sm *kube.StringMap) types.Map {
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

func (model *ConfigMapModel) UpdateFrom(manifest unstructured.Unstructured) error {
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

func (model *ConfigMapModel) GetResouceKey() (kube.ResourceKey, error) {
	k := kube.ResourceKey{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
	}
	k.MetaData.Name = model.MetaData.Name
	if model.MetaData.Namespace != nil {
		k.MetaData.Namespace = model.MetaData.Namespace
	}
	return k, nil
}

func (r *ResourceKubeConfigMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *ResourceKubeConfigMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *ResourceKubeConfigMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceBase.Configure(ctx, req, resp)
}

func (r *ResourceKubeConfigMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &ConfigMapModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *ResourceKubeConfigMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &ConfigMapModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
}

func (r *ResourceKubeConfigMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &ConfigMapModel{}
	r.ResourceBase.Update(ctx, plan, req, resp)
}

func (r *ResourceKubeConfigMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &ConfigMapModel{}
	r.ResourceBase.Delete(ctx, state, req, resp)
}

func (r *ResourceKubeConfigMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Import not supported", "This resource does not support import")
}
