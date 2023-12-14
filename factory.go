package pythondeployer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*Config] {
	return &factory{}
}

type factory struct {
}

func (f factory) Name() string {
	return "python"
}

func (f factory) DeploymentType() deployer.DeploymentType {
	return "python"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*Config] {
	return Schema
}

func (f factory) Create(config *Config, logger log.Logger) (deployer.Connector, error) {
	pythonPath, err := binaryCheck(config.PythonPath)
	if err != nil {
		return &Connector{}, fmt.Errorf("python binary check failed with error: %w", err)
	}

	cleanpath := filepath.Clean(config.WorkDir)
	workdir, err := os.MkdirTemp(cleanpath, "")
	if err != nil {
		return nil, fmt.Errorf(
			"error creating temporary directory for python connector (%w)", err)
	}

	python := cliwrapper.NewCliWrapper(pythonPath, workdir, logger)
	return &Connector{
		config:           config,
		logger:           logger,
		pythonCliWrapper: python,
	}, nil
}

// binaryCheck validates there is a python binary in a valid absolute path
func binaryCheck(pythonPath string) (string, error) {
	if pythonPath == "" {
		pythonPath = "python"
	}
	if !filepath.IsAbs(pythonPath) {
		pythonPathAbs, err := exec.LookPath(pythonPath)
		if err != nil {
			return "", fmt.Errorf("pythonPath executable not found in a valid path with error: %w", err)

		}
		pythonPath = pythonPathAbs
	}
	if _, err := os.Stat(pythonPath); err != nil {
		return "", fmt.Errorf("pythons binary not found with error: %w", err)
	}
	return pythonPath, nil
}
