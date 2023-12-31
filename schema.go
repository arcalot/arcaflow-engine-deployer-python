package pythondeployer

import (
	"regexp"

	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/pythondeployer/internal/util"
)

// Schema describes the deployment options of the Docker deployment mechanism.
var Schema = schema.NewTypedScopeSchema[*Config](
	schema.NewStructMappedObjectSchema[*Config](
		"Config",
		map[string]*schema.PropertySchema{
			"pythonPath": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, regexp.MustCompile("^.*$")),
				schema.NewDisplayValue(schema.PointerTo("Python path"),
					schema.PointerTo("Provides the path of python executable"), nil),
				false,
				nil,
				nil,
				nil,
				schema.PointerTo(util.JSONEncode("python")),
				nil,
			),
			"workdir": schema.NewPropertySchema(
				schema.NewStringSchema(nil, nil, nil),
				schema.NewDisplayValue(schema.PointerTo("Workdir Path"),
					schema.PointerTo("Provides the directory where the modules virtual environments will be stored"), nil),
				false,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
			"modulePullPolicy": schema.NewPropertySchema(
				schema.NewStringEnumSchema(map[string]*schema.DisplayValue{
					string(ModulePullPolicyAlways):       {NameValue: schema.PointerTo("Always")},
					string(ModulePullPolicyIfNotPresent): {NameValue: schema.PointerTo("If not present")},
				}),
				schema.NewDisplayValue(schema.PointerTo("Module pull policy"), schema.PointerTo("When to pull the python module."), nil),
				false,
				nil,
				nil,
				nil,
				schema.PointerTo(util.JSONEncode(string(ModulePullPolicyIfNotPresent))),
				nil,
			),
		},
	),
)
