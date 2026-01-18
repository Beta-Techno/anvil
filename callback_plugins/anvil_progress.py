"""Anvil progress callback plugin.

Emits structured markers that the anvil CLI parses to display clean progress.
Markers:
  @@ROLE:name@@     - Starting a new role
  @@TASK:name@@     - Starting a new task (only emitted for non-role tasks)
  @@OK@@            - Task completed successfully
  @@FAIL:message@@  - Task failed with message
  @@DONE@@          - Playbook complete
"""

from __future__ import absolute_import, division, print_function

__metaclass__ = type

DOCUMENTATION = """
    name: anvil_progress
    type: stdout
    short_description: Clean progress output for anvil CLI
    description:
        - Emits structured markers for the anvil CLI to parse
        - Suppresses verbose task output for clean display
"""

import sys
from ansible.plugins.callback import CallbackBase


class CallbackModule(CallbackBase):
    CALLBACK_VERSION = 2.0
    CALLBACK_TYPE = "stdout"
    CALLBACK_NAME = "anvil_progress"

    def __init__(self):
        super(CallbackModule, self).__init__()
        self._current_role = None
        self._task_count = 0
        self._failed = False

    def _emit(self, marker):
        """Emit a marker and flush immediately."""
        print(marker, file=sys.stdout, flush=True)

    def v2_playbook_on_play_start(self, play):
        pass  # suppress

    def v2_playbook_on_task_start(self, task, is_conditional):
        self._task_count += 1
        role = task._role
        if role:
            role_name = role.get_name()
            if role_name != self._current_role:
                self._current_role = role_name
                self._emit(f"@@ROLE:{role_name}@@")
        else:
            # Non-role task (pre_tasks, post_tasks, etc.)
            task_name = task.get_name()
            if task_name and not task_name.startswith("Gathering"):
                self._emit(f"@@TASK:{task_name}@@")

    def v2_runner_on_ok(self, result):
        pass  # suppress normal ok output

    def v2_runner_on_skipped(self, result):
        pass  # suppress skipped output

    def v2_runner_on_unreachable(self, result):
        host = result._host.get_name()
        self._emit(f"@@FAIL:Host {host} unreachable@@")
        self._failed = True

    def v2_runner_on_failed(self, result, ignore_errors=False):
        if ignore_errors:
            return
        self._failed = True
        task_name = result._task.get_name()
        msg = result._result.get("msg", "")
        if not msg:
            msg = result._result.get("stderr", "")
        if not msg:
            msg = str(result._result)[:200]
        # Clean up the message
        msg = msg.replace("\n", " ").strip()
        self._emit(f"@@FAIL:{task_name}: {msg}@@")

    def v2_playbook_on_stats(self, stats):
        self._emit("@@DONE@@")

    def v2_on_any(self, *args, **kwargs):
        pass  # suppress everything else
