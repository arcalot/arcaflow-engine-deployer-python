package tests

import (
	"context"
	"fmt"
	"go.arcalot.io/assert"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/atp"
	"go.flow.arcalot.io/pluginsdk/schema"
	"os"
	"testing"
)

const examplePluginInput string = "Tester McTesty"

var inOutConfigGitPullAlways = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"Always"
}
`

var inOutConfigGitPullIfNotPresent = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent"
}
`

func TestRunStepGit(t *testing.T) {
	moduleName := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	connector, _ := getConnector(t, inOutConfigGitPullAlways, nil)
	OutputID, OutputData, Error := RunStep(t, connector, moduleName)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginInput)})
}

func TestPullPolicies(t *testing.T) {
	moduleName := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	// this test must be run in the same workdir so it's created upfront
	// and passed to the getConnector func
	workdir := createWorkdir(t)
	connectorAlways, _ := getConnector(t, inOutConfigGitPullAlways, &workdir)
	connectorIfNotPresent, _ := getConnector(t, inOutConfigGitPullIfNotPresent, &workdir)
	// pull mode Always, venv will be removed if present and pulled again
	OutputID, OutputData, Error := RunStep(t, connectorAlways, moduleName)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginInput)})
	// pull mode IfNotPresent, venv will be kept
	OutputID, OutputData, Error = RunStep(t, connectorIfNotPresent, moduleName)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginInput)})
	wrapper := getCliWrapper(t, workdir)
	path, err := wrapper.GetModulePath(moduleName)
	assert.NoError(t, err)
	file, err := os.Stat(*path)
	assert.NoError(t, err)
	// venv path modification time is checked
	startTime := file.ModTime()
	// pull mode Always, venv will be removed if present and pulled again
	OutputID, OutputData, Error = RunStep(t, connectorAlways, moduleName)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginInput)})
	file, err = os.Stat(*path)
	assert.NoError(t, err)
	// venv path modification time is checked
	newTime := file.ModTime()
	// new time check must be greater than the first one checked
	assert.Equals(t, newTime.After(startTime), true)
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string) (string, any, error) {
	stepID := "hello-world"
	input := map[string]any{"name": examplePluginInput}

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
