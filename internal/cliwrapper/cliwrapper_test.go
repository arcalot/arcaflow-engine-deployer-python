package cliwrapper_test

import (
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"os"
	"os/exec"
	"testing"
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
		errP3.Error(), errP.Error())
}

// Test the function PullModule immediately returns an error on
// attempting to find a nonexistent public repo instead of hanging,
// awaiting manual authentication.
func Test_PullModule_NonexistentGitLocation(t *testing.T) {
	testModule := TestModule{
		Location: "nonexistent-repo@git+https://github.com/arcalot/nonexistent-repo.git",
		StepID:   "wait",
		Input: map[string]any{
			"seconds": 0.1,
		},
	}

	tempdir := "/tmp/pullmodule"
	_ = os.RemoveAll(tempdir)
	assert.NoError(t, os.MkdirAll(tempdir, os.ModePerm))
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})

	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)

	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, logger)
	assert.NoError(t, wrap.Venv(testModule.Location))

	err = wrap.PullModule(testModule.Location)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error pip installing")
}
