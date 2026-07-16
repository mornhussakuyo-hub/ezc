package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mornhussakuyo-hub/ezc/internal/clipboard"
	"github.com/mornhussakuyo-hub/ezc/internal/fileops"
	"github.com/mornhussakuyo-hub/ezc/internal/lock"
	"github.com/mornhussakuyo-hub/ezc/internal/search"
	"github.com/mornhussakuyo-hub/ezc/internal/ui"
)

var Version = "dev"

type application struct {
	store *clipboard.Store
}

func Run(arguments []string) int {
	if len(arguments) > 0 && arguments[0] == "__lock-worker" {
		if len(arguments) != 5 {
			return 2
		}
		if err := lock.RunWorker(arguments[1], arguments[2], arguments[3], arguments[4]); err != nil {
			return 1
		}
		return 0
	}

	if len(arguments) == 0 {
		printUsage()
		return 0
	}
	if arguments[0] == "help" || arguments[0] == "--help" {
		printUsage()
		return 0
	}
	if arguments[0] == "version" || arguments[0] == "--version" {
		fmt.Printf("ezc %s\n", Version)
		return 0
	}

	store, err := clipboard.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误：%v\n", err)
		return 1
	}
	current := &application{store: store}
	if err := current.dispatch(arguments[0], arguments[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "错误：%v\n", err)
		return 1
	}
	return 0
}

func (current *application) dispatch(command string, arguments []string) error {
	switch command {
	case "cp":
		return current.copyOrCut(arguments, clipboard.Copy)
	case "ct":
		return current.copyOrCut(arguments, clipboard.Cut)
	case "pst":
		return current.paste(arguments)
	case "pad":
		if len(arguments) != 0 {
			return errors.New("pad 不接受参数")
		}
		return current.pad()
	case "rm":
		return current.remove(arguments)
	default:
		return fmt.Errorf("未知命令 %q；运行 ezc help 查看帮助", command)
	}
}

func (current *application) copyOrCut(arguments []string, operation clipboard.Operation) error {
	path, showHidden, err := parseSourceArguments(arguments)
	if err != nil {
		return err
	}
	if path == "" {
		path, err = selectCurrentDirectory(showHidden)
		if err != nil {
			return err
		}
		if path == "" {
			return nil
		}
	} else {
		path, err = resolveSourceArgument(path, showHidden)
		if err != nil {
			return err
		}
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("解析路径: %w", err)
	}
	if _, err := os.Lstat(absolutePath); err != nil {
		return fmt.Errorf("访问 %q: %w", absolutePath, err)
	}

	previous, err := current.store.Find(absolutePath)
	if err != nil {
		return err
	}
	if previous != nil && previous.Operation == operation {
		previous.AddedAt = time.Now()
		if _, err := current.store.Upsert(*previous); err != nil {
			return err
		}
		fmt.Printf("已置顶：%s\n", absolutePath)
		return nil
	}

	entry := clipboard.Entry{Path: absolutePath, Operation: operation, AddedAt: time.Now()}
	if operation == clipboard.Cut {
		entry.Lock, err = lock.Acquire(absolutePath)
		if err != nil {
			return err
		}
	} else if previous != nil && previous.Operation == clipboard.Cut {
		if err := lock.Release(previous.Lock); err != nil {
			return fmt.Errorf("解除原剪切锁: %w", err)
		}
	}

	if _, err := current.store.Upsert(entry); err != nil {
		if operation == clipboard.Cut {
			_ = lock.Release(entry.Lock)
		}
		return err
	}
	if operation == clipboard.Copy {
		fmt.Printf("已复制到剪切板：%s\n", absolutePath)
	} else {
		fmt.Printf("已剪切到剪切板：%s\n", absolutePath)
	}
	return nil
}

func (current *application) paste(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("用法：ezc pst [目标目录]")
	}
	entries, err := current.store.List()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("剪切板为空")
	}
	target := "."
	if len(arguments) == 1 {
		target = arguments[0]
	}
	return current.pasteEntry(entries[0], target)
}

func (current *application) pasteEntry(entry clipboard.Entry, target string) error {
	if _, err := os.Lstat(entry.Path); os.IsNotExist(err) {
		if entry.Operation == clipboard.Cut {
			_ = lock.Release(entry.Lock)
		}
		_, _ = current.store.Remove(entry.Path)
		return fmt.Errorf("源路径已经不存在，已自动清理索引：%s", entry.Path)
	} else if err != nil {
		return fmt.Errorf("检查源路径: %w", err)
	}

	move := entry.Operation == clipboard.Cut
	if move {
		if err := lock.Release(entry.Lock); err != nil {
			return fmt.Errorf("解除剪切锁: %w", err)
		}
	}
	destination, err := fileops.Paste(entry.Path, target, move, resolveConflict)
	if err != nil {
		if move {
			newLock, lockErr := lock.Acquire(entry.Path)
			if lockErr == nil {
				entry.Lock = newLock
				_ = current.store.Replace(entry)
			} else if !os.IsNotExist(lockErr) {
				return fmt.Errorf("%v；同时未能恢复剪切锁: %v", err, lockErr)
			}
		}
		return err
	}
	if move {
		if _, err := current.store.Remove(entry.Path); err != nil {
			return fmt.Errorf("文件已移动，但删除剪切板索引失败: %w", err)
		}
	}
	fmt.Printf("已粘贴到：%s\n", destination)
	return nil
}

func (current *application) pad() error {
	for {
		if err := current.cleanMissing(); err != nil {
			return err
		}
		entries, err := current.store.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("剪切板为空")
			return nil
		}
		labels := make([]string, len(entries))
		for index, entry := range entries {
			operation := "复制"
			if entry.Operation == clipboard.Cut {
				operation = "剪切"
			}
			labels[index] = fmt.Sprintf("[%s] %s", operation, entry.Path)
		}
		selected, key, err := ui.Select("EZC 剪切板", labels, true)
		if err != nil {
			return err
		}
		if key == ui.KeyCancel {
			return nil
		}
		entry := entries[selected]
		action, actionKey, err := ui.Select("请选择操作", []string{"粘贴到当前目录", "从剪切板移除", "返回"}, false)
		if err != nil {
			return err
		}
		if actionKey == ui.KeyCancel || action == 2 {
			continue
		}
		if action == 0 {
			if err := current.pasteEntry(entry, "."); err != nil {
				fmt.Fprintf(os.Stderr, "错误：%v\n", err)
			}
			continue
		}
		confirmed, err := ui.Confirm(fmt.Sprintf("确定移除 %q 吗？", entry.Path))
		if err != nil {
			return err
		}
		if confirmed {
			if err := current.removeEntry(entry); err != nil {
				return err
			}
			fmt.Printf("已移除：%s\n", entry.Path)
		}
	}
}

func (current *application) remove(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("用法：ezc rm [文件名/目录名]")
	}
	var entry clipboard.Entry
	if len(arguments) == 1 {
		absolutePath, err := filepath.Abs(arguments[0])
		if err != nil {
			return fmt.Errorf("解析路径: %w", err)
		}
		found, err := current.store.Find(absolutePath)
		if err != nil {
			return err
		}
		if found == nil {
			return fmt.Errorf("剪切板中不存在：%s", absolutePath)
		}
		entry = *found
	} else {
		entries, err := current.store.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return errors.New("剪切板为空")
		}
		labels := make([]string, len(entries))
		for index, item := range entries {
			labels[index] = item.Path
		}
		selected, key, err := ui.Select("选择要移除的剪切板项目", labels, false)
		if err != nil {
			return err
		}
		if key == ui.KeyCancel {
			return nil
		}
		entry = entries[selected]
	}
	confirmed, err := ui.Confirm(fmt.Sprintf("确定移除 %q 吗？", entry.Path))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("已取消")
		return nil
	}
	if err := current.removeEntry(entry); err != nil {
		return err
	}
	fmt.Printf("已移除：%s\n", entry.Path)
	return nil
}

func (current *application) removeEntry(entry clipboard.Entry) error {
	if entry.Operation == clipboard.Cut {
		if err := lock.Release(entry.Lock); err != nil {
			return fmt.Errorf("解除剪切锁: %w", err)
		}
	}
	_, err := current.store.Remove(entry.Path)
	return err
}

func (current *application) cleanMissing() error {
	removed, err := current.store.CleanMissing()
	if err != nil {
		return err
	}
	for _, entry := range removed {
		if entry.Operation == clipboard.Cut {
			_ = lock.Release(entry.Lock)
		}
		fmt.Printf("源路径已经不存在，已自动清理索引：%s\n", entry.Path)
	}
	return nil
}

func parseSourceArguments(arguments []string) (string, bool, error) {
	var path string
	showHidden := false
	flagsDone := false
	for _, argument := range arguments {
		if !flagsDone && argument == "--" {
			flagsDone = true
			continue
		}
		if !flagsDone && (argument == "-h" || argument == "--hide") {
			showHidden = true
			continue
		}
		if !flagsDone && strings.HasPrefix(argument, "-") {
			return "", false, fmt.Errorf("未知参数 %q", argument)
		}
		if path != "" {
			return "", false, errors.New("每次只能选择一个文件、目录或查询")
		}
		path = argument
	}
	return path, showHidden, nil
}

func selectCurrentDirectory(showHidden bool) (string, error) {
	filtered, err := currentDirectoryEntries(showHidden)
	if err != nil {
		return "", err
	}
	labels := make([]string, len(filtered))
	for index, entry := range filtered {
		labels[index] = entry.Name()
		if entry.IsDir() {
			labels[index] += string(filepath.Separator)
		}
	}
	selected, key, err := ui.Select("选择文件或目录", labels, false)
	if err != nil {
		return "", err
	}
	if key == ui.KeyCancel {
		return "", nil
	}
	return filtered[selected].Name(), nil
}

func resolveSourceArgument(path string, showHidden bool) (string, error) {
	if _, err := os.Lstat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("访问 %q: %w", path, err)
	}

	entries, err := currentDirectoryEntries(showHidden)
	if err != nil {
		return "", err
	}
	matched, ok := bestDirectoryEntry(entries, path)
	if !ok {
		return "", fmt.Errorf("当前目录中没有匹配 %q 的文件或目录", path)
	}
	return matched.Name(), nil
}

func currentDirectoryEntries(showHidden bool) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("读取当前目录: %w", err)
	}
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !showHidden && isHidden(entry.Name()) {
			continue
		}
		filtered = append(filtered, entry)
	}
	sort.SliceStable(filtered, func(left, right int) bool {
		if filtered[left].IsDir() != filtered[right].IsDir() {
			return filtered[left].IsDir()
		}
		return strings.ToLower(filtered[left].Name()) < strings.ToLower(filtered[right].Name())
	})
	return filtered, nil
}

func bestDirectoryEntry(entries []os.DirEntry, query string) (os.DirEntry, bool) {
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	result, ok := search.New(names).Best(query)
	if !ok {
		return nil, false
	}
	return entries[result.Index], true
}

func resolveConflict(destination string) (fileops.ConflictAction, error) {
	action, err := ui.Conflict(destination)
	if err != nil {
		return fileops.ConflictCancel, err
	}
	switch action {
	case 1:
		return fileops.ConflictOverwrite, nil
	case 2:
		return fileops.ConflictRename, nil
	default:
		return fileops.ConflictCancel, nil
	}
}

func printUsage() {
	fmt.Print(`ezc - 终端文件剪切板

用法：
  ezc cp [-h|--hide] [文件、目录或查询]  复制到剪切板
  ezc ct [-h|--hide] [文件、目录或查询]  剪切并锁定
  ezc pst [目标目录]               粘贴剪切板顶部项目
  ezc pad                          浏览剪切板 TUI
  ezc rm [文件或目录]              从剪切板移除
  ezc version                      显示版本

cp/ct 不传路径时会打开当前目录选择器；传入不存在的路径时会将参数作为拼音或模糊查询并自动选择最佳匹配；-h/--hide 显示隐藏项目。
`)
}
