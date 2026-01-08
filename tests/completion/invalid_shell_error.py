"""
Test completion error handling for invalid shell.

Expected:
- Invalid shell shows appropriate error
"""

import subprocess


def test_completion_invalid_shell_shows_error(coi_binary):
    """Test that invalid shell shows help with available options."""
    result = subprocess.run(
        [coi_binary, "completion", "invalid-shell-xyz"],
        capture_output=True,
        text=True,
        timeout=5,
    )

    # Cobra shows help for invalid subcommands (returns 0)
    # This is standard Cobra behavior - not an error, just shows usage
    output = result.stdout + result.stderr

    # Should show available shell options in help text
    assert "bash" in output.lower(), "Should show bash as available option"
    assert "zsh" in output.lower(), "Should show zsh as available option"
    assert "fish" in output.lower(), "Should show fish as available option"
    assert "usage:" in output.lower() or "available commands:" in output.lower(), (
        "Should show usage information"
    )
