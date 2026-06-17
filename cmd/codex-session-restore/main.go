package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/perlinson/codex-session-restore/internal/app"
)

func main() {
	args := os.Args[1:]
	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)

	if len(args) == 0 {
		args = []string{"switch-current-provider"}
		if logFile, err := openLogFile(); err == nil {
			defer logFile.Close()
			fmt.Fprintf(logFile, "\n[%s] codex-session-restore double-click run\n", time.Now().Format(time.RFC3339))
			stdout = io.MultiWriter(os.Stdout, logFile)
			stderr = io.MultiWriter(os.Stderr, logFile)
		}
	}

	if err := app.Run(args, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func openLogFile() (*os.File, error) {
	exe, err := os.Executable()
	if err == nil {
		if file, fileErr := os.OpenFile(filepath.Join(filepath.Dir(exe), "codex-session-restore.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); fileErr == nil {
			return file, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(home, "codex-session-restore.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}
