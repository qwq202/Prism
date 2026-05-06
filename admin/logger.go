package admin

import (
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"strings"
)

const logDirectory = "logs"

type LogFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func ListLogs() []LogFile {
	return utils.Each(utils.Walk(logDirectory), func(path string) LogFile {
		return LogFile{
			Path: logRelativePath(path),
			Size: utils.GetFileSize(path),
		}
	})
}

func logRelativePath(path string) string {
	rel, err := filepath.Rel(logDirectory, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

func resolveLogPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("log path is empty")
	}
	if filepath.IsAbs(path) {
		return "", errors.New("absolute log path is not allowed")
	}

	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, fmt.Sprintf("..%c", os.PathSeparator)) {
		return "", errors.New("log path escapes log directory")
	}

	root, err := filepath.Abs(logDirectory)
	if err != nil {
		return "", err
	}
	target := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, fmt.Sprintf("..%c", os.PathSeparator)) {
		return "", errors.New("log path escapes log directory")
	}

	if evaluated, err := filepath.EvalSymlinks(target); err == nil {
		evaluatedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(evaluatedRoot, evaluated)
		if err != nil {
			return "", err
		}
		if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, fmt.Sprintf("..%c", os.PathSeparator)) {
			return "", errors.New("log path escapes log directory")
		}
	}

	return target, nil
}

func getLogPath(path string) (string, error) {
	return resolveLogPath(path)
}

func getBlobFile(c *gin.Context, path string) error {
	logPath, err := getLogPath(path)
	if err != nil {
		return err
	}

	c.File(logPath)
	return nil
}

func deleteLogFile(path string) error {
	logPath, err := getLogPath(path)
	if err != nil {
		return err
	}

	return utils.DeleteFile(logPath)
}

func getLatestLogs(n int) string {
	if n <= 0 {
		n = 100
	}

	path, err := getLogPath(globals.DefaultLoggerFile)
	if err != nil {
		return fmt.Sprintf("read error: %s", err.Error())
	}

	content, err := utils.ReadFileLatestLines(path, n)

	if err != nil {
		return fmt.Sprintf("read error: %s", err.Error())
	}

	return content
}
