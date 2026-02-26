# Add Homebrew to coi Container Image

## Goal

Install Homebrew (Linuxbrew) inside the coi container image so users have `brew` available when they `coi shell` in.

## Approach

Use the official Homebrew installer run non-interactively as the `code` user. This matches the existing pattern used for Claude CLI and opencode in `scripts/build/coi.sh`.

## Changes

### `scripts/build/coi.sh`

**New function `install_homebrew()`:**

1. Install additional apt dependencies needed by Homebrew on Linux: `procps`, `file`
2. Run the official installer as the `code` user with `NONINTERACTIVE=1`
3. Verify the binary exists at `/home/linuxbrew/.linuxbrew/bin/brew`
4. Append `eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"` to the code user's `.bashrc` to set up PATH, MANPATH, and INFOPATH

**Placement in `main()`:** After `install_github_cli`, before `cleanup`.

**Header comment:** Add `# - Homebrew` to the list of installed components.

## What this doesn't change

- No pre-installed brew packages (users install what they need at runtime)
- No changes to cleanup, Makefile, install.sh, or any other file
- No new configuration files

## Trade-offs

- Image size increases (~400MB+ for the Homebrew git repos and portable Ruby)
- Build time increases by a few minutes
- Users gain access to Homebrew's full package ecosystem inside the container
