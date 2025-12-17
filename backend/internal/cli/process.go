package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/ProjAnvil/knot/backend/internal/config"
)

// SavePID saves the process ID to the PID file
func SavePID(pid int) error {
	pidPath := config.GetPIDPath()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPID reads the process ID from the PID file
func ReadPID() (int, error) {
	pidPath := config.GetPIDPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // No PID file exists
		}
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// RemovePID removes the PID file
func RemovePID() error {
	pidPath := config.GetPIDPath()
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, signal 0 checks process existence without actually sending a signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// KillProcess kills a process with the given PID
func KillProcess(pid int, graceful bool) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	if graceful {
		// Try graceful shutdown with SIGTERM
		return process.Signal(syscall.SIGTERM)
	}

	// Force kill with SIGKILL
	return process.Signal(syscall.SIGKILL)
}

// GetServerBinaryPath returns the path to the server binary
func GetServerBinaryPath() (string, error) {
	// First, try to find the server binary in the same directory as the CLI
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Check for server binary in the same directory
	serverPath := execPath[:len(execPath)-len("knot")] + "knot-server"
	if _, err := os.Stat(serverPath); err == nil {
		return serverPath, nil
	}

	// Try alternative name patterns
	alternatives := []string{
		serverPath + "-macos-arm64",
		serverPath + "-macos",
		serverPath + "-linux",
		serverPath + ".exe",
	}

	for _, alt := range alternatives {
		if _, err := os.Stat(alt); err == nil {
			return alt, nil
		}
	}

	return "", fmt.Errorf("server binary not found")
}

// StartServer starts the server in the background
func StartServer(serverPath string, port int, host string) (int, error) {
	logPath := config.GetLogPath()

	// Ensure log directory exists
	logDir := config.GetLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Create command
	cmd := exec.Command(serverPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("HOST=%s", host),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start server: %w", err)
	}

	return cmd.Process.Pid, nil
}

// StartServerWithServeCommand starts the CLI binary with the __serve command in background
func StartServerWithServeCommand(cliPath string, port int, host string) (int, error) {
	logPath := config.GetLogPath()

	// Ensure log directory exists
	logDir := config.GetLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Create command with __serve subcommand
	cmd := exec.Command(cliPath, "__serve")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("HOST=%s", host),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start server: %w", err)
	}

	return cmd.Process.Pid, nil
}
