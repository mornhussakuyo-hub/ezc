package fileops

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ConflictAction int

const (
	ConflictCancel ConflictAction = iota
	ConflictOverwrite
	ConflictRename
)

type ConflictResolver func(destination string) (ConflictAction, error)

func Paste(source, targetDirectory string, move bool, resolve ConflictResolver) (string, error) {
	sourceAbsolute, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("解析源路径: %w", err)
	}
	targetAbsolute, err := filepath.Abs(targetDirectory)
	if err != nil {
		return "", fmt.Errorf("解析目标路径: %w", err)
	}
	targetInfo, err := os.Stat(targetAbsolute)
	if err != nil {
		return "", fmt.Errorf("访问目标目录: %w", err)
	}
	if !targetInfo.IsDir() {
		return "", fmt.Errorf("目标不是目录: %s", targetAbsolute)
	}

	sourceInfo, err := os.Lstat(sourceAbsolute)
	if err != nil {
		return "", fmt.Errorf("访问源路径: %w", err)
	}
	if sourceInfo.IsDir() {
		canonicalSource, sourceErr := filepath.EvalSymlinks(sourceAbsolute)
		canonicalTarget, targetErr := filepath.EvalSymlinks(targetAbsolute)
		if sourceErr == nil && targetErr == nil && inside(canonicalSource, canonicalTarget) {
			return "", errors.New("不能将目录粘贴到自身或其子目录中")
		}
	}

	destination := filepath.Join(targetAbsolute, filepath.Base(sourceAbsolute))
	if _, err := os.Lstat(destination); err == nil {
		action, resolveErr := resolve(destination)
		if resolveErr != nil {
			return "", resolveErr
		}
		switch action {
		case ConflictOverwrite:
			if samePath(sourceAbsolute, destination) {
				return "", errors.New("源路径与目标路径相同，不能覆盖自身；请选择自动重命名")
			}
			if err := os.RemoveAll(destination); err != nil {
				return "", fmt.Errorf("移除同名目标: %w", err)
			}
		case ConflictRename:
			destination = availableName(destination)
		default:
			return "", errors.New("已取消粘贴")
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("检查目标路径: %w", err)
	}

	if move {
		if err := os.Rename(sourceAbsolute, destination); err == nil {
			return destination, nil
		}
	}
	if err := copyPath(sourceAbsolute, destination); err != nil {
		_ = os.RemoveAll(destination)
		return "", err
	}
	if move {
		if err := os.RemoveAll(sourceAbsolute); err != nil {
			return "", fmt.Errorf("已复制到目标，但删除源路径失败: %w", err)
		}
	}
	return destination, nil
}

func samePath(left, right string) bool {
	leftInfo, leftStatErr := os.Lstat(left)
	rightInfo, rightStatErr := os.Lstat(right)
	if leftStatErr == nil && rightStatErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true
	}
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		leftAbsolute = filepath.Clean(left)
		rightAbsolute = filepath.Clean(right)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(leftAbsolute), filepath.Clean(rightAbsolute))
	}
	return filepath.Clean(leftAbsolute) == filepath.Clean(rightAbsolute)
}

func inside(sourceDirectory, targetDirectory string) bool {
	relative, err := filepath.Rel(sourceDirectory, targetDirectory)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func availableName(path string) string {
	directory := filepath.Dir(path)
	name := filepath.Base(path)
	extension := filepath.Ext(name)
	stem := strings.TrimSuffix(name, extension)
	for index := 1; ; index++ {
		candidate := filepath.Join(directory, fmt.Sprintf("%s (%d)%s", stem, index, extension))
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func copyPath(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("读取源路径: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return fmt.Errorf("读取符号链接: %w", err)
		}
		if err := os.Symlink(target, destination); err != nil {
			return fmt.Errorf("创建符号链接: %w", err)
		}
		return nil
	}
	if info.IsDir() {
		return copyDirectory(source, destination, info.Mode())
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("暂不支持复制特殊文件: %s", source)
	}
	return copyFile(source, destination, info)
}

func copyDirectory(source, destination string, mode fs.FileMode) error {
	if err := os.Mkdir(destination, mode.Perm()); err != nil {
		return fmt.Errorf("创建目录: %w", err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("读取目录: %w", err)
	}
	for _, entry := range entries {
		if err := copyPath(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string, info fs.FileInfo) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开源文件: %w", err)
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("创建目标文件: %w", err)
	}
	succeeded := false
	defer func() {
		_ = output.Close()
		if !succeeded {
			_ = os.Remove(destination)
		}
	}()

	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("复制文件内容: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("保存目标文件: %w", err)
	}
	succeeded = true
	return nil
}
