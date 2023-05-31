package tests

import (
	"context"
	"fmt"
	"go.arcalot.io/assert"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/atp"
	"os"
	"testing"
)

const templatePluginInput string = "Tester McTesty"

var inOutConfigGitPullAlwaysNoOverride = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"Always"
}
`
var inOutConfigGitOverrideCheks = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent",
	"overrideModuleCompatibility":"true"
}
`

var inOutConfigGitPullAlways = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"Always",
	"overrideModuleCompatibility":"true"
}
`

var inOutConfigGitPullIfNotPresent = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent"
}
`

var inOutConfigGitOverrideCheks = `
{
	"pythonPath":"/usr/bin/python3.9",
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent",
	"overrideModuleCompatibility":"true"
}
`

func TestRunIncompatiblePlugin(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git"
	connector, _ := getConnector(t, inOutConfigGitPullAlwaysNoOverride, nil)
	_, _, err := RunStep(t, connector, moduleName)
	assert.Error(t, err)
	assert.Equals(t, err.Error(), "impossible to run module arcaflow_plugin_template_python, marked as incompatible in package metadata")
}

func TestRunIncompatiblePluginOverride(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git"
	connector, _ := getConnector(t, inOutConfigGitOverrideCheks, nil)
	outputID, outputData, err := RunStep(t, connector, moduleName)
	assert.Equals(t, *outputID, "success")
	assert.NoError(t, err)
	assert.Equals(t,
		outputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", templatePluginInput)})
}

func TestRunStepGit(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/tsebastiani/arcaflow-plugin-template-python.git@6145c2cd0760495ea6dc5b7399b6d7692e81d368"
	connector, _ := getConnector(t, inOutConfigGitPullAlways, nil)
	outputID, outputData, err := RunStep(t, connector, moduleName)
	assert.Equals(t, *outputID, "success")
	assert.NoError(t, err)
	assert.Equals(t,
		outputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", templatePluginInput)})
}

func TestPullPolicies(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git"
	// this test must be run in the same workdir so it's created upfront
	// and passed to the getConnector func
	workdir := createWorkdir(t)
	connectorAlways, _ := getConnector(t, inOutConfigGitPullAlways, &workdir)
	connectorIfNotPresent, _ := getConnector(t, inOutConfigGitPullIfNotPresent, &workdir)
	// pull mode Always, venv will be removed if present and pulled again
	outputID, outputData, err := RunStep(t, connectorAlways, moduleName)
	assert.NoError(t, err)
	assert.NotNil(t, outputData)
	assert.NotNil(t, outputID)
	assert.Equals(t, *outputID, "success")

	assert.Equals(t,
		outputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", templatePluginInput)})
	// pull mode IfNotPresent, venv will be kept
	outputID, outputData, err = nil, nil, nil
	outputID, outputData, err = RunStep(t, connectorIfNotPresent, moduleName)
	assert.Equals(t, *outputID, "success")
	assert.NoError(t, err)
	assert.Equals(t,
		outputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", templatePluginInput)})
	wrapper := getCliWrapper(t, false, workdir)
	path, err := wrapper.GetModulePath(moduleName)
	assert.NoError(t, err)
	file, err := os.Stat(*path)
	assert.NoError(t, err)
	// venv path modification time is checked
	startTime := file.ModTime()
	// pull mode Always, venv will be removed if present and pulled again
	outputID, outputData, err = nil, nil, nil
	outputID, outputData, err = RunStep(t, connectorAlways, moduleName)
	assert.Equals(t, *outputID, "success")
	assert.NoError(t, err)
	assert.Equals(t,
		outputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", templatePluginInput)})
	file, err = os.Stat(*path)
	assert.NoError(t, err)
	// venv path modification time is checked
	newTime := file.ModTime()
	// new time check must be greater than the first one checked
	assert.Equals(t, newTime.After(startTime), true)
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string) (*string, any, error) {
	stepID := "hello-world"
	input := map[string]any{"name": templatePluginInput}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	plugin, err := connector.Deploy(context.Background(), moduleName)

	if err != nil {
		return nil, nil, err
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
	outputID, outputData, err := atpClient.Execute(ctx, stepID, input)
	return &outputID, outputData, err

}
