package pythondeployer

import (
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/config"
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/util"
	"go.flow.arcalot.io/pluginsdk/schema"
	"os/exec"
	"regexp"
)

func pythonGetDefaultPath() string {
	path, err := exec.LookPath("python")
	if err != nil {
		panic("python binary not found in $PATH, please provide it in configuration")
	}
	return path
}

// Schema describes the deployment options of the Docker deployment mechanism.
var Schema = schema.NewTypedScopeSchema[*config.Config](
	schema.NewStructMappedObjectSchema[*config.Config](
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
				schema.PointerTo(util.JSONEncode(pythonGetDefaultPath())),
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
					string(config.ModulePullPolicyAlways):       {NameValue: schema.PointerTo("Always")},
					string(config.ModulePullPolicyIfNotPresent): {NameValue: schema.PointerTo("If not present")},
				}),
				schema.NewDisplayValue(schema.PointerTo("Module pull policy"), schema.PointerTo("When to pull the python module."), nil),
				false,
				nil,
				nil,
				nil,
				schema.PointerTo(util.JSONEncode(string(config.ModulePullPolicyIfNotPresent))),
				nil,
			),
		},
	),
)
