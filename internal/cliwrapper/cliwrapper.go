package cliwrapper

import (
	"bytes"
	"errors"
	"fmt"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/models"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type cliWrapper struct {
	pythonFullPath string
	pluginDir      string
	connectorDir   string
	logger         log.Logger
}

const RunnableClassifier string = "Arcaflow :: Python Deployer :: Runnable"

func NewCliWrapper(
	pythonFullPath string,
	connectorDir string,
	workDir string,
	logger log.Logger,
) CliWrapper {
	return &cliWrapper{
		pythonFullPath: pythonFullPath,
		logger:         logger,
		connectorDir:   connectorDir,
		pluginDir:      workDir,
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
	gitRegex := "^([a-zA-Z0-9]+[-_\\.]*)+@git\\+http[s]{0,1}:\\/\\/([a-zA-Z0-9]+[-.\\/]*)+(@[a-z0-9]+)*$"
	matchGit, _ := regexp.MatchString(gitRegex, fullModuleName)
	if !matchGit {
		return nil, errors.New("wrong module name format, please use <module-name>@git+<repo_url>[@<commit_sha>]")
	}
	parseModuleNameGit(fullModuleName, &pythonModule)
	return &pythonModule, nil
}

func (p *cliWrapper) GetModulePath(fullModuleName string) (*string, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, err
	}
	modulePath := ""
	if pythonModule.ModuleVersion != nil {
		modulePath = fmt.Sprintf("%s/%s_%s", p.connectorDir, *pythonModule.ModuleName, *pythonModule.ModuleVersion)
	} else {
		modulePath = fmt.Sprintf("%s/%s_latest", p.connectorDir, *pythonModule.ModuleName)
	}
	return &modulePath, err
}

func (p *cliWrapper) PullModule(fullModuleName string, pullPolicy string) error {
	pipInstallArgs := []string{"install"}

	if pullPolicy == "Always" {
		pipInstallArgs = append(pipInstallArgs, "--force-reinstall")
	}

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
	cmdPip := exec.Command(pipPath, pipInstallArgs...)

	// Make git non-interactive, so that it never prompts for credentials.
	// Otherwise, you can hit edge cases where git will wait for manual
	// authentication causing pip to hang b/c pip calls `git clone` in a
	// subprocess.
	cmdPip.Env = append(cmdPip.Environ(), "GIT_TERMINAL_PROMPT=0")

	var cmdPipStderr bytes.Buffer
	cmdPip.Stderr = &cmdPipStderr

	err = cmdPip.Run()
	if cmdPipStderr.Len() > 0 {
		p.logger.Warningf("pip install stderr: %s", cmdPipStderr.String())
	}
	if err != nil {
		return fmt.Errorf("error pip installing '%w'", err)
	}
	return nil
}

func (p *cliWrapper) Deploy(fullModuleName string) (*CliWrapperPlugin, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, err
	}

	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return nil, err
	}
	venvPython := filepath.Join(*modulePath, "venv/bin/python")
	args := []string{"-m"}
	moduleInvokableName := strings.ReplaceAll(*pythonModule.ModuleName, "-", "_")
	args = append(args, moduleInvokableName, "--atp")

	deployCommand := exec.Command(venvPython, args...) //nolint:gosec
	deployCommand.Stderr = &bytes.Buffer{}

	// execute plugin in its own directory in case the plugin needs
	// to write to its current working directory
	deployCommand.Dir = p.pluginDir

	stdin, err := deployCommand.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := deployCommand.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = deployCommand.Start()
	if err != nil {
		return nil, err
	}
	return &CliWrapperPlugin{
		containerImage: fullModuleName,
		proc:           deployCommand.Process,
		logger:         p.logger,
		stdin:          stdin,
		stdout:         stdout,
		stdErrBuff:     bytes.Buffer{},
	}, nil
}

func (p *cliWrapper) Venv(fullModuleName string) error {
	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return err
	}
	venv_path := filepath.Join(*modulePath, "venv")
	cmdCreateVenv := exec.Command(p.pythonFullPath, "-m", "venv", venv_path)
	var cmdCreateOut bytes.Buffer
	cmdCreateVenv.Stderr = &cmdCreateOut
	err = cmdCreateVenv.Run()
	if cmdCreateOut.Len() > 0 {
		p.logger.Warningf("venv creation stderr %s", cmdCreateOut.String())
	}
	if err != nil {
		return fmt.Errorf("error creating venv '%w'", err)
	}
	return nil
}
