package cliwrapper_test

import (
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"go.flow.arcalot.io/pythondeployer/tests"
	"os"
	"sync"
	"testing"
)

// Test pull module immediately returns an error on attempting
// to find a nonexistent public repo instead of hanging, waiting
// manual authentication.
func Test_PullModule_NonexistentGitLocation(t *testing.T) {

	testModule := tests.TestModule{
		Location: "nonexistent-repo@git+https://github.com/arcalot/nonexistent-repo.git",
		StepID:   "wait",
		Input: map[string]any{
			"seconds": 0.1,
		},
	}

	pythonPath, err := tests.GetPythonPath()
	assert.NoError(t, err)
	tempdir := "/tmp/pullmodule"
	assert.NoError(t, os.MkdirAll(tempdir, os.ModePerm))
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, logger)
	assert.NoError(t, wrap.Venv(testModule.Location))
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := wrap.PullModule(testModule.Location, "Always")
		assert.Error(t, err)
	}()

	wg.Wait()
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})
}
