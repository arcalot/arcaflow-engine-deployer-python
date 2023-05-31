package tests

import (
	"fmt"
	"github.com/tsebastiani/arcaflow-engine-deployer-python/internal/cliwrapper"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"os/exec"
	"testing"
)

func TestPullImage(t *testing.T) {
	module := "arcaflow-plugin-template-python@git+https://github.com/tsebastiani/arcaflow-plugin-template-python.git"
	workDir := createWorkdir(t)
	pythonPath, err := exec.LookPath("python")
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	python := cliwrapper.NewCliWrapper(pythonPath, workDir, logger, false)
	err = pullModule(python, module, workDir, t)
	if err != nil {
		logger.Errorf(err.Error())
	}
	assert.NoError(t, err)
}

func TestImageExists(t *testing.T) {
	module := "arcaflow-plugin-template-python@git+https://github.com/tsebastiani/arcaflow-plugin-template-python.git"
	workDir := createWorkdir(t)
	pythonPath, err := exec.LookPath("python")
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	python := cliwrapper.NewCliWrapper(pythonPath, workDir, logger, false)
	removeModuleIfExists(module, python, t)
	exists, err := python.ModuleExists(module)
	assert.Nil(t, err)
	assert.Equals(t, *exists, false)
	err = pullModule(python, module, workDir, t)
	if err != nil {
		logger.Errorf(err.Error())
	}
	assert.NoError(t, err)
	exists, err = python.ModuleExists(module)
	assert.NoError(t, err)
	assert.Equals(t, *exists, true)
}

func TestImageFormatValidation(t *testing.T) {
	moduleGitNoCommit := "arcaflow-plugin-template-python@git+https://github.com/tsebastiani/arcaflow-plugin-template-python.git"
	moduleGitCommit := "arcaflow-plugin-template-python@git+https://github.com/tsebastiani/arcaflow-plugin-template-python.git@8e43b657db73929d6f8ccb893f059bb67658523f"
	moduleWrongFormat := "https://arcalot.io"
	wrongFormatMessage := "wrong module name format, please use <module-name>@git+<repo_url>[@<commit_sha>]"
	workDir := createWorkdir(t)
	pythonPath, err := exec.LookPath("python")
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	wrapperGit := cliwrapper.NewCliWrapper(pythonPath, workDir, logger, false)

	// happy path
	path, err := wrapperGit.GetModulePath(moduleGitCommit)
	assert.NoError(t, err)
	assert.Equals(
		t,
		*path,
		fmt.Sprintf("%s/arcaflow-plugin-template-python_8e43b657db73929d6f8ccb893f059bb67658523f", workDir),
	)

	path, err = wrapperGit.GetModulePath(moduleGitNoCommit)
	assert.NoError(t, err)
	assert.Equals(t, *path, fmt.Sprintf("%s/arcaflow-plugin-template-python_latest", workDir))

	_, err = wrapperGit.GetModulePath(moduleWrongFormat)
	assert.Error(t, err)
	assert.Equals(t, err.Error(), wrongFormatMessage)

}
