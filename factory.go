package pythondeployer

import (
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/cliwrapper"
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/config"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*config.Config] {
	return &factory{}
}

type factory struct {
}

func (f factory) ID() string {
	return "python"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*config.Config] {
	return Schema
}

func (f factory) Create(config *config.Config, logger log.Logger) (deployer.Connector, error) {
	python := cliwrapper.NewCliWrapper(config.PythonPath, config.WorkDir, logger)
	return &Connector{
		config:           config,
		logger:           logger,
		pythonCliWrapper: python,
	}, nil
}
