package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openhue/openhue-go"

	"github.com/ryanolee/terraform-provider-talk/internal/config"
	"github.com/ryanolee/terraform-provider-talk/internal/datasources"
	"github.com/ryanolee/terraform-provider-talk/internal/functions"
	"github.com/ryanolee/terraform-provider-talk/internal/resources"
)

type (
	OpenhueProvider struct {
		Version string
	}

	OpenhueProviderModel struct {
		BridgeIp     types.String `tfsdk:"bridge_ip"`
		BridgeApiKey types.String `tfsdk:"bridge_api_key"`
		Cache        types.Bool   `tfsdk:"cache"`
	}
)

func New() provider.Provider {
	return &OpenhueProvider{}
}

func (p *OpenhueProvider) Metadata(_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openhue"
}

func (p *OpenhueProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bridge_ip": schema.StringAttribute{
				Optional:    true,
				Description: "The IP address of the Hue bridge. If not provided, the plugin will attempt to discover one on the local network.",
			},
			"bridge_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The API key for the Hue bridge. If not provided, the plugin will attempt to create one but you will need to press the link button on the Hue bridge to authenticate.",
			},
			"cache": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to cache the bridge IP and API key in the provider. If true, the plugin will not attempt to discover the bridge IP or create an API key on subsequent runs.",
			},
		},
	}
}

func (p *OpenhueProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model OpenhueProviderModel

	resp.Diagnostics.Append(
		req.Config.Get(
			ctx,
			&model,
		)...,
	)

	authConfig, err := config.GetAuthConfig(ctx, model.BridgeIp.ValueString(), model.BridgeApiKey.ValueString(), model.Cache.ValueBool())
	ctx = tflog.SetField(ctx, "bridge_ip", model.BridgeIp.String())
	ctx = tflog.SetField(ctx, "bridge_api_key", model.BridgeApiKey.String())
	ctx = tflog.MaskAllFieldValuesStrings(ctx, "bridge_api_key")

	tflog.Info(ctx, fmt.Sprintf("Configuring provider with bridge_ip: %s", model.BridgeIp.String()))

	if err != nil {
		resp.Diagnostics.AddError("failed to get auth config", fmt.Sprintf("failed to get auth config: %s", err.Error()))
		return
	}

	// Create a new client with the provided bridge IP and API key
	client, err := openhue.NewClientWithResponses(fmt.Sprintf("https://%s", authConfig.BridgeIp), openhue.WithRequestEditorFn(
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("hue-application-key", authConfig.BridgeApiKey)
			return nil
		},
	))

	if err != nil {
		resp.Diagnostics.AddError("failed to create home", spew.Sprintf("failed to create home: %s", err.Error()))
		return
	}

	// @todo - Find a way to isolate this to the clinet
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Handoff "Client" to the provider
	resp.DataSourceData = client
	resp.ResourceData = client
	resp.EphemeralResourceData = client
}

func (p *OpenhueProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewRoom,
		resources.NewLight,
	}
}

func (p *OpenhueProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewLightDataSource,
	}
}

// With the provider.Provider implementation
func (p *OpenhueProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		functions.NewHextod65Function,
	}
}
