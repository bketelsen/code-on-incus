"""
Pytest configuration and fixtures for CLI integration tests.
"""

import os
import subprocess
import sys

import pytest

# Add tests directory to Python path so 'from support.helpers import ...' works
tests_dir = os.path.dirname(os.path.abspath(__file__))
if tests_dir not in sys.path:
    sys.path.insert(0, tests_dir)


@pytest.fixture(scope="session")
def coi_binary():
    """Return path to coi binary."""
    # Look for coi binary in project root
    binary_path = os.path.join(os.path.dirname(__file__), "..", "coi")
    if not os.path.exists(binary_path):
        pytest.skip("coi binary not found - run 'make build' first")
    return binary_path


@pytest.fixture
def workspace_dir(tmp_path):
    """Provide an isolated temporary workspace directory for each test."""
    # Create a unique workspace directory for this test
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    return str(workspace)


@pytest.fixture
def cleanup_containers(workspace_dir, coi_binary):
    """Cleanup test containers after each test."""
    # Import here to avoid circular imports
    from support.helpers import calculate_container_name, get_container_list

    yield

    # Calculate container names for this workspace (slots 1-10)
    workspace_containers = set()
    for slot in range(1, 11):
        workspace_containers.add(calculate_container_name(workspace_dir, slot))

    # Get all running containers and delete any that belong to this test's workspace
    containers = get_container_list()
    for container in containers:
        if container in workspace_containers:
            subprocess.run(
                [coi_binary, "container", "delete", container, "--force"],
                capture_output=True,
                timeout=30,
                check=False,
            )


@pytest.fixture(scope="session")
def fake_claude_path():
    """Return path to fake Claude CLI for testing.

    This allows tests to run without requiring a real Claude Code license.
    The fake Claude simulates basic Claude behavior for testing container
    orchestration logic.
    """
    fake_path = os.path.join(os.path.dirname(__file__), "..", "testdata", "fake-claude")
    if not os.path.exists(os.path.join(fake_path, "claude")):
        pytest.skip("fake-claude not found")
    return os.path.abspath(fake_path)


@pytest.fixture(scope="session")
def fake_claude_image(coi_binary):
    """Build and return a test image with fake Claude pre-installed.

    This image includes fake Claude at /usr/local/bin/claude, allowing
    tests to run 10x+ faster without requiring a real Claude Code license.

    The image is built once per test session and reused across all tests.
    """
    image_name = "coi-test-fake-claude"

    # Check if image already exists
    result = subprocess.run(
        [coi_binary, "image", "exists", image_name],
        capture_output=True
    )

    if result.returncode == 0:
        return image_name  # Already built

    # Build image with fake Claude
    script_path = os.path.join(
        os.path.dirname(__file__),
        "..",
        "testdata",
        "fake-claude",
        "install.sh"
    )

    if not os.path.exists(script_path):
        pytest.skip(f"Fake Claude install script not found: {script_path}")

    print(f"\nBuilding test image with fake Claude (one-time setup)...")

    result = subprocess.run(
        [coi_binary, "build", "custom", image_name,
         "--script", script_path],
        capture_output=True,
        text=True,
        timeout=300
    )

    if result.returncode != 0:
        pytest.skip(f"Could not build fake Claude image: {result.stderr}")

    print(f"✓ Test image '{image_name}' built successfully")
    return image_name
