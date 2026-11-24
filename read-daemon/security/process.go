package security

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// SetProcessGroup configures the command to start in a new process group.
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}
}

// KillProcessGroup forcefully terminates an entire process group by sending
// SIGTERM followed by SIGKILL to the negative process group ID (-pgid).
func KillProcessGroup(pgid int) error {
	return KillProcessTree(pgid, pgid)
}

// KillProcessTree terminates the root process, all its descendants, and all associated
// process groups (including those created by interactive shell job control).
func KillProcessTree(rootPid int, pgid int) error {
	// Guard against dangerous zero, negative, or PID 1 calls
	if rootPid <= 1 && pgid <= 1 {
		return nil
	}

	// 1. Identify all descendant processes in the tree before sending signals
	descendants := findDescendants(rootPid)

	// Collect all unique process groups among root and descendants
	pgids := make(map[int]bool)
	if pgid > 1 {
		pgids[pgid] = true
	}
	for _, dpid := range descendants {
		if dpgid, err := syscall.Getpgid(dpid); err == nil && dpgid > 1 {
			pgids[dpgid] = true
		}
	}

	// 2. Send SIGTERM first to allow graceful termination
	for g := range pgids {
		_ = syscall.Kill(-g, syscall.SIGTERM)
	}
	for _, dpid := range descendants {
		_ = syscall.Kill(dpid, syscall.SIGTERM)
	}
	if rootPid > 1 {
		_ = syscall.Kill(rootPid, syscall.SIGTERM)
	}

	// Brief grace period
	time.Sleep(50 * time.Millisecond)

	// 3. Send SIGKILL to ensure complete termination
	for g := range pgids {
		_ = syscall.Kill(-g, syscall.SIGKILL)
	}
	for _, dpid := range descendants {
		_ = syscall.Kill(dpid, syscall.SIGKILL)
	}
	if rootPid > 1 {
		_ = syscall.Kill(rootPid, syscall.SIGKILL)
	}

	// 4. Double check /proc for any stubborn processes belonging to these groups or descendants
	if pgid > 1 {
		for _, pid := range findPidsInGroup(pgid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}

	return nil
}

func findDescendants(rootPid int) []int {
	var descendants []int
	if rootPid <= 1 {
		return descendants
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return findDescendantsFallback(rootPid)
	}

	childrenMap := make(map[int][]int)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		statBytes, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}

		statStr := string(statBytes)
		closeParen := strings.LastIndex(statStr, ")")
		if closeParen == -1 || closeParen+2 >= len(statStr) {
			continue
		}

		fields := strings.Fields(statStr[closeParen+2:])
		if len(fields) >= 2 {
			if ppid, err := strconv.Atoi(fields[1]); err == nil {
				childrenMap[ppid] = append(childrenMap[ppid], pid)
			}
		}
	}

	queue := []int{rootPid}
	visited := make(map[int]bool)
	visited[rootPid] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, child := range childrenMap[curr] {
			if !visited[child] {
				visited[child] = true
				descendants = append(descendants, child)
				queue = append(queue, child)
			}
		}
	}

	return descendants
}

// GetProcessGroupMemory calculates total Resident Set Size (RSS) memory in bytes
// consumed by all processes in the specified process group by inspecting /proc.
func GetProcessGroupMemory(pgid int) (uint64, error) {
	if pgid <= 1 {
		return 0, fmt.Errorf("invalid pgid: %d", pgid)
	}

	pageSize := uint64(os.Getpagesize())
	var totalRSS uint64

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return getProcessGroupMemoryFallback(pgid)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pidStr := entry.Name()
		if _, err := strconv.Atoi(pidStr); err != nil {
			continue
		}

		// Read /proc/[pid]/stat to inspect the process group
		statBytes, err := os.ReadFile(filepath.Join("/proc", pidStr, "stat"))
		if err != nil {
			continue
		}

		statStr := string(statBytes)
		closeParen := strings.LastIndex(statStr, ")")
		if closeParen == -1 || closeParen+2 >= len(statStr) {
			continue
		}

		fields := strings.Fields(statStr[closeParen+2:])
		if len(fields) < 3 {
			continue
		}

		// Field 5 of stat (3rd field after ") ") is pgrp
		pgrp, err := strconv.Atoi(fields[2])
		if err != nil || pgrp != pgid {
			continue
		}

		// Read /proc/[pid]/statm to get resident pages (field 2)
		statmBytes, err := os.ReadFile(filepath.Join("/proc", pidStr, "statm"))
		if err != nil {
			continue
		}
		statmFields := strings.Fields(string(statmBytes))
		if len(statmFields) >= 2 {
			rssPages, err := strconv.ParseUint(statmFields[1], 10, 64)
			if err == nil {
				totalRSS += rssPages * pageSize
			}
		}
	}

	return totalRSS, nil
}

func findPidsInGroup(pgid int) []int {
	var pids []int
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return findPidsInGroupFallback(pgid)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		statBytes, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}

		statStr := string(statBytes)
		closeParen := strings.LastIndex(statStr, ")")
		if closeParen == -1 || closeParen+2 >= len(statStr) {
			continue
		}

		fields := strings.Fields(statStr[closeParen+2:])
		if len(fields) >= 3 {
			if pgrp, err := strconv.Atoi(fields[2]); err == nil && pgrp == pgid {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}

func findDescendantsFallback(rootPid int) []int {
	var descendants []int
	out, err := exec.Command("ps", "-A", "-o", "pid,ppid").Output()
	if err != nil {
		return descendants
	}
	childrenMap := make(map[int][]int)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, err1 := strconv.Atoi(fields[0])
			ppid, err2 := strconv.Atoi(fields[1])
			if err1 == nil && err2 == nil {
				childrenMap[ppid] = append(childrenMap[ppid], pid)
			}
		}
	}
	queue := []int{rootPid}
	visited := make(map[int]bool)
	visited[rootPid] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, child := range childrenMap[curr] {
			if !visited[child] {
				visited[child] = true
				descendants = append(descendants, child)
				queue = append(queue, child)
			}
		}
	}
	return descendants
}

func findPidsInGroupFallback(pgid int) []int {
	var pids []int
	out, err := exec.Command("ps", "-A", "-o", "pid,pgid").Output()
	if err != nil {
		return pids
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, err1 := strconv.Atoi(fields[0])
			pgrp, err2 := strconv.Atoi(fields[1])
			if err1 == nil && err2 == nil && pgrp == pgid {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}

func getProcessGroupMemoryFallback(pgid int) (uint64, error) {
	out, err := exec.Command("ps", "-A", "-o", "pgid,rss").Output()
	if err != nil {
		return 0, err
	}
	var totalRSS uint64
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pgrp, err1 := strconv.Atoi(fields[0])
			rssKB, err2 := strconv.ParseUint(fields[1], 10, 64)
			if err1 == nil && err2 == nil && pgrp == pgid {
				totalRSS += rssKB * 1024
			}
		}
	}
	return totalRSS, nil
}

// ParseSignal converts standard signal strings into syscall.Signal.
func ParseSignal(sigName string) syscall.Signal {
	upper := strings.ToUpper(strings.TrimSpace(sigName))
	switch upper {
	case "SIGINT", "INT":
		return syscall.SIGINT
	case "SIGTERM", "TERM":
		return syscall.SIGTERM
	case "SIGHUP", "HUP":
		return syscall.SIGHUP
	case "SIGQUIT", "QUIT":
		return syscall.SIGQUIT
	case "SIGKILL", "KILL":
		return syscall.SIGKILL
	case "SIGWINCH", "WINCH":
		return syscall.SIGWINCH
	case "SIGUSR1", "USR1":
		return syscall.SIGUSR1
	case "SIGUSR2", "USR2":
		return syscall.SIGUSR2
	case "SIGSTOP", "STOP":
		return syscall.SIGSTOP
	case "SIGCONT", "CONT":
		return syscall.SIGCONT
	default:
		return 0
	}
}

// SendSignalToProcessGroup sends the given signal to all processes in pgid.
func SendSignalToProcessGroup(pgid int, sig syscall.Signal) error {
	if pgid <= 1 {
		return fmt.Errorf("invalid process group id: %d", pgid)
	}
	return syscall.Kill(-pgid, sig)
}

const (
	// DefaultFallbackMemoryLimit is 512MB, used strictly when dynamic memory detection fails.
	DefaultFallbackMemoryLimit uint64 = 512 * 1024 * 1024
	// DynamicMemoryPercentage defines the RAM ceiling allocated to child processes (30%).
	DynamicMemoryPercentage uint64 = 30
)

var (
	cachedMemoryLimit uint64
	memoryLimitOnce   sync.Once
)

// CalibrateMemoryLimit reads available system memory at daemon startup and computes
// the child process memory ceiling (30% of available RAM).
// If memory detection fails, it defaults to 512MB and logs a warning.
func CalibrateMemoryLimit() uint64 {
	return CalibrateMemoryLimitFrom("/proc/meminfo")
}

// CalibrateMemoryLimitFrom reads memory metrics from the given path (e.g. /proc/meminfo).
// It searches for MemAvailable first, falling back to MemTotal if MemAvailable is missing.
func CalibrateMemoryLimitFrom(meminfoPath string) uint64 {
	file, err := os.Open(meminfoPath)
	if err != nil {
		log.Printf("[WARN] Dynamic memory detection failed from %s: %v; falling back to 512MB default limit", meminfoPath, err)
		return DefaultFallbackMemoryLimit
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var memAvailableKB, memTotalKB uint64
	foundAvailable := false
	foundTotal := false

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if strings.HasPrefix(fields[0], "MemAvailable:") {
				if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					memAvailableKB = val
					foundAvailable = true
				}
			} else if strings.HasPrefix(fields[0], "MemTotal:") {
				if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					memTotalKB = val
					foundTotal = true
				}
			}
		}
	}

	targetKB := memAvailableKB
	if !foundAvailable {
		if foundTotal {
			targetKB = memTotalKB
		} else {
			log.Printf("[WARN] Dynamic memory detection failed: no MemAvailable or MemTotal found in %s; falling back to 512MB default limit", meminfoPath)
			return DefaultFallbackMemoryLimit
		}
	}

	if targetKB == 0 {
		log.Printf("[WARN] Dynamic memory detection reported 0 kB in %s; falling back to 512MB default limit", meminfoPath)
		return DefaultFallbackMemoryLimit
	}

	// 30% of detected memory in bytes
	limitBytes := (targetKB * 1024 * DynamicMemoryPercentage) / 100
	if limitBytes == 0 {
		return DefaultFallbackMemoryLimit
	}
	return limitBytes
}

// GetDynamicMemoryLimit returns the lazily initialized dynamic memory limit.
func GetDynamicMemoryLimit() uint64 {
	memoryLimitOnce.Do(func() {
		cachedMemoryLimit = CalibrateMemoryLimit()
	})
	return cachedMemoryLimit
}

// SetProcessRlimitAS applies the address space resource limit (RLIMIT_AS) to a child process.
// If limitBytes is 0, the dynamically calibrated memory limit is used.
func SetProcessRlimitAS(pid int, limitBytes uint64) error {
	if limitBytes == 0 {
		limitBytes = GetDynamicMemoryLimit()
	}

	if pid <= 0 || pid == os.Getpid() {
		rlim := syscall.Rlimit{
			Cur: limitBytes,
			Max: limitBytes,
		}
		return syscall.Setrlimit(syscall.RLIMIT_AS, &rlim)
	}

	// For child processes, apply via SYS_PRLIMIT64
	type rlimit64 struct {
		Cur uint64
		Max uint64
	}
	newLim := rlimit64{
		Cur: limitBytes,
		Max: limitBytes,
	}
	_, _, errno := syscall.RawSyscall6(
		syscall.SYS_PRLIMIT64,
		uintptr(pid),
		uintptr(syscall.RLIMIT_AS),
		uintptr(unsafe.Pointer(&newLim)),
		0, 0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
