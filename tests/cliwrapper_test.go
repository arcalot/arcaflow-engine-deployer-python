package tests

import (
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"testing"
)

func TestPullImage(t *testing.T) {
	module := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	workDir := createWorkdir(t)

	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	python := cliwrapper.NewCliWrapper(pythonPath, workDir, logger)
	err = pullModule(python, module, workDir, t)
	if err != nil {
		logger.Errorf(err.Error())
	}
	assert.NoError(t, err)
}

func TestImageExists(t *testing.T) {
	module := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	workDir := createWorkdir(t)
	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	python := cliwrapper.NewCliWrapper(pythonPath, workDir, logger)
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
	moduleGitNoCommit := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	moduleGitCommit := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git@50b6787e0a53b94f7cafb9c40262054832b8c989"
	moduleWrongFormat := "https://arcalot.io"
	wrongFormatMessage := "wrong module name format, please use <module-name>@git+<repo_url>[@<commit_sha>]"
	workDir := createWorkdir(t)
	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	wrapperGit := cliwrapper.NewCliWrapper(pythonPath, workDir, logger)

	// happy path
	path, err := wrapperGit.GetModulePath(moduleGitCommit)
	assert.NoError(t, err)
	assert.Equals(
		t,
		*path,
		fmt.Sprintf("%s/arcaflow-plugin-example_50b6787e0a53b94f7cafb9c40262054832b8c989", workDir),
	)

	path, err = wrapperGit.GetModulePath(moduleGitNoCommit)
	assert.NoError(t, err)
	assert.Equals(t, *path, fmt.Sprintf("%s/arcaflow-plugin-example_latest", workDir))

	_, err = wrapperGit.GetModulePath(moduleWrongFormat)
	assert.Error(t, err)
	assert.Equals(t, err.Error(), wrongFormatMessage)

}
