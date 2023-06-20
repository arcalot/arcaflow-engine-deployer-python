package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	ModuleExists(fullModuleName string) (*bool, error)
	PullModule(fullModuleName string) error
	Deploy(fullModuleName string) (io.WriteCloser, io.ReadCloser, error)
	KillAndClean() error
	GetModulePath(fullModuleName string) (*string, error)
}
