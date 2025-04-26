package tfprovider

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kresource"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"
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
	RegisterResource(func() resource.Resource {
		ks := KubernetesSecret{}
		ks.ResourceBase.tfTypeNameSuffix = "_secret"

		attr := map[string]schema.Attribute{
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the data cannot be updated",
				Optional:            true,
			},
			"file_data":      tfparts.DefineFileListSchema(false),
			"text_file_data": tfparts.DefineFileListSchema(false),
			"data": schema.MapAttribute{
				MarkdownDescription: "Base64 encoded data to store in the secret",
				ElementType:         types.StringType,
				Sensitive:           true,
				Optional:            true,
			},
			"string_data": schema.MapAttribute{
				MarkdownDescription: "Plain text data to store in the secret ( will be base64 encoded )",
				ElementType:         types.StringType,
				Sensitive:           true,
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the secret",
				Optional:            true,
				Computed:            true,
			},
			"hashes": schema.MapAttribute{
				MarkdownDescription: "A map of hashes of the data in the secret",
				ElementType:         types.StringType,
				Computed:            true,
			},
		}
		ks.ResourceBase.schema = schema.Schema{
			Description: "A Kubernetes Secret resource. This resource manages the lifecycle of a Kubernetes Secret.",
			Attributes: MergeResourceAttributes(
				attr,
				tfparts.FetchRequestAttributes(),
				tfparts.ApiOptionsResourceAttributes(),
			),
			Blocks: map[string]schema.Block{
				"metadata": tfparts.LongMetadataSchemaBlock(),
			},
		}
		return &ks
	})
}

// KubernetesSecret defines the resource implementation.
type KubernetesSecret struct {
	ResourceBase[*SecretModel]
}

// SecretModel describes the resource data model.
type SecretModel struct {
	Type          types.String             `tfsdk:"type"`
	MetaData      tfparts.ResourceMetaData `tfsdk:"metadata"`
	Immutable     types.Bool               `tfsdk:"immutable"`
	Filenames     *tfparts.FilesModel      `tfsdk:"file_data"`
	TextFilenames *tfparts.FilesModel      `tfsdk:"text_file_data"`
	Data          types.Map                `tfsdk:"data"`
	StringData    types.Map                `tfsdk:"string_data"`
	ApiOptions    *tfparts.APIOptionsModel `tfsdk:"api_options"`
	Hashes        types.Map                `tfsdk:"hashes"`

	tfparts.FetchMap
	values kresource.StringMap
}

func (model *SecretModel) BuildManifest(manifest *unstructured.Unstructured) error {

	err := model.Filenames.AddToStringMap(&model.values)
	if err != nil {
		return err
	}
	err = tfparts.AddMapsToStringMap(&model.values, &model.StringData, &model.Data)
	if err != nil {
		return err
	}

	manifest.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":        model.MetaData.Name,
			"labels":      model.MetaData.Labels,
			"annotations": model.MetaData.Annotations,
		},
		"data":       model.values.GetUnstructuredBase64(),
		"stringData": model.values.GetUnstructuredText(),
	})
	if model.MetaData.Namespace != nil {
		manifest.SetNamespace(*model.MetaData.Namespace)
	}
	immutable := model.Immutable.ValueBool()
	if immutable {
		manifest.Object["immutable"] = immutable
	}
	sType := model.Type.ValueString()
	if sType == "" {
		sType = "Opaque"
	}
	manifest.Object["type"] = sType
	return nil
}

func (model *SecretModel) UpdateFrom(manifest unstructured.Unstructured) error {
	s, _, _ := unstructured.NestedString(manifest.Object, "type")
	model.Type = basetypes.NewStringValue(s)
	b, _, _ := unstructured.NestedBool(manifest.Object, "immutable")
	model.Immutable = basetypes.NewBoolValue(b)

	model.values.Clear()
	hashes := make(map[string]types.String)
	if manifest.Object["data"] != nil {
		for k, v := range manifest.Object["data"].(map[string]interface{}) {
			s := v.(string)
			model.values.AddBase64(k, s)
			hash := md5.Sum([]byte(s))
			s = hex.EncodeToString(hash[:])
			hashes[k] = basetypes.NewStringValue(s)
		}
	}
	if manifest.Object["stringData"] == nil {
		for k, v := range manifest.Object["stringData"].(map[string]interface{}) {
			s := v.(string)
			model.values.AddText(k, s)
			hash := md5.Sum([]byte(s))
			s = hex.EncodeToString(hash[:])
			hashes[k] = basetypes.NewStringValue(s)
		}
	}
	model.Hashes, _ = types.MapValueFrom(context.Background(), types.StringType, hashes)

	return nil
}

func (model *SecretModel) GetResouceKey() (kresource.ResourceKey, error) {
	k := kresource.ResourceKey{
		ApiVersion: "v1",
		Kind:       "Secret",
	}
	k.MetaData.Name = model.MetaData.Name
	if model.MetaData.Namespace != nil {
		k.MetaData.Namespace = model.MetaData.Namespace
	}
	return k, nil
}

func (r *KubernetesSecret) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *KubernetesSecret) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *KubernetesSecret) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceBase.Configure(ctx, req, resp)
}

func (r *KubernetesSecret) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &SecretModel{}
	r.ResourceBase.Create(ctx, plan, req, resp)
}

func (r *KubernetesSecret) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := &SecretModel{}
	r.ResourceBase.Read(ctx, state, req, resp)
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
