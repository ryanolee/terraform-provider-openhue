package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lucasb-eyer/go-colorful"
)

var _ function.Function = &Hextod65Function{}

type (
	Hextod65Function struct{}

	Hextod65FunctionReturn struct {
		X float32 `tfsdk:"x"`
		Y float32 `tfsdk:"y"`
		Z float32 `tfsdk:"z"`
	}
)

func NewHextod65Function() function.Function {
	return &Hextod65Function{}
}

func (f *Hextod65Function) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hextod65"
}

func (f *Hextod65Function) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Converts a hex color to the D65 color space.",
		Description: "Given a string value, returns a struct containing x, y and z properties.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "hex_color",
				Description: "The hex color to convert.",
			},
		},
		Return: function.ObjectReturn{
			AttributeTypes: map[string]attr.Type{
				"x": types.Float32Type,
				"y": types.Float32Type,
				"z": types.Float32Type,
			},
		},
	}
}

func (f *Hextod65Function) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hexColor string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &hexColor))

	color, err := colorful.Hex(hexColor)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert hex color"))
		return
	}
	x, y, z := color.Xyy()

	returnValue := &Hextod65FunctionReturn{
		X: float32(x),
		Y: float32(y),
		Z: float32(z),
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &returnValue))
}
