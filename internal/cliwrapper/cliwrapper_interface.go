package cliwrapper

import (
	"io"
)

type CliWrapper interface {
	PullModule(fullModuleName string, pullPolicy string) error
	Deploy(fullModuleName string) (io.WriteCloser, io.ReadCloser, error)
	KillAndClean() error
	GetModulePath(fullModuleName string) (*string, error)
	Venv(fullModuleName string) error
}
