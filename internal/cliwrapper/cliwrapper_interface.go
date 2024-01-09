package cliwrapper

import (
	"go.arcalot.io/exex"
	"io"
)

type CliWrapper interface {
	PullModule(fullModuleName string) error
	Deploy(fullModuleName string, pluginDirAbsPath string) (io.WriteCloser, io.ReadCloser, io.ReadCloser, *exex.Cmd, error)
	GetModulePath(fullModuleName string) (*string, error)
	ModuleExists(fullModuleName string) (*bool, error)
	Venv(fullModuleName string) error
}
