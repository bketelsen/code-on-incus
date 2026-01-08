"""
Test that --mount-claude-config=false shows initial Claude setup.

Flow:
1. Start session with --mount-claude-config=false
2. Verify initial Claude setup prompts appear
3. Verify we see configuration questions (text style, bypass permissions)
4. Exit session

Expected:
- Initial Claude setup prompts appear
- Configuration questions are shown
- No pre-configured Claude environment
"""

import time

from support.helpers import (
    assert_clean_exit,
    exit_claude,
    spawn_coi,
    wait_for_container_ready,
    wait_for_text_in_monitor,
    with_live_screen,
)


def test_no_mount_shows_initial_setup(coi_binary, cleanup_containers, workspace_dir):
    """Test that --mount-claude-config=false shows initial Claude setup prompts."""

    # Start session with --mount-claude-config=false
    child = spawn_coi(
        coi_binary, ["shell", "--tmux=true", "--mount-claude-config=false"], cwd=workspace_dir
    )

    wait_for_container_ready(child)

    with with_live_screen(child) as monitor:
        # Wait for initial setup to appear
        time.sleep(10)

        # Check for initial Claude setup prompts by looking for the text style option
        # The setup screen shows "Light mode", "Dark mode", etc.
        setup_found = (
            wait_for_text_in_monitor(monitor, "Light mode", timeout=30)
            or wait_for_text_in_monitor(monitor, "Dark mode", timeout=5)
        )

        if not setup_found:
            # Debug: print what we actually see
            print(f"\nScreen content:\n{monitor.last_display}\n")

        assert setup_found, (
            "Expected initial Claude setup prompts to appear with --mount-claude-config=false, "
            "but they were not found"
        )

        # Exit Claude - use Ctrl+C since we're in setup, not at prompt
        time.sleep(1)
        clean_exit = exit_claude(child, use_ctrl_c=True, timeout=30)

    assert_clean_exit(clean_exit, child)
