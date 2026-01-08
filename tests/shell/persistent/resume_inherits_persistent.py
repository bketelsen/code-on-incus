"""
Test that resuming a persistent session without --persistent still keeps it persistent.

Flow:
1. Start first session with --persistent
2. Exit session (container stays)
3. Resume session with --resume but WITHOUT --persistent
4. Exit session
5. Verify container STILL exists (inherited persistent mode)
6. Cleanup

Expected:
- Persistent flag is inherited from session metadata
- Container persists even when --persistent not passed to --resume
"""

import subprocess
import time

from support.helpers import (
    assert_clean_exit,
    calculate_container_name,
    exit_claude,
    send_prompt,
    spawn_coi,
    wait_for_container_ready,
    wait_for_prompt,
    wait_for_text_in_monitor,
    with_live_screen,
)


def test_persistent_resume_inherits_persistent_mode(
    coi_binary, cleanup_containers, workspace_dir
):
    """Test that resuming a persistent session without --persistent still keeps it persistent."""

    # First session with --persistent
    child1 = spawn_coi(
        coi_binary, ["shell", "--tmux=true", "--persistent", "--slot=22"], cwd=workspace_dir
    )
    wait_for_container_ready(child1)
    container_name = calculate_container_name(workspace_dir, 22)
    wait_for_prompt(child1)

    with with_live_screen(child1) as monitor1:
        time.sleep(2)
        send_prompt(child1, "Print 20 times ONLY result of sum of 5000 and 7000 and NOTHING ELSE")
        responded = wait_for_text_in_monitor(monitor1, "12000", timeout=30)
        assert responded, "Claude did not respond in first session"

        # Exit first session
        time.sleep(1)
        clean_exit = exit_claude(child1, timeout=90)
        time.sleep(3)

    assert_clean_exit(clean_exit, child1)

    # Verify container exists and is stopped
    result = subprocess.run(
        ["sg", "incus-admin", "-c", f"incus list {container_name} --format=csv"],
        capture_output=True,
        text=True,
        shell=False,
    )
    assert container_name in result.stdout, f"Container {container_name} should exist"

    # Second session with --resume but WITHOUT --persistent
    # Should inherit persistent mode from session metadata (auto-detect latest session)
    child2 = spawn_coi(
        coi_binary,
        ["shell", "--tmux=true", "--resume"],
        cwd=workspace_dir,
    )
    wait_for_container_ready(child2)
    # Give extra time for Claude to load from restored session
    time.sleep(5)
    wait_for_prompt(child2)

    with with_live_screen(child2) as monitor2:
        time.sleep(2)
        send_prompt(
            child2, "Print 20 times ONLY result of sum of 8000 and 9000 and NOTHING ELSE"
        )
        responded = wait_for_text_in_monitor(monitor2, "17000", timeout=30)
        assert responded, "Claude did not respond in resumed session"

        # Exit resumed session
        time.sleep(1)
        clean_exit2 = exit_claude(child2, timeout=90)
        time.sleep(3)

    assert_clean_exit(clean_exit2, child2)

    # Verify container STILL EXISTS (should have inherited persistent mode)
    result = subprocess.run(
        ["sg", "incus-admin", "-c", f"incus list {container_name} --format=csv"],
        capture_output=True,
        text=True,
        shell=False,
    )
    assert container_name in result.stdout, (
        "Container should still exist (inherited persistent mode)"
    )

    # Cleanup
    subprocess.run(
        ["sg", "incus-admin", "-c", f"incus delete --force {container_name}"],
        check=False,
        shell=False,
    )
