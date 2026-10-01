package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// MemoryTracker continuously polls the process RSS while it executes
// and falls back to OS rusage when completed.
type MemoryTracker struct {
	cmd            *exec.Cmd
	stopCh         chan struct{}
	maxPolledBytes atomic.Uint64
}

// StartMemoryTracker creates and launches live memory tracking for cmd
func StartMemoryTracker(cmd *exec.Cmd) *MemoryTracker {
	mt := &MemoryTracker{
		cmd:    cmd,
		stopCh: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-mt.stopCh:
				return
			case <-ticker.C:
				if cmd.Process == nil {
					continue
				}
				rss, err := readProcessRSSBytes(cmd.Process.Pid)
				if err == nil {
					for {
						curr := mt.maxPolledBytes.Load()
						if rss <= curr || mt.maxPolledBytes.CompareAndSwap(curr, rss) {
							break
						}
					}
				}
			}
		}
	}()

	return mt
}

// Stop terminates the tracker and returns peak RSS in megabytes
func (mt *MemoryTracker) Stop() float64 {
	close(mt.stopCh)

	polledBytes := mt.maxPolledBytes.Load()
	if polledBytes > 0 {
		return float64(polledBytes) / (1024 * 1024)
	}

	// Fallback to syscall.Rusage from OS process state
	if mt.cmd.ProcessState != nil {
		if sysUsage := mt.cmd.ProcessState.SysUsage(); sysUsage != nil {
			if rusage, ok := sysUsage.(*syscall.Rusage); ok {
				if runtime.GOOS == "darwin" {
					return float64(rusage.Maxrss) / (1024 * 1024)
				}
				// Linux / Unix Maxrss is in KB
				return float64(rusage.Maxrss) / 1024
			}
		}
	}
	return 0
}

func readProcessRSSBytes(pid int) (uint64, error) {
	if runtime.GOOS != "linux" {
		return 0, nil
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, fmt.Errorf("invalid statm format")
	}
	rssPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return rssPages * uint64(os.Getpagesize()), nil
}
