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
	python3Path, errPython3 := exec.LookPath("python3")
	if errPython3 != nil {
		pythonPath, errPython := exec.LookPath("python")
		if errPython != nil {
			return "", fmt.Errorf("error getting Python3 (%s) and python (%s)", errPython3, errPython)
		}
		return pythonPath, nil
	}
	return python3Path, nil
}

// Test pull module immediately returns an error on attempting
// to find a nonexistent public repo instead of hanging, waiting
// manual authentication.
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

	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)

	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, logger)
	assert.NoError(t, wrap.Venv(testModule.Location))

	err = wrap.PullModule(testModule.Location, "Always")
	assert.Error(t, err)

	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})
}
