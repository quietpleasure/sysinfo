// Package sysinfo снимает сведения о памяти и диске сервера через
// системные утилиты free и df.
package sysinfo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DiskDevice — раздел, по которому показываем занятость диска.
const DiskDevice = "/dev/vda1"

const cmdTimeout = 3 * time.Second

// Mem — строка Mem из `free -h`.
type Mem struct {
	Total     string
	Used      string
	Free      string
	Shared    string
	BuffCache string
	Available string
}

// Disk — строка раздела из `df -h`.
type Disk struct {
	Filesystem string
	Size       string
	Used       string
	Avail      string
	UsePercent string
	MountedOn  string
}

// Info — сводка о системе. Если команда не отработала, соответствующее
// поле nil, а текст ошибки лежит в MemErr/DiskErr.
type Info struct {
	Mem     *Mem
	MemErr  string
	Disk    *Disk
	DiskErr string
}

// Collect вызывает free и df и собирает результат.
func Collect(ctx context.Context) *Info {
	info := &Info{}

	if mem, err := memory(ctx); err != nil {
		info.MemErr = err.Error()
	} else {
		info.Mem = mem
	}

	if disk, err := disk(ctx, DiskDevice); err != nil {
		info.DiskErr = err.Error()
	} else {
		info.Disk = disk
	}

	return info
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	// Без локализации, иначе вместо "Mem:" будет "Память:" и т.п.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

func memory(ctx context.Context) (*Mem, error) {
	out, err := run(ctx, "free", "-h")
	if err != nil {
		return nil, err
	}
	return parseFree(out)
}

func parseFree(out string) (*Mem, error) {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 7 && f[0] == "Mem:" {
			return &Mem{
				Total:     f[1],
				Used:      f[2],
				Free:      f[3],
				Shared:    f[4],
				BuffCache: f[5],
				Available: f[6],
			}, nil
		}
	}
	return nil, fmt.Errorf("free: строка Mem не найдена")
}

func disk(ctx context.Context, device string) (*Disk, error) {
	// -P — POSIX-формат: одна строка на раздел, без переносов длинных имён.
	out, err := run(ctx, "df", "-hP")
	if err != nil {
		return nil, err
	}
	return parseDf(out, device)
}

// parseDf ищет раздел device, а если его нет (например, локально) —
// раздел, смонтированный в "/".
func parseDf(out, device string) (*Disk, error) {
	var root *Disk
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		d := &Disk{
			Filesystem: f[0],
			Size:       f[1],
			Used:       f[2],
			Avail:      f[3],
			UsePercent: f[4],
			MountedOn:  strings.Join(f[5:], " "),
		}
		if d.Filesystem == device {
			return d, nil
		}
		if d.MountedOn == "/" {
			root = d
		}
	}
	if root != nil {
		return root, nil
	}
	return nil, fmt.Errorf("df: раздел %s не найден", device)
}
