package cliwrapper_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"go.arcalot.io/assert"
	"go.arcalot.io/exex"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
)

type TestModule struct {
	Location string
	StepID   string
	Input    map[string]any
}

func GetPythonPath() (string, error) {
	var errP3, errP error
	if p, errP3 := exec.LookPath("python3"); errP3 == nil {
		return p, nil
	}
	if p, errP := exec.LookPath("python"); errP == nil {
		return p, nil
	}
	return "", fmt.Errorf("errors getting paths for Python3 (%s) and python (%s)",
		errP3.Error(), errP.Error()) //nolint:govet // errP3 and errP will not be nil if this line is reached
}

// Test the function PullModule immediately returns an error on
// attempting to find a python module from a nonexistent public repo
// instead of hanging, awaiting manual authentication.
func Test_PullModule_NonexistentGitLocation(t *testing.T) {
	testModule := TestModule{
		Location: "nonexistent@git+https://github.com/arcalot/nonexistent.git",
	}

	tempdir := "/tmp/pullmodule1"
	assert.NoError(t, os.MkdirAll(tempdir, 0750))
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})

	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)

	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, logger)

	err = wrap.PullModule(testModule.Location)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pip installing")
	var exErr *exex.ExitError
	assert.Equals(t, errors.As(err, &exErr), true)
	stderrStr := string(exErr.Stderr)
	assert.Contains(t, stderrStr, "git clone")
	assert.Contains(t, stderrStr, "exit code: 128")
}

// Test the function PullModule returns a module name format error
// when the location provided included a typo.
func Test_PullModule_ErrorModuleNameFmt(t *testing.T) {
	testModule := TestModule{
		Location: "git+https://github.com/arcalot/arcaflow-plugin-utilities.git@0.6.1",
	}

	tempdir := "/tmp/pullmodule2"
	assert.NoError(t, os.MkdirAll(tempdir, 0750))
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})

	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)

	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, logger)

	err = wrap.PullModule(testModule.Location)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong module name format")
}
