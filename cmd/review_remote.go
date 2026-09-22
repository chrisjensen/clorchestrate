package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/chrisjensen/clorchestrate/internal/remoteexec"
)

// remoteFileExists returns true if the file exists (may be empty).
func remoteFileExists(server, path string) bool {
	if server == "" {
		_, err := os.Stat(path)
		return err == nil
	}
	return remoteexec.Run(server, "test", "-f", path).Run() == nil
}

// remoteDirExists returns true if path exists and is a directory.
func remoteDirExists(server, path string) bool {
	if server == "" {
		info, err := os.Stat(path)
		return err == nil && info.IsDir()
	}
	return remoteexec.Run(server, "test", "-d", path).Run() == nil
}

// remoteFileNonEmpty returns true if the file exists and has non-zero size.
func remoteFileNonEmpty(server, path string) bool {
	if server == "" {
		info, err := os.Stat(path)
		return err == nil && info.Size() > 0
	}
	return remoteexec.Run(server, "test", "-s", path).Run() == nil
}

// screenSessionRunning returns true if a screen session with the given name exists on the server.
func screenSessionRunning(server, name string) bool {
	if server == "" {
		return false
	}
	return remoteexec.RunShell(server, fmt.Sprintf("screen -ls | grep -qF %s", remoteexec.Quote([]string{"." + name}))).Run() == nil
}

// readRemoteFile reads a file from the server (or locally when server is "").
func readRemoteFile(server, path string) (string, error) {
	var out []byte
	var err error
	if server == "" {
		out, err = os.ReadFile(path)
	} else {
		out, err = remoteexec.Run(server, "cat", path).Output()
	}
	if err != nil {
		return "", fmt.Errorf("read result file %s: %w", path, err)
	}
	return string(out), nil
}

// remoteHomeDir returns the home directory on the given server (empty = local).
func remoteHomeDir(server string) (string, error) {
	var out []byte
	var err error
	if server == "" {
		home, e := os.UserHomeDir()
		return home, e
	}
	// $HOME needs shell expansion, so this can't be argv form; the command
	// string is a fixed literal with no interpolated input.
	out, err = remoteexec.RunShell(server, "echo $HOME").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// expandHome replaces a leading "~" with homeDir.
func expandHome(path, homeDir string) string {
	if strings.HasPrefix(path, "~/") {
		return homeDir + path[1:]
	}
	if path == "~" {
		return homeDir
	}
	return path
}

// remoteGlob runs a shell glob on the server and returns matching paths.
// pattern is intentionally left unquoted so its "*" expands — callers must
// only pass patterns built from SafeName-validated/config-derived parts,
// never unvalidated input.
func remoteGlob(server, pattern string) ([]string, error) {
	shellCmd := fmt.Sprintf("ls -d %s 2>/dev/null || true", pattern)
	out, err := remoteexec.RunShell(server, shellCmd).Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}
