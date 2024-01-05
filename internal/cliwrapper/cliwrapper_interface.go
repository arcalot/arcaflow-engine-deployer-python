package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	PullModule(fullModuleName string, pullPolicy string) error
	Deploy(fullModuleName string, pluginDirAbsPath string) (io.WriteCloser, io.ReadCloser, error)
	KillAndClean() error
	GetModulePath(fullModuleName string) (*string, error)
	ModuleExists(fullModuleName string) (*bool, error)
	Venv(fullModuleName string) error
}
