// +build !windows

package performance

// setWindowsHighPriority is a no-op on non-Windows platforms.
func (pm *PerformanceManager) setWindowsHighPriority() error {
    // Not supported on this OS; return nil so callers don't fail.
    if pm.logger != nil {
        pm.logger.Debug("Set process to high priority skipped: not windows")
    }
    return nil
}
