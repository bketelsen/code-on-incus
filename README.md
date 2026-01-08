# claude-on-incus (`coi`)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mensfeld/claude-on-incus)](https://golang.org/)
[![Latest Release](https://img.shields.io/github/v/release/mensfeld/claude-on-incus)](https://github.com/mensfeld/claude-on-incus/releases)

**The Professional Claude Code Container Runtime for Linux**

Run Claude Code in isolated, production-grade Incus containers with zero permission headaches, perfect file ownership, and true multi-session support.

**Security First:** Unlike Docker or bare-metal execution, your environment variables, SSH keys, and Git credentials are **never** exposed to Claude. Containers run in complete isolation with no access to your host credentials unless explicitly mounted.

*Think Docker for Claude, but with system containers that actually work like real machines.*

## Demo

<!-- Placeholder for asciicast demo - to be added -->

## Features

**Core Capabilities**
- Multi-slot support - Run parallel Claude sessions for the same workspace
- Session persistence - Resume sessions with `.claude` directory restoration
- Persistent containers - Keep containers alive between sessions (installed tools preserved)
- Workspace isolation - Each session mounts your project directory
- **Workspace files persist even in ephemeral mode** - Only the container is deleted, your work is always saved

**Security & Isolation**
- Automatic UID mapping - No permission hell, files owned correctly
- System containers - Full security isolation, better than Docker privileged mode
- Project separation - Complete isolation between workspaces
- **Credential protection** - No risk of SSH keys, `.env` files, or Git credentials being exposed to Claude

**Developer Experience**
- 10 CLI commands - shell, run, build, list, info, attach, images, clean, tmux, version
- Shell completions - Built-in bash/zsh/fish completions via `coi completion`
- Smart configuration - TOML-based with profiles and hierarchy
- Tmux integration - Background processes and session management
- Claude config mounting - Automatic `~/.claude` sync (enabled by default)

**Safe `--dangerous` Flags**
- Claude Code CLI uses `--dangerously-disable-sandbox` and `--dangerously-allow-write-to-root` flags
- **These are safe inside containers** because the "root" is the container root, not your host system
- Containers are ephemeral or isolated - any changes are contained and don't affect your host
- This gives Claude full capabilities while keeping your system protected

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/mensfeld/claude-on-incus/master/install.sh | bash

# Build image (first time only, ~5-10 minutes)
coi build sandbox

# Start coding
cd your-project
coi shell

# That's it! Claude is now running in an isolated container with:
# - Your project mounted at /workspace
# - Correct file permissions (no more chown!)
# - Full Docker access inside the container
# - All workspace changes persisted automatically
# - No access to your host SSH keys, env vars, or credentials
```

## Why Incus Over Docker?

### What is Incus?

Incus is a modern Linux container and virtual machine manager, forked from LXD. Unlike Docker (which uses application containers), Incus provides **system containers** that behave like lightweight VMs with full init systems.

### Key Differences

| Feature | **claude-on-incus (Incus)** | Docker |
|---------|---------------------------|--------|
| **Container Type** | System containers (full OS) | Application containers |
| **Init System** | Full systemd/init | No init (single process) |
| **UID Mapping** | Automatic UID shifting | Manual mapping required |
| **Security** | Unprivileged by default | Often requires privileged mode |
| **File Permissions** | Preserved (UID shifting) | Host UID conflicts |
| **Startup Time** | ~1-2 seconds | ~0.5-1 second |
| **Docker-in-Container** | Native support | Requires DinD hacks |

### Benefits

**No Permission Hell** - Incus automatically maps container UIDs to host UIDs. Files created by Claude in-container have correct ownership on host. No `chown` needed.

**True Isolation** - Full system container means Claude can run Docker, systemd services, etc. Safer than Docker's privileged mode.

**Persistent State** - System containers can be stopped/started without data loss. Ideal for long-running Claude sessions.

**Resource Efficiency** - Share kernel like Docker, lower overhead than VMs, better density for parallel sessions.

## Installation

```bash
# One-shot install (recommended)
curl -fsSL https://raw.githubusercontent.com/mensfeld/claude-on-incus/master/install.sh | bash

# This will:
# - Download and install coi to /usr/local/bin
# - Check for Incus installation
# - Verify you're in incus-admin group
# - Show next steps
```

### Build Images

```bash
# Basic image (5-10 minutes)
coi build sandbox

# Optional: Privileged image with Git/SSH (adds 2-3 minutes)
coi build privileged
```

**What's included:**
- `coi-sandbox`: Ubuntu 22.04 + Docker + Node.js 20 + Claude CLI + tmux
- `coi-privileged`: Everything above + GitHub CLI + SSH + Git config

### Verify Installation

```bash
coi version        # Check version
incus version      # Verify Incus access
groups | grep incus-admin  # Confirm group membership
```

## Usage

### Basic Commands

```bash
# Interactive Claude session
coi shell

# Persistent mode - keep container between sessions
coi shell --persistent

# Use specific slot for parallel sessions
coi shell --slot 2

# Privileged mode (Git/SSH access)
coi shell --privileged

# Resume previous session
coi shell --resume

# Attach to existing session
coi attach

# List active sessions
coi list

# Cleanup
coi clean
```

### Global Flags

```bash
--workspace PATH       # Workspace directory to mount (default: current directory)
--slot NUMBER          # Slot number for parallel sessions (0 = auto-allocate)
--privileged           # Use privileged image (Git/SSH/sudo)
--persistent           # Keep container between sessions
--resume [SESSION_ID]  # Resume from session
--profile NAME         # Use named profile
--env KEY=VALUE        # Set environment variables
```

## Persistent Mode

By default, containers are **ephemeral** (deleted on exit). Your **workspace files always persist** regardless of mode.

Enable **persistent mode** to also keep the container and its installed packages:

**Via CLI:**
```bash
coi shell --persistent
```

**Via config (recommended):**
```toml
# ~/.config/claude-on-incus/config.toml
[defaults]
persistent = true
```

**Benefits:**
- Install once, use forever - `apt install`, `npm install`, etc. persist
- Faster startup - Reuse existing container instead of rebuilding
- Build artifacts preserved - No re-compiling on each session

**What persists:**
- **Ephemeral mode:** Workspace files only (container deleted)
- **Persistent mode:** Workspace files + container state + installed packages

## Configuration

Config file: `~/.config/claude-on-incus/config.toml`

```toml
[defaults]
image = "coi-sandbox"
privileged = false
persistent = true
mount_claude_config = true

[paths]
sessions_dir = "~/.claude-on-incus/sessions"
storage_dir = "~/.claude-on-incus/storage"

[incus]
project = "default"
group = "incus-admin"
claude_uid = 1000

[profiles.rust]
image = "coi-rust"
environment = { RUST_BACKTRACE = "1" }
persistent = true
```

**Configuration hierarchy** (highest precedence last):
1. Built-in defaults
2. System config (`/etc/claude-on-incus/config.toml`)
3. User config (`~/.config/claude-on-incus/config.toml`)
4. Project config (`./.claude-on-incus.toml`)
5. CLI flags

## Use Cases

| Use Case | Problem | Solution |
|----------|---------|----------|
| **Individual Developers** | Multiple projects with different tool versions | Each project gets isolated container with specific tools |
| **Teams** | "Works on my machine" syndrome | Share `.claude-on-incus.toml`, everyone gets identical environment |
| **AI/ML Development** | Need Docker inside container | Incus natively supports Docker-in-container, no DinD hacks |
| **Security-Conscious** | Can't use Docker privileged mode | True isolation without privileged mode |

## Requirements

- **Incus** - Linux container manager
- **Go 1.21+** - For building from source
- **incus-admin group** - User must be in group

## Troubleshooting

### "incus is not available"
```bash
sudo apt update && sudo apt install -y incus
sudo incus admin init --auto
sudo usermod -aG incus-admin $USER
# Log out and back in
```

### "permission denied" errors
```bash
groups | grep incus-admin  # Check membership
sudo usermod -aG incus-admin $USER  # Add yourself
# Log out and back in
```

### Container won't start
```bash
incus info  # Check daemon status
sudo systemctl start incus
```

## Project Status

**Production Ready** - All core features are fully implemented and tested.

**Implemented Features:**
- All CLI commands (shell, run, build, list, info, attach, images, clean, tmux, version)
- Multi-slot parallel sessions
- Session persistence with `.claude` state restoration
- Persistent containers
- Automatic UID mapping
- TOML-based configuration with profiles
- Comprehensive integration test suite

## License

MIT

## Author

Maciej Mensfeld ([@mensfeld](https://github.com/mensfeld))

## See Also

- [FAQ](FAQ.md) - Frequently asked questions
- [CHANGELOG](CHANGELOG.md) - Version history and release notes
- [Integration Tests](INTE.md) - Comprehensive E2E testing documentation
- [Incus](https://linuxcontainers.org/incus/) - Linux container manager
