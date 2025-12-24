package agentui

import (
	"os"
	"path/filepath"
)

const (
	defaultDirName     = ".zid-agent"
	defaultLogFileName = "logs.txt"
	defaultCfgFileName = "config.json"
)

func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultDirName), nil
}

func DefaultLogPath() (string, error) {
	dir, err := DefaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultLogFileName), nil
}

func DefaultConfigPath() (string, error) {
	dir, err := DefaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultCfgFileName), nil
}
