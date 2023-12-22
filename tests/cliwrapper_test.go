package tests

import (
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/pythondeployer/internal/cliwrapper"
	"os"
	"sync"
	"testing"
)

// Test pull module immediately returns an error on attempting
// to find a nonexistent public repo instead of hanging, waiting
// manual authentication.
func Test_PullModule_NonexistentGitLocation(t *testing.T) {

	testModule := TestModule{
		location: "nonexistent-repo@git+https://github.com/arcalot/nonexistent-repo.git",
		stepID:   "wait",
		input: map[string]any{
			"seconds": 0.1,
		},
	}

	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	tempdir := "/tmp/pullmodule"
	assert.NoError(t, os.MkdirAll(tempdir, os.ModePerm))
	pluginDir, err := os.MkdirTemp(tempdir, "")
	assert.NoError(t, err)
	logger := log.NewTestLogger(t)
	wrap := cliwrapper.NewCliWrapper(pythonPath, tempdir, pluginDir, logger)
	assert.NoError(t, wrap.Venv(testModule.location))
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := wrap.PullModule(testModule.location, "Always")
		assert.Error(t, err)
	}()

	wg.Wait()
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tempdir))
	})
}
