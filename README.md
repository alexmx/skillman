# Skillman

A lightweight package manager for [Agent Skills](https://agentskills.io) — install skills from GitHub or local paths directly into your workspace, for any AI coding agent.

<p align="center">
<img width="900" alt="terminal" src="https://github.com/user-attachments/assets/a4cc4bf5-7674-4ff5-a109-9665af9c154b" />
</p>

## Features

- **Install anywhere** — Pull skills from GitHub repos or local directories, optionally under a custom name
- **Multi-agent** — Works with Claude Code, Cursor, Codex, and GitHub Copilot out of the box
- **Declarative** — `.skillman/config.yml` tracks every skill and its source, so updates are one command
- **Interactive** — TUI skill picker, agent selector, and inline skill review

## Installation

### Homebrew

```bash
brew install alexmx/tools/skillman
```

### Mise

```bash
mise use --global github:alexmx/skillman
```

## Quick Start

Install a skill from GitHub:

```bash
cd ~/my-project
skillman install github.com/anthropics/skills
```

An interactive picker lets you choose which skills to install. Each one is copied into `.skillman/skills/` and symlinked into every selected agent's skill directory.

List what's installed:

```bash
skillman list
```
```
NAME    SOURCE                          REF
pdf     github.com/anthropics/skills    main@abc123de
commit  github.com/anthropics/skills    main@abc123de
```

Toggle which agents a skill is linked to:

```bash
skillman config
```

## Command Reference

| Command | Description | Example |
|---------|-------------|---------|
| `install <source>` | Install skills into the current workspace | `skillman install github.com/org/repo` |
| `remove [names...]` | Remove skills from the current workspace | `skillman rm pdf` |
| `rename <old> <new>` | Rename an installed skill | `skillman mv pdf acme-pdf` |
| `update [name]` | Update a skill to the latest version | `skillman update pdf` |
| `list` | List skills in the current workspace | `skillman ls` |
| `config` | View and toggle agent symlinks | `skillman config` |

### Install Sources

```bash
# GitHub repository (interactive skill picker)
skillman install github.com/org/repo

# Specific skill from a repository
skillman install github.com/org/repo/skill-name

# Pin to a specific tag or ref
skillman install github.com/org/repo@v1.0

# Local directory
skillman install ./my-skill
```

URLs with `https://` and a trailing `.git` are normalized automatically.

### Custom Names

Install a skill under a different name with `--as` — handy for namespacing with a prefix or avoiding a clash with an existing skill:

```bash
skillman install github.com/org/repo/use-something --as my-use-something
```

You can rename an installed skill at any time. Updates keep tracking the original source:

```bash
skillman rename pdf acme-pdf   # or: skillman mv pdf acme-pdf
```

### Install and Forget

Pass `--no-track` to install a skill without recording it in `.skillman/config.yml`. The skill is set up and ready to use, but skillman won't manage it for `update` or `list`:

```bash
skillman install github.com/org/repo/pdf --no-track
```

## Supported Agents

| Agent | Skill Directory |
|-------|----------------|
| Claude Code | `.claude/skills/` |
| Cursor | `.cursor/skills/` |
| Codex | `.codex/skills/` |
| GitHub Copilot | `.github/skills/` |

## Workspace Layout

Skills live in `.skillman/` (commit it to git) and are symlinked into each agent's directory:

```
my-project/
├── .skillman/
│   ├── config.yml                      # tracks skills with sources
│   └── skills/
│       ├── pdf/SKILL.md
│       └── commit/SKILL.md
├── .claude/skills/
│   ├── pdf -> ../../.skillman/skills/pdf       # relative symlink
│   └── commit -> ../../.skillman/skills/commit
└── .cursor/skills/
    ├── pdf -> ../../.skillman/skills/pdf
    └── commit -> ../../.skillman/skills/commit
```

```yaml
# .skillman/config.yml
skills:
  - name: pdf
    source: github.com/anthropics/skills
    ref: main
    commit: abc123def456
  - name: my-skill
    source: local
    path: /path/to/my-skill
```

## License

MIT License — see [LICENSE](LICENSE) for details.
