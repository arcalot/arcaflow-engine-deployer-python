package pythondeployer

import (
	"context"
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/cliwrapper"
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/config"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"os"
)

type Connector struct {
	config           *config.Config
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

	if *imageExists && c.config.ModulePullPolicy == config.ModulePullPolicyAlways {
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
	} else if *imageExists && c.config.ModulePullPolicy == config.ModulePullPolicyIfNotPresent {
		c.logger.Debugf("module already present skipping pull: %s", fullModuleName)
		return nil
	}
	// check for module compatibility, this is done once if the pull policy is
	// ModulePullPolicyIfNotPresent or per every run if the policy is ModulePullPolicyAlways
	c.logger.Debugf("checking module compatibility: %s", fullModuleName)
	err = c.pythonCliWrapper.CheckModuleCompatibility(fullModuleName)
	if err != nil && !c.config.OverrideModuleCompatibility {
		return err
	}

	if err != nil && !c.config.OverrideModuleCompatibility {
		c.logger.Warningf("you're running an incompatible module overriding compatibility checks," +
			"this action may lead to engine crashes, be careful")
	}

	c.logger.Debugf("pulling module: %s", fullModuleName)
	if err := c.pythonCliWrapper.PullModule(fullModuleName); err != nil {
		return err
	}

	return nil
}
