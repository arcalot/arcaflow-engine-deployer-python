package cliwrapper

import (
	"fmt"
	"go.arcalot.io/exex"

	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/models"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type cliWrapper struct {
	pythonFullPath string
	connectorDir   string
	logger         log.Logger
}

const RunnableClassifier string = "Arcaflow :: Python Deployer :: Runnable"

func NewCliWrapper(
	pythonFullPath string,
	connectorDir string,
	logger log.Logger,
) CliWrapper {
	return &cliWrapper{
		pythonFullPath: pythonFullPath,
		logger:         logger,
		connectorDir:   connectorDir,
	}
}

func parseModuleNameGit(fullModuleName string, module *models.PythonModule) {
	nameSourceVersion := strings.Split(fullModuleName, "@")
	source := strings.Replace(nameSourceVersion[1], "git+", "", 1)
	(*module).ModuleName = &nameSourceVersion[0]
	(*module).Repo = &source
	if len(nameSourceVersion) == 3 {
		(*module).ModuleVersion = &nameSourceVersion[2]
	}
}

func parseModuleName(fullModuleName string) (*models.PythonModule, error) {
	pythonModule := models.NewPythonModule(fullModuleName)
	gitRegex := `^[a-zA-Z0-9]+([-_.][a-zA-Z0-9]+)*@git\+https?://[a-zA-Z0-9]+([-._/][a-zA-Z0-9]*)*(@[a-zA-Z0-9]+)?$`
	matchGit, _ := regexp.MatchString(gitRegex, fullModuleName)
	if !matchGit {
		return nil, fmt.Errorf("'%s' has wrong module name format, please use <module-name>@git+<repo_url>[@<commit_sha>]", fullModuleName)
	}
	parseModuleNameGit(fullModuleName, &pythonModule)
	return &pythonModule, nil
}

func (p *cliWrapper) GetModulePath(fullModuleName string) (*string, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, err
	}
	modulePath := filepath.Join(p.connectorDir, *pythonModule.ModuleName)
	if pythonModule.ModuleVersion != nil {
		modulePath += "_" + *pythonModule.ModuleVersion
	} else {
		modulePath += "_latest"
	}
	return &modulePath, err
}

func (p *cliWrapper) ModuleExists(fullModuleName string) (*bool, error) {
	moduleExists := false
	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(*modulePath); os.IsNotExist(err) {
		// false
		return &moduleExists, nil
	}

	moduleExists = true
	return &moduleExists, nil
}

func (p *cliWrapper) PullModule(fullModuleName string) error {
	// every plugin python module gets its own python virtual environment
	err := p.Venv(fullModuleName)
	if err != nil {
		return err
	}

	pipInstallArgs := []string{"install"}

	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return err
	}
	module, err := pythonModule.PipPackageName()
	if err != nil {
		return err
	}
	pipInstallArgs = append(pipInstallArgs, *module)

	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return err
	}

	pipPath := filepath.Join(*modulePath, "venv/bin/pip")
	//cmdPip := exec.Command(pipPath, pipInstallArgs...)
	cmdPip := exex.Command(pipPath, pipInstallArgs...)

	// Make git non-interactive, so that it never prompts for credentials.
	// Otherwise, you can hit edge cases where git will wait for manual
	// authentication causing pip to hang because pip calls `git clone` in
	// a subprocess.
	cmdPip.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmdPip.Output()
	if len(output) > 0 {
		p.logger.Debugf("pip install stdout: %s", output)
	}
	if err != nil {
		return exex.AppendStderr(
			err,
			fmt.Sprintf("error pip installing %s", fullModuleName))
	}
	return nil
}

func (p *cliWrapper) Deploy(fullModuleName string, pluginDirAbsPath string) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exex.Cmd, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	venvPython := filepath.Join(*modulePath, "venv/bin/python")
	args := []string{"-m"}
	moduleInvokableName := strings.ReplaceAll(*pythonModule.ModuleName, "-", "_")
	args = append(args, moduleInvokableName, "--atp")

	deployCommand := exex.Command(venvPython, args...)
	// execute plugin in its own directory in case the plugin needs
	// to write to its current working directory
	deployCommand.Dir = pluginDirAbsPath

	stdin, err := deployCommand.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdout, err := deployCommand.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stderr, err := deployCommand.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = deployCommand.Start()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf(
			"error starting python process for %s (%w)", fullModuleName, err)
	}
	return stdin, stdout, stderr, deployCommand, nil
}

// Venv creates a Python virtual environment for the given
// Python module in its connector's working directory.
func (p *cliWrapper) Venv(fullModuleName string) error {
	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return err
	}
	venvPath := filepath.Join(*modulePath, "venv")
	cmdCreateVenv := exex.Command(p.pythonFullPath, "-m", "venv", "--clear", venvPath)
	output, err := cmdCreateVenv.Output()
	if len(output) > 0 {
		p.logger.Debugf("venv creation stdout %s", output)
	}
	if err != nil {
		return exex.AppendStderr(err,
			fmt.Sprintf("error creating venv for %s", fullModuleName))
	}
	return nil

}
