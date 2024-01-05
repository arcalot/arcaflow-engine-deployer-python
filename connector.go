package pythondeployer

import (
	"context"
	"fmt"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"sync"
)

type Connector struct {
	config        *Config
	logger        log.Logger
	pythonFactory cliwrapper.CliWrapperFactory
	venvs         map[string]struct{}
	lock          *sync.Mutex
}

func (c *Connector) Deploy(ctx context.Context, image string) (deployer.Plugin, error) {

	pythonCliWrapper, err := c.pythonFactory.Create("", c.logger)
	if err != nil {
		return nil, err
	}

	err2 := c.pullMod(ctx, image, pythonCliWrapper)
	if err2 != nil {
		return nil, err2
	}

	stdin, stdout, err := pythonCliWrapper.Deploy(image)
	if err != nil {
		return nil, err
	}

	cliPlugin := CliPlugin{
		wrapper:        pythonCliWrapper,
		containerImage: image,
		stdin:          stdin,
		stdout:         stdout,
		logger:         c.logger,
	}

	return &cliPlugin, nil
}

// pullMod synchronizes the creation of Python virtual environments for Python
// module plugins, during the concurrent instantiation of Python cli plugins,
// so that this connector will only pull a module once if it is not present
func (c *Connector) pullMod(_ context.Context, fullModuleName string, pythonCliWrapper cliwrapper.CliWrapper) error {
	c.lock.Lock()
	_, cachedPath := c.venvs[fullModuleName]
	if !cachedPath {
		modulePresent, err := pythonCliWrapper.ModuleExists(fullModuleName)
		if err != nil {
			return fmt.Errorf("error looking for python module (%w)", err)
		}

		if *modulePresent && c.config.ModulePullPolicy == ModulePullPolicyIfNotPresent {
			// remember we found the module if someone asks again later
			c.venvs[fullModuleName] = struct{}{}
			// file is present, so we do not pull it
			return nil
		}

		// else file is not present, or our pull policy is Always, so let's go
		err = pythonCliWrapper.Venv(fullModuleName)
		if err != nil {
			return err
		}
		c.logger.Debugf("pull policy: %s", c.config.ModulePullPolicy)
		c.logger.Debugf("pulling module: %s", fullModuleName)
		if err := pythonCliWrapper.PullModule(fullModuleName, string(c.config.ModulePullPolicy)); err != nil {
			return err
		}
		// remember we found the module if someone asks again later
		c.venvs[fullModuleName] = struct{}{}
	}
	c.lock.Unlock()
	return nil
}
