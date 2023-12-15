package pythondeployer

import (
	"context"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"sync"
)

type Connector struct {
	config        *Config
	logger        log.Logger
	pythonFactory cliwrapper.CliWrapperFactory
	venv          string
	pulled        bool
	lock          *sync.Mutex
}

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {

	pythonCliWrapper, err := c.pythonFactory.Create("", c.logger)
	if err != nil {
		return nil, err
	}

	c.lock.Lock()
	if !c.pulled {
		err := c.pull(ctx, pythonCliWrapper, image)
		if err != nil {
			return nil, err
		}
	}
	c.lock.Unlock()
	//if err := c.pullModule(ctx, pythonCliWrapper, image); err != nil {
	//	return nil, err
	//}

	stdin, stdout, err := pythonCliWrapper.Deploy(image)
	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		wrapper:        pythonCliWrapper,
		containerImage: image,
		config:         c.config,
		stdin:          stdin,
		stdout:         stdout,
		logger:         c.logger,
	}

	return &cliPlugin, nil
}

func (c *Connector) pull(_ context.Context, pythonCliWrapper cliwrapper.CliWrapper, fullModuleName string) error {
	c.logger.Debugf("pull policy: %s", c.config.ModulePullPolicy)
	c.logger.Debugf("pulling module: %s", fullModuleName)
	if err := pythonCliWrapper.PullModule(fullModuleName); err != nil {
		return err
	}
	c.pulled = true
	return nil
}

func (c *Connector) pullModule(_ context.Context, pythonCliWrapper cliwrapper.CliWrapper, fullModuleName string) error {
	c.logger.Debugf("pull policy: %s", c.config.ModulePullPolicy)
	imageExists, err := pythonCliWrapper.ModuleExists(fullModuleName)
	if err != nil {
		return err
	}

	if *imageExists && c.config.ModulePullPolicy == ModulePullPolicyIfNotPresent {
		c.logger.Debugf("module already present skipping pull: %s", fullModuleName)
		return nil
	} else if *imageExists && c.config.ModulePullPolicy == ModulePullPolicyAlways {
		// if the module exists but the policy is to pull always
		// deletes the module venv path and the module is pulled again
		err := pythonCliWrapper.RemoveImage(fullModuleName)
		if err != nil {
			return err
		}
		c.logger.Debugf("module already present, ModulePullPolicy == \"Always\", pulling again...")
	}

	c.logger.Debugf("pulling module: %s", fullModuleName)
	if err := pythonCliWrapper.PullModule(fullModuleName); err != nil {
		return err
	}

	return nil
}
