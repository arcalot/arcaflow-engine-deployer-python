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
	stdErrBuff     bufferThreadSafe
}

type bufferThreadSafe struct {
	b    bytes.Buffer
	lock sync.Mutex
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
		// use thread safe buffer for concurrent access to this buffer by
		// the cli wrapper and the cli plugin
		stdErrBuff: bufferThreadSafe{bytes.Buffer{}, sync.Mutex{}},
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
	modulePath := filepath.Join(p.connectorDir, *pythonModule.ModuleName)
	if pythonModule.ModuleVersion != nil {
		modulePath += "_" + *pythonModule.ModuleVersion
	} else {
		modulePath += "_latest"
	}
	return &modulePath, err
}

func (p *cliWrapper) ModuleExists(fullModuleName string) (string, error) {
	moduleAbspath, err := p.GetModulePath(fullModuleName)
	if err != nil {
		return "", fmt.Errorf("error getting python module path while attempting to pull (%w)", err)
	}
	_, fileNotPresent := os.Stat(*moduleAbspath)
	if fileNotPresent != nil {
		// os could not find file
		return "", nil
	}
	// else file found
	return *moduleAbspath, nil
}

func (p *cliWrapper) PullModule(fullModuleName string, pullPolicy string) error {
	pipInstallArgs := []string{"install"}

	// Pip's default behavior is to pull if the module is not present in the
	// environment (i.e. IfNotPresent), so behavior is altered here when the
	// user's pull policy is Always.
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
	p.deployCommand.Stderr = &p.stdErrBuff

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

// Venv creates a Python virtual environment for the given
// Python module in its connector's working directory.
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
