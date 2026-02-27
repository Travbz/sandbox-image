// Package main implements the sandbox entrypoint binary. It reads control
// plane environment variables, strips them from the environment, drops
// privileges to the agent user, and execs the agent command.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	// Control plane env vars — read then stripped before agent starts.
	envSessionToken    = "SESSION_TOKEN"
	envControlPlaneURL = "CONTROL_PLANE_URL"
	envSessionID       = "SESSION_ID"

	// Agent configuration env vars.
	envAgentCommand = "AGENT_COMMAND"
	envAgentArgs    = "AGENT_ARGS"
	envAgentUser    = "AGENT_USER"
	envAgentWorkdir = "AGENT_WORKDIR"

	defaultUser    = "agent"
	defaultWorkdir = "/workspace"
)

func main() {
	// Read control plane vars into local variables.
	sessionToken := os.Getenv(envSessionToken)
	controlPlaneURL := os.Getenv(envControlPlaneURL)
	sessionID := os.Getenv(envSessionID)

	// Strip control plane vars from the environment. If the agent inspects
	// os.Environ, these are gone.
	os.Unsetenv(envSessionToken)
	os.Unsetenv(envControlPlaneURL)
	os.Unsetenv(envSessionID)

	// Log stripped vars existence (not values) for debugging.
	fmt.Fprintf(os.Stderr, "[entrypoint] session_id=%s control_plane=%s token_present=%t\n",
		sessionID, controlPlaneURL, sessionToken != "")

	// Resolve agent command.
	agentCmd := os.Getenv(envAgentCommand)
	if agentCmd == "" {
		// Fall back to remaining CLI args.
		if len(os.Args) > 1 {
			agentCmd = os.Args[1]
		} else {
			fatal("no agent command specified (set AGENT_COMMAND or pass as argument)")
		}
	}

	// Resolve agent args.
	var agentArgs []string
	if argsStr := os.Getenv(envAgentArgs); argsStr != "" {
		agentArgs = strings.Fields(argsStr)
	} else if len(os.Args) > 2 {
		agentArgs = os.Args[2:]
	}

	// Clean up agent config vars from env.
	os.Unsetenv(envAgentCommand)
	os.Unsetenv(envAgentArgs)

	// Set working directory.
	workdir := os.Getenv(envAgentWorkdir)
	if workdir == "" {
		workdir = defaultWorkdir
	}
	os.Unsetenv(envAgentWorkdir)
	if err := os.Chdir(workdir); err != nil {
		fmt.Fprintf(os.Stderr, "[entrypoint] warning: chdir %s: %v\n", workdir, err)
	}

	// Drop privileges to the agent user.
	targetUser := os.Getenv(envAgentUser)
	if targetUser == "" {
		targetUser = defaultUser
	}
	os.Unsetenv(envAgentUser)
	dropPrivileges(targetUser)

	// Resolve the full path to the agent binary.
	agentBin, err := exec.LookPath(agentCmd)
	if err != nil {
		fatal(fmt.Sprintf("agent binary not found: %s: %v", agentCmd, err))
	}

	// Build the exec argv.
	argv := append([]string{agentBin}, agentArgs...)

	// Build the sanitized environment.
	env := os.Environ()

	fmt.Fprintf(os.Stderr, "[entrypoint] exec: %s %v (workdir=%s)\n", agentBin, agentArgs, workdir)

	// Replace this process with the agent.
	if err := syscall.Exec(agentBin, argv, env); err != nil {
		fatal(fmt.Sprintf("exec %s: %v", agentBin, err))
	}
}

// dropPrivileges changes the process UID/GID to the specified user.
// If the user doesn't exist or the process is not root, it logs a
// warning and continues (allows running as non-root in dev).
func dropPrivileges(username string) {
	if os.Getuid() != 0 {
		fmt.Fprintf(os.Stderr, "[entrypoint] not root (uid=%d), skipping privilege drop\n", os.Getuid())
		return
	}

	uid, gid, err := lookupUser(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[entrypoint] warning: user lookup %q: %v (continuing as root)\n", username, err)
		return
	}

	if err := syscall.Setgroups([]int{gid}); err != nil {
		fmt.Fprintf(os.Stderr, "[entrypoint] warning: setgroups: %v\n", err)
	}
	if err := syscall.Setgid(gid); err != nil {
		fatal(fmt.Sprintf("setgid(%d): %v", gid, err))
	}
	if err := syscall.Setuid(uid); err != nil {
		fatal(fmt.Sprintf("setuid(%d): %v", uid, err))
	}

	fmt.Fprintf(os.Stderr, "[entrypoint] dropped privileges to %s (uid=%d gid=%d)\n", username, uid, gid)
}

// lookupUser resolves a username to UID and GID by reading /etc/passwd.
// This avoids importing os/user which pulls in cgo on some platforms.
func lookupUser(username string) (uid, gid int, err error) {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return 0, 0, fmt.Errorf("reading /etc/passwd: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		if fields[0] == username {
			uid, err := strconv.Atoi(fields[2])
			if err != nil {
				return 0, 0, fmt.Errorf("parsing uid: %w", err)
			}
			gid, err := strconv.Atoi(fields[3])
			if err != nil {
				return 0, 0, fmt.Errorf("parsing gid: %w", err)
			}
			return uid, gid, nil
		}
	}

	return 0, 0, fmt.Errorf("user %q not found in /etc/passwd", username)
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "[entrypoint] fatal: %s\n", msg)
	os.Exit(1)
}
