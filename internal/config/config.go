package config

type Config struct {
	PythonPath string `json:"pythonPath"`
	WorkDir    string `json:"workdir"`
	//RootDir          string           `json:"rootdir"`
	PythonSemVer     string           `json:"semver"`
	ModulePullPolicy ModulePullPolicy `json:"modulePullPolicy"`
}

type ModulePullPolicy string

const (
	// ModulePullPolicyAlways means that the module will be pulled for every workflow run.
	ModulePullPolicyAlways ModulePullPolicy = "Always"
	// ModulePullPolicyIfNotPresent means the image will be pulled if the module is not present locally
	ModulePullPolicyIfNotPresent ModulePullPolicy = "IfNotPresent"
)
