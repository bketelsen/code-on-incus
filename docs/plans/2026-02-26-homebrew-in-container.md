# Add Homebrew to coi Container Image — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Install Homebrew (Linuxbrew) in the coi container image so users have `brew` available inside `coi shell`.

**Architecture:** Add a new `install_homebrew()` function to `scripts/build/coi.sh` using the official Homebrew installer run non-interactively as the `code` user. Follows the same pattern as `install_claude_cli()` and `install_opencode()`.

**Tech Stack:** Bash, Homebrew installer, Linuxbrew

---

### Task 1: Add `install_homebrew()` function to coi.sh

**Files:**
- Modify: `scripts/build/coi.sh:11` (header comment)
- Modify: `scripts/build/coi.sh:288-289` (insert new function after `install_github_cli`)
- Modify: `scripts/build/coi.sh:358-359` (add call in `main()`)

**Step 1: Update the header comment**

In `scripts/build/coi.sh`, the header lists installed components (lines 5-11). Add `# - Homebrew` to the list:

```bash
# It installs all dependencies needed for CLI tool execution:
# - Base development tools
# - Node.js LTS
# - Claude CLI
# - Docker
# - GitHub CLI
# - Homebrew
# - dummy (test stub for testing)
```

**Step 2: Add the `install_homebrew()` function**

Insert this function after the `install_github_cli()` function (after line 288) and before `configure_tmp_cleanup()`:

```bash
#######################################
# Install Homebrew (Linuxbrew)
# See: https://brew.sh
#######################################
install_homebrew() {
    log "Installing Homebrew..."

    # Homebrew on Linux needs procps and file beyond our base deps
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq procps file

    # Run the official installer non-interactively as the code user
    su - "$CODE_USER" -c 'NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'

    # Verify installation
    local BREW_PATH="/home/linuxbrew/.linuxbrew/bin/brew"
    if [[ ! -x "$BREW_PATH" ]]; then
        log "ERROR: brew not found at $BREW_PATH after installation."
        log "Installation may have failed or installed to an unexpected location."
        exit 1
    fi

    # Add brew to code user's shell environment (sets PATH, MANPATH, INFOPATH)
    echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> "/home/$CODE_USER/.bashrc"

    log "Homebrew $($BREW_PATH --version 2>/dev/null | head -1 || echo 'installed')"
}
```

**Step 3: Add `install_homebrew` call to `main()`**

In the `main()` function, add the call after `install_github_cli` and before `cleanup`:

```bash
    install_github_cli
    install_homebrew
    cleanup
```

**Step 4: Review the full diff**

Run: `git diff scripts/build/coi.sh`

Verify:
- Header comment includes `# - Homebrew`
- New function follows existing patterns (log, verify, error handling)
- Call is in the right position in `main()`

**Step 5: Commit**

```bash
git add scripts/build/coi.sh
git commit -m "feat: add Homebrew (Linuxbrew) to container image"
```
