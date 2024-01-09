package connector_test

import (
	"context"
	"encoding/json"
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/atp"
	"go.flow.arcalot.io/pluginsdk/schema"
	pythondeployer "go.flow.arcalot.io/pythondeployer"
	"go.flow.arcalot.io/pythondeployer/internal/config"
	"go.flow.arcalot.io/pythondeployer/internal/connector"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"testing"
	//"go.arcalot.io/exex"
)

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

const examplePluginNickname string = "pythonuser"

var inOutConfigGitPullIfNotPresent = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent",
    "pythonSemver": "3.0.0"
}
`

func TestRunStepGit(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@52d1a9559c60a615dbd97114572f16d70fa30b1b"
	stepID := "hello-world"
	input := map[string]any{
		"name": examplePluginNickname,
	}

	connector_, _ := GetConnector(t, inOutConfigGitPullIfNotPresent, nil)
	OutputID, OutputData, Error := RunStep(t, connector_, moduleName, stepID, input)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginNickname)})
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string, stepID string, input map[string]any) (string, any, error) {
	plugin, err := connector.Deploy(context.Background(), moduleName)

	if err != nil {
		return "", nil, err
	}
	t.Cleanup(func() {
		assert.NoError(t, plugin.Close())
	})

	atpClient := atp.NewClient(plugin)
	pluginSchema, err := atpClient.ReadSchema()
	assert.NoError(t, err)
	assert.NotNil(t, pluginSchema)
	steps := pluginSchema.Steps()
	step, ok := steps[stepID]
	if !ok {
		t.Fatalf("no such step: %s", stepID)
	}

	_, err = step.Input().Unserialize(input)
	assert.NoError(t, err)

	// Executes the step and validates that the output is correct.
	receivedSignalsChan := make(chan schema.Input)
	emittedSignalsChan := make(chan schema.Input)
	executionResult := atpClient.Execute(
		schema.Input{RunID: t.Name(), ID: stepID, InputData: input},
		receivedSignalsChan,
		emittedSignalsChan,
	)
	assert.NoError(t, atpClient.Close())

	return executionResult.OutputID, executionResult.OutputData, executionResult.Error
}

// This test ensures that this deployer can create and execute
// connectors concurrently, and that those connectors can deploy
// plugins concurrently that create side-effects local to their
// filesystem, and one connector can pull multiple python modules
func TestDeployConcurrent_ConnectorsAndPluginsWithDifferentModules(t *testing.T) {
	type TestModule struct {
		location string
		stepID   string
		input    map[string]any
	}
	testModules := map[string]TestModule{
		"fio": {
			stepID:   "workload",
			location: "arcaflow-plugin-fio@git+https://github.com/arcalot/arcaflow-plugin-fio.git@de07b3e48cefdaa084eb0445616abc2d13670191",
			input: map[string]any{
				"name":    "poisson-rate-submit",
				"cleanup": "true",
				"params": map[string]any{
					"size":           "500KiB",
					"readwrite":      "randrw",
					"ioengine":       "sync",
					"iodepth":        32,
					"io_submit_mode": "inline",
					"rate_iops":      50,
					"rate_process":   "poisson",
					"buffered":       0,
				},
			},
		},
		"template": {
			stepID:   "hello-world",
			location: "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@52d1a9559c60a615dbd97114572f16d70fa30b1b",
			input: map[string]any{
				"name": "arca lot",
			},
		},
		// no git commit hash is used here, so that the code executes
		// the 'latest' branch of cliwrapper.GetModulePath()
		"wait": {
			stepID:   "wait",
			location: "arcaflow-plugin-wait@git+https://github.com/arcalot/arcaflow-plugin-wait.git",
			input: map[string]any{
				"seconds": "0.5",
			},
		},
	}

	rootDir := "/tmp/multi-module"
	serializedConfig := map[string]any{
		"workdir":          rootDir,
		"modulePullPolicy": "IfNotPresent",
		"pythonSemver":     "3.0.0",
	}

	// idempotent test directory creation
	_ = os.RemoveAll(rootDir)
	assert.NoError(t, os.MkdirAll(rootDir, os.ModePerm))
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(rootDir))
	})

	factory := pythondeployer.NewFactory()
	deployerSchema := factory.ConfigurationSchema()
	unserializedConfig, err := deployerSchema.UnserializeType(serializedConfig)
	assert.NoError(t, err)

	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath

	// Choose how many connectors and plugins to make
	const n_connectors = 4
	const n_plugin_copies = 10
	wg := sync.WaitGroup{}
	wg.Add(n_connectors * len(testModules) * n_plugin_copies)

	logger := log.NewTestLogger(t)

	// Test for issues that might occur during concurrent creation of connectors
	// and deployment of plugins
	// Make a goroutine for each connector
	for j := 0; j < n_connectors; j++ {
		connector_, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
		assert.NoError(t, err)

		go func(connector deployer.Connector) {
			for k := 0; k < n_plugin_copies; k++ {
				for _, testModule_ := range testModules {
					go func(testModule TestModule) {
						defer wg.Done()

						output_id, output_data, err := RunStep(
							t, connector, testModule.location, testModule.stepID, testModule.input)
						assert.NoError(t, err)
						assert.Equals(t, output_id, "success")
						assert.MapNotContainsKeyAny(t, "error", output_data.(map[any]any))
						if output_id == "error" {
							errorMsg, ok := output_data.(map[any]any)["error"]
							if ok {
								logger.Debugf("plugin error '%s'", errorMsg.(string))
							}
						}
					}(testModule_)
				}
			}
		}(connector_)
	}
	// Wait for all the plugins to be done
	wg.Wait()
}

func CreateWorkdir(t *testing.T) string {
	workdir := fmt.Sprintf("/tmp/%s", RandString(10))
	if _, err := os.Stat(workdir); !os.IsNotExist(err) {
		err := os.RemoveAll(workdir)
		assert.NoError(t, err)
	}
	err := os.Mkdir(workdir, os.ModePerm)
	assert.NoError(t, err)
	return workdir
}

func RandString(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func GetConnector(t *testing.T, configJSON string, workdir *string) (deployer.Connector, *config.Config) {
	var serializedConfig any
	if err := json.Unmarshal([]byte(configJSON), &serializedConfig); err != nil {
		t.Fatal(err)
	}
	f := pythondeployer.NewFactory()
	connectorSchema := f.ConfigurationSchema()
	unserializedConfig, err := connectorSchema.UnserializeType(serializedConfig)
	assert.NoError(t, err)
	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath
	// NOTE: randomizing Workdir to avoid parallel tests to
	// remove python folders while other tests are running
	// causing the test to fail
	if workdir == nil {
		unserializedConfig.WorkDir = CreateWorkdir(t)
	} else {
		unserializedConfig.WorkDir = *workdir
	}

	connector_, err := f.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	return connector_, unserializedConfig
}

func TestConnector_PullMod(t *testing.T) {
	logger := log.NewTestLogger(t)

	testCases := map[string]struct {
		pullpolicy      config.ModulePullPolicy
		module_exists   bool
		expected_result bool
	}{
		"ifnotpresent_exists": {
			config.ModulePullPolicyIfNotPresent,
			true,
			false,
		},
		"ifnotpresent_not_exist": {
			config.ModulePullPolicyIfNotPresent,
			false,
			true,
		},
		"always_exists": {
			config.ModulePullPolicyAlways,
			true,
			true,
		},
		"always_not_exists": {
			config.ModulePullPolicyAlways,
			false,
			true,
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Config{
				WorkDir:          "",
				PythonSemVer:     "",
				PythonPath:       "",
				ModulePullPolicy: tc.pullpolicy,
			}
			testPythonCli := &pythonCliStub{
				PullPolicy:  tc.pullpolicy,
				PyModExists: tc.module_exists,
			}
			connector_ := connector.NewConnector(
				&cfg,
				logger,
				"",
				testPythonCli)
			err := connector_.PullMod(
				context.Background(), "", testPythonCli)
			assert.NoError(t, err)
			assert.Equals(t, testPythonCli.PyModPulled, tc.expected_result)
		})
	}
}

type pythonCliStub struct {
	PyModExists bool
	PyModPulled bool
	PullPolicy  config.ModulePullPolicy
}

func (p *pythonCliStub) PullModule(fullModuleName string) error {
	moduleExists, _ := p.ModuleExists("")
	if !*moduleExists || p.PullPolicy == config.ModulePullPolicyAlways {
		p.PyModPulled = true
	}
	return nil
}

func (p *pythonCliStub) Deploy(fullModuleName string, pluginDirAbsPath string) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exec.Cmd, error) {
	return nil, nil, nil, nil, nil
}

func (p *pythonCliStub) GetModulePath(_ string) (*string, error) {
	return nil, nil
}

func (p *pythonCliStub) ModuleExists(_ string) (*bool, error) {
	exists := &p.PyModExists
	return exists, nil
}

func (p *pythonCliStub) Venv(_ string) error {
	return nil
}
