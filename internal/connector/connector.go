package connector

import (
	"context"
	"fmt"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"go.flow.arcalot.io/pythondeployer/internal/config"
	"os"
	"path/filepath"
	"sync"
)

type Connector struct {
	// the working directory for this connector to persist filesystem
	// side-effects (i.e. install a python module)
	connectorDir string
	// the set of python modules this connector has seen installed
	// into its connector directory
	modules   map[string]struct{}
	pythonCli cliwrapper.CliWrapper
	config    *config.Config
	logger    log.Logger
	lock      sync.Mutex
}

func NewConnector(config *config.Config, logger log.Logger, connectorDir string, pythonCli cliwrapper.CliWrapper) Connector {
	return Connector{
		config:       config,
		logger:       logger,
		connectorDir: connectorDir,
		pythonCli:    pythonCli,
		lock:         sync.Mutex{},
		modules:      make(map[string]struct{}),
	}
}

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	err := c.PullMod(ctx, image, c.pythonCli)
	if err != nil {
		return nil, err
	}

	pluginDirAbspath, err := c.CreatePluginDir("")
	if err != nil {
		return nil, err
	}

	stdin, stdout, deployCommand, stdErrBuff, err := c.pythonCli.Deploy(image, *pluginDirAbspath)
	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		containerImage: image,
		stdin:          stdin,
		stdout:         stdout,
		deployCommand:  deployCommand,
		stdErrBuff:     stdErrBuff,
		logger:         c.logger,
	}

	return &cliPlugin, nil
}

// PullMod synchronizes the creation of Python virtual environments for Python
// module plugins, during the concurrent instantiation of Python cli plugins,
// so that this connector will only pull a module once if it is not present
func (c *Connector) PullMod(_ context.Context, fullModuleName string, pythonCli cliwrapper.CliWrapper) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, cachedPath := c.modules[fullModuleName]
	if !cachedPath {
		modulePresent, err := pythonCli.ModuleExists(fullModuleName)
		if err != nil {
			return fmt.Errorf("error looking for python module (%w)", err)
		}

		if *modulePresent && c.config.ModulePullPolicy == config.ModulePullPolicyIfNotPresent {
			// remember we found the module if someone asks again later
			c.modules[fullModuleName] = struct{}{}
			// file is present, so we do not pull it
			return nil
		}

		// else file is not present, or our pull policy is Always, so let's go
		c.logger.Debugf("pull policy: %s", c.config.ModulePullPolicy)
		c.logger.Debugf("pulling module: %s", fullModuleName)
		if err := pythonCli.PullModule(fullModuleName, string(c.config.ModulePullPolicy)); err != nil {
			return err
		}
		// remember we found the module if someone asks again later
		c.modules[fullModuleName] = struct{}{}
	}
	return nil
}

func (c *Connector) CreatePluginDir(pluginDir string) (*string, error) {
	var workdir string
	var err error
	if pluginDir == "" {
		workdir, err = os.MkdirTemp(c.connectorDir, "")
		if err != nil {
			return nil, err
		}
	} else {
		workdir = filepath.Join(c.connectorDir, filepath.Clean(pluginDir))
		err = os.MkdirAll(workdir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return &workdir, nil
}
