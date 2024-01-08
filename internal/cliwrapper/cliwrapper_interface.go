package cliwrapper

import (
	"io"
	"os/exec"
)

type CliWrapper interface {
	PullModule(fullModuleName string, pullPolicy string) error
	Deploy(fullModuleName string, pluginDirAbsPath string) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exec.Cmd, error)
	GetModulePath(fullModuleName string) (*string, error)
	ModuleExists(fullModuleName string) (*bool, error)
	Venv(fullModuleName string) error
}
