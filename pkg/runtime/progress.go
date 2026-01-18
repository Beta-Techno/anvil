package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProgressRenderer handles parsing callback markers and rendering clean output.
type ProgressRenderer struct {
	logFile     *os.File
	currentRole string
	spinner     int
	lastUpdate  time.Time
	failed      bool
	failMsg     string
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgressRenderer creates a renderer that also logs to a file.
func NewProgressRenderer() (*ProgressRenderer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	logPath := filepath.Join(home, ".config", "anvil", "last-run.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	return &ProgressRenderer{
		logFile:    logFile,
		lastUpdate: time.Now(),
	}, nil
}

// LogPath returns the path to the log file.
func (p *ProgressRenderer) LogPath() string {
	if p.logFile != nil {
		return p.logFile.Name()
	}
	return ""
}

// Close closes the log file.
func (p *ProgressRenderer) Close() {
	if p.logFile != nil {
		p.logFile.Close()
	}
}

// Failed returns true if any task failed.
func (p *ProgressRenderer) Failed() bool {
	return p.failed
}

// FailMessage returns the failure message if any.
func (p *ProgressRenderer) FailMessage() string {
	return p.failMsg
}

// ProcessOutput reads from r, parses markers, renders progress, and logs everything.
func (p *ProgressRenderer) ProcessOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// Log everything
		if p.logFile != nil {
			fmt.Fprintln(p.logFile, line)
		}
		// Parse markers
		p.processLine(line)
	}
}

func (p *ProgressRenderer) processLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Parse markers
	if strings.HasPrefix(line, "@@") && strings.HasSuffix(line, "@@") {
		marker := line[2 : len(line)-2]
		p.handleMarker(marker)
		return
	}

	// Non-marker line - could be error output, log it but don't display
}

func (p *ProgressRenderer) handleMarker(marker string) {
	switch {
	case strings.HasPrefix(marker, "ROLE:"):
		roleName := marker[5:]
		p.showRoleProgress(roleName)

	case strings.HasPrefix(marker, "TASK:"):
		taskName := marker[5:]
		p.showTaskProgress(taskName)

	case strings.HasPrefix(marker, "FAIL:"):
		msg := marker[5:]
		p.showFailure(msg)

	case marker == "DONE":
		p.showComplete()

	case marker == "OK":
		// Individual task OK - we don't show these, just role transitions
	}
}

func (p *ProgressRenderer) showRoleProgress(roleName string) {
	// Complete previous role if any
	if p.currentRole != "" {
		p.clearLine()
		fmt.Printf("  \033[32m✓\033[0m %s\n", p.currentRole)
	}
	p.currentRole = roleName
	p.updateSpinner()
}

func (p *ProgressRenderer) showTaskProgress(taskName string) {
	// For non-role tasks, show them directly
	if p.currentRole == "" {
		p.clearLine()
		fmt.Printf("  → %s...\n", taskName)
	}
}

func (p *ProgressRenderer) showFailure(msg string) {
	p.failed = true
	p.failMsg = msg
	p.clearLine()
	if p.currentRole != "" {
		fmt.Printf("  \033[31m✗\033[0m %s\n", p.currentRole)
	}
	fmt.Printf("    \033[31m%s\033[0m\n", msg)
}

func (p *ProgressRenderer) showComplete() {
	// Complete the last role
	if p.currentRole != "" && !p.failed {
		p.clearLine()
		fmt.Printf("  \033[32m✓\033[0m %s\n", p.currentRole)
	}
	p.currentRole = ""
}

func (p *ProgressRenderer) updateSpinner() {
	p.spinner = (p.spinner + 1) % len(spinnerFrames)
	p.clearLine()
	fmt.Printf("  %s %s...", spinnerFrames[p.spinner], p.currentRole)
}

func (p *ProgressRenderer) clearLine() {
	fmt.Print("\r\033[K")
}

// Tick updates the spinner animation. Call this periodically.
func (p *ProgressRenderer) Tick() {
	if p.currentRole != "" && time.Since(p.lastUpdate) > 100*time.Millisecond {
		p.updateSpinner()
		p.lastUpdate = time.Now()
	}
}
