package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/marcinwyszynski/backendplugin"
)

const (
	stateSuffix = ".state"
	lockSuffix  = ".lock"
)

type FileBackend struct {
	baseDir string
}

func (f *FileBackend) Configure(_ context.Context, config map[string]string) error {
	baseDir, ok := config["directory"]
	if !ok {
		return errors.New("missing directory configuration")
	}

	f.baseDir = baseDir

	return nil
}

func (f *FileBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	files, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, err
	}

	var workspaces []string

	for _, file := range files {
		if strings.HasSuffix(file.Name(), stateSuffix) {
			workspaces = append(workspaces, strings.TrimSuffix(file.Name(), stateSuffix))
		}
	}

	return workspaces, nil
}

func (f *FileBackend) DeleteWorkspace(_ context.Context, workspace string, force bool) error {
	return os.Remove(f.stateFilePath(workspace))
}

func (f *FileBackend) GetStatePayload(_ context.Context, workspace string) (*backendplugin.Payload, error) {
	data, err := os.ReadFile(f.stateFilePath(workspace))
	if err != nil {
		if os.IsNotExist(err) {
			// No file, no problem.
			return nil, nil
		}

		return nil, err
	}

	checksum := md5.Sum(data)

	return &backendplugin.Payload{
		Data: data,
		MD5:  checksum[:],
	}, nil
}

func (f *FileBackend) PutState(_ context.Context, workspace string, data []byte) error {
	return os.WriteFile(f.stateFilePath(workspace), data, 0644)
}

func (f *FileBackend) DeleteState(_ context.Context, workspace string) error {
	return os.Remove(f.stateFilePath(workspace))
}

func (f *FileBackend) LockState(_ context.Context, workspace string, info *backendplugin.LockInfo) (string, error) {
	_, err := os.Stat(f.lockFilePath(workspace))
	if err == nil {
		return "", errors.New("lock already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}

	content, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(f.lockFilePath(workspace), content, 0644); err != nil {
		return "", err
	}

	return info.ID, nil
}

func (f *FileBackend) UnlockState(_ context.Context, workspace, id string) error {
	data, err := os.ReadFile(f.lockFilePath(workspace))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("lock does not exist")
		}
	}

	var info backendplugin.LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}

	if info.ID != id {
		return errors.New("lock ID does not match")
	}

	return os.Remove(f.lockFilePath(workspace))
}

func (f *FileBackend) lockFilePath(workspace string) string {
	return f.baseDir + "/" + workspace + lockSuffix
}

func (f *FileBackend) stateFilePath(workspace string) string {
	return f.baseDir + "/" + workspace + stateSuffix
}
