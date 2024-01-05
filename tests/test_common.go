package tests

import (
	"fmt"
	"os/exec"
)

type TestModule struct {
	Location string
	StepID   string
	Input    map[string]any
}

func GetPythonPath() (string, error) {
	python3Path, errPython3 := exec.LookPath("python3")
	if errPython3 != nil {
		pythonPath, errPython := exec.LookPath("python")
		if errPython != nil {
			return "", fmt.Errorf("error getting Python3 (%s) and python (%s)", errPython3, errPython)
		}
		return pythonPath, nil
	}
	return python3Path, nil
}
