package pythondeployer

import (
	"context"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"os"
)

type Connector struct {
	config           *Config
	logger           log.Logger
	pythonCliWrapper cliwrapper.CliWrapper
}

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {
	if err := c.pullModule(ctx, image); err != nil {
		return nil, err
	}

	stdin, stdout, err := c.pythonCliWrapper.Deploy(image)

	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		wrapper:        c.pythonCliWrapper,
		containerImage: image,
		config:         c.config,
		stdin:          stdin,
		stdout:         stdout,
		logger:         c.logger,
	}

	return &cliPlugin, nil
}

func (c *Connector) pullModule(_ context.Context, fullModuleName string) error {
	c.logger.Debugf("pull policy: %s", c.config.ModulePullPolicy)
	imageExists, err := c.pythonCliWrapper.ModuleExists(fullModuleName)
	if err != nil {
		return err
	}

	if *imageExists && c.config.ModulePullPolicy == ModulePullPolicyAlways {
		// if the module exists but the policy is to pull always
		// deletes the module venv path and the module is pulled again
		modulePath, err := c.pythonCliWrapper.GetModulePath(fullModuleName)
		if err != nil {
			return err
		}

		err = os.RemoveAll(*modulePath)
		if err != nil {
			return err
		}
		c.logger.Debugf("module already present, ModulePullPolicy == \"Always\", pulling again...")
	} else if *imageExists && c.config.ModulePullPolicy == ModulePullPolicyIfNotPresent {
		c.logger.Debugf("module already present skipping pull: %s", fullModuleName)
		return nil
	}

	c.logger.Debugf("pulling module: %s", fullModuleName)
	if err := c.pythonCliWrapper.PullModule(fullModuleName); err != nil {
		return err
	}

	return nil
}
