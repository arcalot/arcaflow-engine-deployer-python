package factory

import (
	"fmt"
	"go.flow.arcalot.io/pythondeployer/internal/config"
	"go.flow.arcalot.io/pythondeployer/internal/connector"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	pythonDeployerSchema "go.flow.arcalot.io/pythondeployer/pkg/schema"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*config.Config] {

	return &factory{connectorCounter: &atomic.Int64{}}
}

type factory struct {
	connectorCounter *atomic.Int64
}

func (f factory) Name() string {
	return "python"
}

func (f factory) DeploymentType() deployer.DeploymentType {
	return "python"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*config.Config] {
	return pythonDeployerSchema.Schema
}

func (f factory) NextConnectorIndex() int {
	f.connectorCounter.Store(f.connectorCounter.Add(1))
	return int(f.connectorCounter.Load())
}

func (f factory) Create(config *config.Config, logger log.Logger) (deployer.Connector, error) {
	pythonPath, err := binaryCheck(config.PythonPath)
	if err != nil {
		return &connector.Connector{}, fmt.Errorf("python binary check failed with error: %w", err)
	}

	connectorFilename := strings.Join([]string{
		"connector",
		strings.Replace(config.PythonSemVer, ".", "-", -1),
		strconv.Itoa(f.NextConnectorIndex())},
		"_")

	absWorkDir, err := filepath.Abs(config.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("error determining absolute path for python deployer's working directory (%w)", err)
	}
	connectorFilepath := filepath.Join(absWorkDir, connectorFilename)
	err = os.MkdirAll(connectorFilepath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating temporary directory for python connector (%w)", err)
	}

	pythonCli := cliwrapper.NewCliWrapper(pythonPath, connectorFilepath, logger)

	cn := connector.NewConnector(
		config, logger, connectorFilepath, &pythonCli)
	return &cn, nil
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
