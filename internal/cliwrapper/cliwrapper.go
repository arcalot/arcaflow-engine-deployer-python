package cliwrapper

import (
	"bytes"
	"errors"
	"fmt"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/models"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type cliWrapper struct {
	pythonFullPath string
	workDir        string
	deployCommand  *exec.Cmd
	logger         log.Logger
	stdErrBuff     bytes.Buffer
}

const RunnableClassifier string = "Arcaflow :: Python Deployer :: Runnable"

func NewCliWrapper(pythonFullPath string,
	workDir string,
	logger log.Logger,
) CliWrapper {
	return &cliWrapper{
		pythonFullPath: pythonFullPath,
		logger:         logger,
		workDir:        workDir,
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
		modulePath = fmt.Sprintf("%s/%s_%s", p.workDir, *pythonModule.ModuleName, *pythonModule.ModuleVersion)
	} else {
		modulePath = fmt.Sprintf("%s/%s_latest", p.workDir, *pythonModule.ModuleName)
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
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return err
	}
	module, err := pythonModule.PipPackageName()
	if err != nil {
		return err
	}
	// create module path
	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return err
	}
	if err := os.Mkdir(*modulePath, os.ModePerm); err != nil {
		return err
	}

	// create venv
	if _, err := os.Stat(p.pythonFullPath); os.IsNotExist(err) {
		return fmt.Errorf("python interpreter not found in %s", p.pythonFullPath)
	}

	cmdCreateVenv := exec.Command(p.pythonFullPath, "-m", "venv", "venv")
	cmdCreateVenv.Dir = *modulePath
	var cmdCreateOut bytes.Buffer
	cmdCreateVenv.Stderr = &cmdCreateOut
	if err := cmdCreateVenv.Run(); err != nil {
		return fmt.Errorf("error while creating venv. Stderr: '%s', err: '%s'", cmdCreateOut.String(), err)
	}

	// pull module
	pipPath := fmt.Sprintf("%s/venv/bin/pip", *modulePath)
	cmdPip := exec.Command(pipPath, "install", *module)
	var cmdPipOut bytes.Buffer
	cmdPip.Stderr = &cmdPipOut
	if err := cmdPip.Run(); err != nil {
		return fmt.Errorf("error while running pip. stderr: '%s', err: '%s'", cmdPipOut.String(), err)
	}
	return nil
}

func (p *cliWrapper) Deploy(fullModuleName string) (io.WriteCloser, io.ReadCloser, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, nil, err
	}
	args := []string{"-m"}
	moduleInvokableName := strings.ReplaceAll(*pythonModule.ModuleName, "-", "_")
	args = append(args, moduleInvokableName)
	args = append(args, "--atp")
	venvPath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return nil, nil, err
	}
	venvPython := fmt.Sprintf("%s/venv/bin/python", *venvPath)

	p.deployCommand = exec.Command(venvPython, args...) //nolint:gosec
	p.deployCommand.Stderr = &p.stdErrBuff
	stdin, err := p.deployCommand.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := p.deployCommand.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := p.deployCommand.Start(); err != nil {
		return nil, nil, fmt.Errorf("error while attempting to run python stderr: '%s', err: '%s'", p.stdErrBuff.String(), err.Error())
	}
	return stdin, stdout, nil
}

func (p *cliWrapper) KillAndClean() error {
	if p.stdErrBuff.Len() > 0 {
		p.logger.Errorf("stderr present after plugin execution: %s", p.stdErrBuff.String())
	} else {
		p.logger.Infof("stderr empty")
	}
	p.logger.Infof("killing config process with pid %d", p.deployCommand.Process.Pid)
	err := p.deployCommand.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}
