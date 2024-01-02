package cliwrapper

type CliWrapper interface {
	PullModule(fullModuleName string, pullPolicy string) error
	Deploy(fullModuleName string) (*CliWrapperPlugin, error)
	GetModulePath(fullModuleName string) (*string, error)
	Venv(fullModuleName string) error
}
