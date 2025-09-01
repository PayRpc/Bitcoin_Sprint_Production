//go:build windows
// +build windows

package performance

import (
	"fmt"
	"syscall"
)

// setWindowsHighPriority sets high priority on Windows using syscall bindings.
func (pm *PerformanceManager) setWindowsHighPriority() error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setProcessPriorityClass := kernel32.NewProc("SetPriorityClass")
	getCurrentProcess := kernel32.NewProc("GetCurrentProcess")

	const HIGH_PRIORITY_CLASS = 0x00000080

	handle, _, _ := getCurrentProcess.Call()
	ret, _, err := setProcessPriorityClass.Call(handle, HIGH_PRIORITY_CLASS)

	if ret == 0 {
		return fmt.Errorf("SetPriorityClass failed: %v", err)
	}

	// Log at debug level if logger is available
	if pm.logger != nil {
		pm.logger.Debug("Set process to high priority (windows)")
	}
	return nil
}
