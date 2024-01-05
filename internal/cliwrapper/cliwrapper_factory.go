package cliwrapper

import (
	"go.arcalot.io/log/v2"
	"os"
	"path/filepath"
)

type CliWrapperFactory struct {
	connectorDirAbsPath string
	pythonAbsPath       string
	logger              log.Logger
}

func NewCliWrapperFactory(pythonPath string, connectorDir string, logger log.Logger) (CliWrapperFactory, error) {
	connectorDirAbsPath, err := filepath.Abs(filepath.Clean(connectorDir))
	if err != nil {
		return CliWrapperFactory{}, err
	}
	pythonAbsPath, err := filepath.Abs(filepath.Clean(pythonPath))
	if err != nil {
		return CliWrapperFactory{}, err
	}
	return CliWrapperFactory{
		connectorDirAbsPath: connectorDirAbsPath,
		pythonAbsPath:       pythonAbsPath,
		logger:              logger,
	}, nil
}

// Create instantiates a CliWrapper by providing or creating an empty, unique
// directory for the plugin to execute within as its current working
// directory.
func (f CliWrapperFactory) Create(pluginDir string, logger log.Logger) (CliWrapper, error) {
	var workdir string
	var err error
	if pluginDir == "" {
		workdir, err = os.MkdirTemp(f.connectorDirAbsPath, "")
		if err != nil {
			return nil, err
		}
	} else {
		workdir = filepath.Join(f.connectorDirAbsPath, filepath.Clean(pluginDir))
		err = os.MkdirAll(workdir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return NewCliWrapper(
		f.pythonAbsPath, f.connectorDirAbsPath, logger), nil
}
