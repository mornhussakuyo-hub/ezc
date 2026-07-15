package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Key int

var inputReader = bufio.NewReader(os.Stdin)

const (
	KeyCancel Key = iota
	KeyEnter
	KeyRight
)

func Select(title string, items []string, allowRight bool) (int, Key, error) {
	if len(items) == 0 {
		return -1, KeyCancel, errors.New("没有可选择的项目")
	}
	state, err := enterRaw()
	if err != nil {
		return selectByNumber(title, items)
	}
	columns, rows := terminalDimensions()
	renderedLines := 0
	fmt.Print("\x1b[?25l")
	defer func() {
		clearInline(renderedLines)
		leaveRaw(state)
	}()

	selected := 0
	for {
		renderedLines = renderInline(title, items, selected, allowRight, columns, rows, renderedLines)
		key, err := readKey(inputReader)
		if err != nil {
			return -1, KeyCancel, err
		}
		switch key {
		case "up":
			selected = (selected - 1 + len(items)) % len(items)
		case "down":
			selected = (selected + 1) % len(items)
		case "right":
			if allowRight {
				return selected, KeyRight, nil
			}
		case "enter":
			return selected, KeyEnter, nil
		case "cancel":
			return -1, KeyCancel, nil
		}
	}
}

func Confirm(question string) (bool, error) {
	fmt.Printf("%s [y/N]: ", question)
	line, err := inputReader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func Conflict(destination string) (int, error) {
	fmt.Printf("目标已存在：%s\n", destination)
	fmt.Print("请选择 [o] 覆盖 / [r] 自动重命名 / [c] 取消: ")
	line, err := inputReader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "o", "overwrite":
		return 1, nil
	case "r", "rename":
		return 2, nil
	default:
		return 0, nil
	}
}

func renderInline(title string, items []string, selected int, allowRight bool, columns, rows, previousLines int) int {
	boxWidth := min(max(columns-2, 20), 100)
	contentWidth := boxWidth - 4
	visibleCount := min(len(items), min(10, max(3, rows/2-4)))
	start := selected - visibleCount/2
	start = max(0, min(start, len(items)-visibleCount))
	end := start + visibleCount

	instructions := "↑↓ 移动 · Enter 确定 · q 退出"
	if allowRight {
		instructions = "↑↓ 移动 · →/Enter 操作 · q 退出"
	}
	lines := []string{
		"┌" + strings.Repeat("─", boxWidth-2) + "┐",
		"│ " + fitAndPad(title, contentWidth) + " │",
		"│ " + fitAndPad(fmt.Sprintf("%d/%d · %s", selected+1, len(items), instructions), contentWidth) + " │",
	}
	for index := start; index < end; index++ {
		item := fitAndPad("  "+items[index], contentWidth)
		if index == selected {
			item = fitAndPad("› "+items[index], contentWidth)
			lines = append(lines, "│ \x1b[7m"+item+"\x1b[0m │")
		} else {
			lines = append(lines, "│ "+item+" │")
		}
	}
	lines = append(lines, "└"+strings.Repeat("─", boxWidth-2)+"┘")

	if previousLines > 0 {
		fmt.Printf("\x1b[%dA", previousLines)
	}
	for _, line := range lines {
		fmt.Printf("\r\x1b[2K%s\r\n", line)
	}
	return len(lines)
}

func clearInline(lines int) {
	if lines == 0 {
		return
	}
	fmt.Printf("\x1b[%dA", lines)
	for index := 0; index < lines; index++ {
		fmt.Print("\r\x1b[2K")
		if index < lines-1 {
			fmt.Print("\x1b[1B")
		}
	}
	if lines > 1 {
		fmt.Printf("\x1b[%dA", lines-1)
	}
	fmt.Print("\r")
}

func fitAndPad(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if displayWidth(text) <= width {
		return text + strings.Repeat(" ", width-displayWidth(text))
	}
	if width == 1 {
		return "…"
	}
	var builder strings.Builder
	used := 0
	for _, value := range text {
		valueWidth := runeWidth(value)
		if used+valueWidth > width-1 {
			break
		}
		builder.WriteRune(value)
		used += valueWidth
	}
	builder.WriteRune('…')
	used++
	return builder.String() + strings.Repeat(" ", width-used)
}

func displayWidth(text string) int {
	width := 0
	for _, value := range text {
		width += runeWidth(value)
	}
	return width
}

func runeWidth(value rune) int {
	if unicode.IsControl(value) || unicode.Is(unicode.Mn, value) {
		return 0
	}
	if value >= 0x1100 && (value <= 0x115f || value == 0x2329 || value == 0x232a ||
		(value >= 0x2e80 && value <= 0xa4cf) || (value >= 0xac00 && value <= 0xd7a3) ||
		(value >= 0xf900 && value <= 0xfaff) || (value >= 0xfe10 && value <= 0xfe19) ||
		(value >= 0xfe30 && value <= 0xfe6f) || (value >= 0xff00 && value <= 0xff60) ||
		(value >= 0xffe0 && value <= 0xffe6) || (value >= 0x1f300 && value <= 0x1faff)) {
		return 2
	}
	return 1
}

func readKey(reader *bufio.Reader) (string, error) {
	value, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	switch value {
	case '\r', '\n':
		return "enter", nil
	case 'q', 'Q', 3:
		return "cancel", nil
	case 'k', 'K':
		return "up", nil
	case 'j', 'J':
		return "down", nil
	case 'l', 'L':
		return "right", nil
	case 0x1b:
		second, err := reader.ReadByte()
		if err != nil {
			return "cancel", nil
		}
		if second != '[' {
			return "cancel", nil
		}
		third, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		switch third {
		case 'A':
			return "up", nil
		case 'B':
			return "down", nil
		case 'C':
			return "right", nil
		}
	}
	return "", nil
}

func selectByNumber(title string, items []string) (int, Key, error) {
	fmt.Println(title)
	for index, item := range items {
		fmt.Printf("%d) %s\n", index+1, item)
	}
	fmt.Print("输入序号（留空取消）: ")
	line, err := inputReader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return -1, KeyCancel, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return -1, KeyCancel, nil
	}
	index, err := strconv.Atoi(line)
	if err != nil || index < 1 || index > len(items) {
		return -1, KeyCancel, errors.New("无效的选择")
	}
	return index - 1, KeyEnter, nil
}
