package pythondeployer

import (
	"fmt"
	"go.flow.arcalot.io/pythondeployer/internal/config"
	"go.flow.arcalot.io/pythondeployer/internal/connector"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
)

// NewFactory creates a new factory for the Docker deployer.
func NewFactory() deployer.ConnectorFactory[*config.Config] {
	connectorCounter := int64(0)
	return &factory{
		connectorCounter: &connectorCounter,
	}
}

type factory struct {
	connectorCounter *int64
}

func (f factory) Name() string {
	return "python"
}

func (f factory) DeploymentType() deployer.DeploymentType {
	return "python"
}

func (f factory) ConfigurationSchema() *schema.TypedScopeSchema[*config.Config] {
	return Schema
}

func (f factory) NextConnectorIndex() int {
	return int(atomic.AddInt64(f.connectorCounter, 1))
}

func (f factory) Create(config *config.Config, logger log.Logger) (deployer.Connector, error) {
	pythonPath, err := binaryCheck(config.PythonPath)
	if err != nil {
		return &connector.Connector{}, fmt.Errorf("python binary check failed with error: %w", err)
	}

	pythonSemver := config.PythonSemVer
	if pythonSemver == "" {
		outputSemver, err := f.parsePythonVersion(pythonPath)
		if err != nil {
			return nil, err
		}
		pythonSemver = *outputSemver
	}

	connectorFilename := strings.Join([]string{
		"connector",
		strings.Replace(pythonSemver, ".", "-", -1),
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
		config, logger, connectorFilepath, pythonCli)
	return &cn, nil
}

// parsePythonVersion function gets the output of a command that asks the
// Python executable for its semantic version string.
func (f factory) parsePythonVersion(pythonPath string) (*string, error) {
	versionCmd := exec.Command(pythonPath, "--version")
	output, err := versionCmd.Output()
	if err != nil {
		return nil, err
	}
	fmt.Printf("%v\n", output)
	re, err := regexp.Compile(`\d+\.\d+\.\d+`)
	if err != nil {
		return nil, err
	}
	found := re.FindString(string(output))
	return &found, nil
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
