# CLAUDE.md

## Python tooling

This project uses **uv** exclusively for environment and dependency management. Do not use `pip`, `pipx`, `pyenv`, or `python -m venv`.

- `uv sync` — install dependencies from the lockfile
- `uv add <pkg>` — add a dependency
- `uv run python <script>` — run a script
- `uv run <cmd>` — run any command inside the project venv
