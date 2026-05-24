package utils

import (
	"bufio"
	"chat/globals"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxSafeFileNameRunes = 120

func CreateFolder(path string) bool {
	if err := os.MkdirAll(path, os.ModePerm); err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

func Exists(path string) bool {
	err := os.Mkdir(path, os.ModePerm)
	return err != nil && os.IsExist(err)
}

func DirSafe(path string) string {
	CreateFolder(path)
	return path
}

func FileDirSafe(file string) string {
	if strings.LastIndex(file, "/") == -1 {
		return file
	}

	return DirSafe(file[:strings.LastIndex(file, "/")])
}

func FileSafe(file string) string {
	FileDirSafe(file)
	return file
}

func SafeJoin(base string, parts ...string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", errors.New("base path cannot be empty")
	}

	baseClean := filepath.Clean(base)
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ReplaceAll(part, "\\", "/"))
		if part == "" {
			return "", errors.New("path segment cannot be empty")
		}
		if strings.ContainsRune(part, 0) {
			return "", errors.New("path segment cannot contain null bytes")
		}
		if strings.HasPrefix(part, "/") {
			return "", fmt.Errorf("path segment escapes base: %s", part)
		}

		clean := filepath.Clean(filepath.FromSlash(part))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, fmt.Sprintf("..%c", os.PathSeparator)) || filepath.IsAbs(clean) {
			return "", fmt.Errorf("path segment escapes base: %s", part)
		}
		cleanParts = append(cleanParts, clean)
	}

	target := filepath.Join(append([]string{baseClean}, cleanParts...)...)
	baseAbs, err := filepath.Abs(baseClean)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, fmt.Sprintf("..%c", os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base: %s", target)
	}

	return filepath.ToSlash(target), nil
}

func SafeFileName(raw string, fallback string) string {
	name := strings.TrimSpace(raw)
	var builder strings.Builder
	for _, char := range name {
		switch {
		case char == 0:
			continue
		case char < 32 || char == 127:
			builder.WriteRune('-')
		case strings.ContainsRune(`/\<>:"|?*`, char):
			builder.WriteRune('-')
		default:
			builder.WriteRune(char)
		}
	}

	name = strings.Trim(builder.String(), ". ")
	if name == "" {
		name = strings.TrimSpace(fallback)
	}
	if name == "" {
		name = "file"
	}

	runes := []rune(name)
	if len(runes) > maxSafeFileNameRunes {
		name = strings.TrimSpace(string(runes[:maxSafeFileNameRunes]))
	}
	if name == "" {
		return "file"
	}
	return name
}

func WriteFile(path string, data string, folderSafe bool) error {
	if folderSafe {
		FileDirSafe(path)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	if _, err := file.WriteString(data); err != nil {
		globals.Warn(fmt.Sprintf("[utils] write file error: %s (path: %s, bytes len: %d)", err.Error(), path, len(data)))
		return err
	}
	return nil
}

func ReadFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func Walk(path string) []string {
	var files []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, handlePath(path))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return files
}

func GetFileSize(path string) int64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	stat, err := file.Stat()
	if err != nil {
		return 0
	}
	return stat.Size()
}

func GetFileCreated(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	stat, err := file.Stat()
	if err != nil {
		return ""
	}
	return stat.ModTime().String()
}

func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func CopyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		err := in.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), src))
		}
	}(in)

	FileDirSafe(dst)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), dst))
		}
	}(out)

	_, err = io.Copy(out, in)
	return err
}

func DeleteFile(path string) error {
	return os.Remove(path)
}

func ReadFileLatestLines(path string, length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be greater than 0")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) < length {
		length = len(lines)
	}

	return strings.Join(lines[len(lines)-length:], "\n"), nil
}
