package tfprovider

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/davidjspooner/terraform-provider-kubernetes/internal/terraform/tfparts"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSourceKubeManifestFiles{}

func init() {
	// Register the data source with the provider.
	RegisterDataSource(func() datasource.DataSource {
		return &DataSourceKubeManifestFiles{
			tfTypeNameSuffix: "_manifest_files",
		}
	})
}

// DataSourceKubeManifestFiles defines the resource implementation.
type DataSourceKubeManifestFiles struct {
	provider         *KubernetesResourceProvider
	tfTypeNameSuffix string
}

var ManifestType = map[string]attr.Type{
	"kind":     types.StringType,
	"name":     types.StringType,
	"manifest": types.StringType,
	"source":   types.StringType,
}

// FileManifestsModel describes the resource data tfshared.
type FileManifestsModel struct {
	FileNames tfparts.FilesModel `tfsdk:"file_data"`
	Manifests types.Map          `tfsdk:"manifests"`
}

func (r *DataSourceKubeManifestFiles) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.tfTypeNameSuffix
}

func (r *DataSourceKubeManifestFiles) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner manifests",

		Attributes: map[string]schema.Attribute{
			"file_data": tfparts.DefineFileListSchema(true),
			"manifests": schema.MapNestedAttribute{
				MarkdownDescription: "A Kubernetes manifest. This resource manages the lifecycle of a Kubernetes manifest.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							MarkdownDescription: "The extracted kind of the resource .",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The extracted name of the resource manifest.",
							Computed:            true,
						},
						"manifest": schema.StringAttribute{
							MarkdownDescription: "The entire manifest ( as yaml text ) of the resource.",
							Computed:            true,
						},
						"source": schema.StringAttribute{
							MarkdownDescription: "The source of the resource manifest.",
							Computed:            true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func (r *DataSourceKubeManifestFiles) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*KubernetesResourceProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

type manifestWithLineNumber struct {
	lineNumber int
	manifest   string
}

func (r *DataSourceKubeManifestFiles) readManifestsFromReader(_ context.Context, reader io.Reader, _ *FileManifestsModel) []manifestWithLineNumber {
	scanner := bufio.NewScanner(reader)
	var manifest bytes.Buffer
	var manifests []manifestWithLineNumber
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		if line == "---" {
			if manifest.Len() >= 0 {
				manifests = append(manifests, manifestWithLineNumber{
					lineNumber: lineNumber,
					manifest:   manifest.String(),
				})
			}
			manifest.Reset()
			continue
		}
		manifest.WriteString(line)
		manifest.WriteString("\n")
	}
	if manifest.Len() >= 0 {
		manifests = append(manifests, manifestWithLineNumber{
			lineNumber: lineNumber,
			manifest:   manifest.String(),
		})
	}
	return manifests
}

func (r *DataSourceKubeManifestFiles) expandTemplate(templateString string, values map[string]string) (string, error) {
	t := template.New("file")
	t, err := t.Parse(templateString)
	if err != nil {
		return "", err
	}
	var expanded bytes.Buffer
	err = t.Execute(&expanded, values)
	if err != nil {
		return "", err
	}
	return expanded.String(), nil
}

func (r *DataSourceKubeManifestFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileManifestsModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	sm := &kube.StringMap{}
	config.FileNames.AddToStringMap(sm)

	var values map[string]string
	var diags diag.Diagnostics
	diags = config.FileNames.Values.ElementsAs(ctx, &values, false)
	resp.Diagnostics.Append(diags...)

	allmanifests := make(map[string]attr.Value)

	if !resp.Diagnostics.HasError() {
		sm.ForEachTextContent(func(filename string, content string) error {
			var err error
			if len(values) > 0 {
				content, err = r.expandTemplate(content, values)
				if err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Failed to expand file %s", filename),
						err.Error(),
					)
					return err
				}
			}

			sr := strings.NewReader(content)
			manifests := r.readManifestsFromReader(ctx, sr, &config)
			for _, manifestWithLineNumber := range manifests {

				manifest, err := tfparts.ReadManifest(manifestWithLineNumber.manifest)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						resp.Diagnostics.AddError(
							fmt.Sprintf("Failed to parse yaml from file %s line %d", filename, manifestWithLineNumber.lineNumber),
							err.Error(),
						)
					}
					continue
				}
				manifest.Source = fmt.Sprintf("%s:%d", filename, manifestWithLineNumber.lineNumber)
				key := manifest.Key()
				allmanifests[key], diags = types.ObjectValueFrom(ctx, ManifestType, manifest)
			}
			return nil
		})
	}

	// Copy allmanifests elements to config.Manifests
	config.Manifests, diags = basetypes.NewMapValue(types.ObjectType{AttrTypes: ManifestType}, allmanifests)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
