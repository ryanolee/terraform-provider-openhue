package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openhue/openhue-go"
)

type LightDataSource struct {
	client *openhue.ClientWithResponses
}

type LightDataSourceModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	On   types.Bool   `tfsdk:"on"`
}

func NewLightDataSource() datasource.DataSource {
	return &LightDataSource{}
}

func (d *LightDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_light", req.ProviderTypeName)
}

func (d *LightDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"on": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (d *LightDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model LightDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedApiResp, err := d.client.GetLightsWithResponse(ctx)

	if err != nil {
		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: %s", err.Error()))
		return
	}

	if parsedApiResp.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: %s, %s", parsedApiResp.HTTPResponse.Status, string(parsedApiResp.Body)))
		return
	}

	getLightResponse := *parsedApiResp.JSON200.Data

	// Search for the light in the response
	var targetLight *openhue.LightGet
	for _, light := range getLightResponse {
		if *light.Metadata.Name == model.Name.ValueString() {
			targetLight = &light
			break
		}
	}

	if targetLight == nil {
		lampNames := make([]string, len(getLightResponse))
		for i, light := range getLightResponse {
			lampNames[i] = *light.Metadata.Name
		}

		resp.Diagnostics.AddError("failed to find light", fmt.Sprintf("failed to create light: light %s not found. Available lamps %s", model.Name.String(), strings.Join(lampNames, ", ")))
		return
	}

	// Set the ID of the light
	model.Id = types.StringPointerValue(targetLight.Id)
	model.On = types.BoolPointerValue(targetLight.On.On)

	// Set the model in the response
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

}

func (d *LightDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Configure can be called multiple times (sometimes without provider data)
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*openhue.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("expected openhue.ClientWithResponses", fmt.Sprintf("Expected *openhue.Client, got %T", req.ProviderData))
		return
	}

	d.client = client
}
