package cliwrapper

import (
	"bytes"
	"errors"
	"fmt"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/models"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type cliWrapper struct {
	pythonFullPath string
	pluginDir      string
	connectorDir   string
	deployCommand  *exec.Cmd
	logger         log.Logger
	stdErrBuff     *bufferThreadSafe
}

type bufferThreadSafe struct {
	b    bytes.Buffer
	lock *sync.Mutex
}

func (b *bufferThreadSafe) Len() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.b.Len()
}

func (b *bufferThreadSafe) Write(p []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.b.Write(p)
}

func (b *bufferThreadSafe) String() string {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.b.String()
}

func (b *bufferThreadSafe) Reset() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.b.Reset()
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
		stdErrBuff:     &bufferThreadSafe{bytes.Buffer{}, &sync.Mutex{}},
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

	// TODO add issue to fix this bug
	// if the user puts in an incorrect repo name
	// it will hang when the command runs
	cmdPip := exec.Command(pipPath, pipInstallArgs...)
	var cmdPipOut bytes.Buffer
	cmdPip.Stderr = &cmdPipOut

	if err := cmdPip.Run(); err != nil {
		return fmt.Errorf("error while running pip. stderr: '%s', err: '%s'", cmdPipOut.String(), err)
	} else if cmdPipOut.Len() > 0 {
		p.logger.Warningf("Python deployer pip command had stderr output: %s", cmdPipOut.String())
	}
	return nil
}

func (p *cliWrapper) Deploy(fullModuleName string) (io.WriteCloser, io.ReadCloser, error) {
	pythonModule, err := parseModuleName(fullModuleName)
	if err != nil {
		return nil, nil, err
	}

	modulePath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return nil, nil, err
	}
	venvPython := filepath.Join(*modulePath, "venv/bin/python")
	args := []string{"-m"}
	moduleInvokableName := strings.ReplaceAll(*pythonModule.ModuleName, "-", "_")
	args = append(args, moduleInvokableName, "--atp")

	p.deployCommand = exec.Command(venvPython, args...) //nolint:gosec
	p.deployCommand.Stderr = p.stdErrBuff

	// execute plugin in its own directory in case the plugin needs
	// to write to its current working directory
	p.deployCommand.Dir = p.pluginDir

	stdin, err := p.deployCommand.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := p.deployCommand.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	err = p.deployCommand.Start()
	if p.stdErrBuff.Len() > 0 {
		p.logger.Warningf("python process stderr already has content '%s'", p.stdErrBuff.String())
	}
	if err != nil {
		return nil, nil, fmt.Errorf("error starting python process (%w)", err)
	}
	return stdin, stdout, nil
}

func (p *cliWrapper) KillAndClean() error {
	p.logger.Infof("killing config process with pid %d", p.deployCommand.Process.Pid)

	// even if this error was non-nil, we would not handle it differently
	_ = p.deployCommand.Process.Kill()

	_, err := p.deployCommand.Process.Wait()
	if err != nil {
		return err
	}
	if p.stdErrBuff.Len() > 0 {
		p.logger.Warningf("stderr present after plugin execution: '%s'", p.stdErrBuff.String())
	}
	return nil
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
	if err := cmdCreateVenv.Run(); err != nil {
		return fmt.Errorf("error while creating venv. Stderr: '%s', err: '%s'", cmdCreateOut.String(), err)
	} else if cmdCreateOut.Len() > 0 {
		p.logger.Warningf("Python deployer venv command had stderr output: %s", cmdCreateOut.String())
	}
	return nil
}
