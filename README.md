# Skillman

A package manager for [Agent Skills](https://agentskills.io).

Skillman manages Agent Skills — install from GitHub or local paths directly into your workspace for any supported AI coding agent.

<p align="center">
<img width="900" alt="terminal" src="https://github.com/user-attachments/assets/a4cc4bf5-7674-4ff5-a109-9665af9c154b" />
<p/>

## Features

- **Install** — Fetch skills from GitHub repos or local directories directly into your workspace
- **Multi-agent** — Supports Claude Code, Cursor, Codex, and GitHub Copilot out of the box
- **Declarative** — `.skillman/config.yml` tracks skills with their sources for easy updates
- **Interactive** — TUI-based skill picker, agent selector, and inline skill review

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

### Install a skill from GitHub

```bash
cd ~/my-project
skillman install github.com/anthropics/skills
```

An interactive picker lets you choose which skills to install. Skills are copied into `.skillman/skills/` and symlinked into each agent's skill directory.

### List installed skills

```bash
skillman list
```
```
NAME    SOURCE                          REF
pdf     github.com/anthropics/skills    main@abc123de
commit  github.com/anthropics/skills    main@abc123de
```

### Configure agents

```bash
skillman config
```

Shows your workspace skills and which agents they're linked to, then lets you toggle agents on or off interactively.

## Command Reference

| Command | Description | Example |
|---------|-------------|---------|
| `install <source>` | Install skills into the current workspace | `skillman install github.com/org/repo` |
| `remove [names...]` | Remove skills from the current workspace | `skillman rm pdf` |
| `update [name]` | Update a skill to the latest version | `skillman update pdf` |
| `list` | List skills in the current workspace | `skillman ls` |
| `config` | View and configure agent symlinks | `skillman config` |

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

URL formats with `https://` and trailing `.git` are normalized automatically.

## Supported Agents

| Agent | Skill Directory |
|-------|----------------|
| Claude Code | `.claude/skills/` |
| Cursor | `.cursor/skills/` |
| Codex | `.codex/skills/` |
| GitHub Copilot | `.github/skills/` |

## Workspace Layout

```
my-project/
├── .skillman/                          # committed to git
│   ├── config.yml                      # tracks skills with sources
│   └── skills/
│       ├── pdf/
│       │   └── SKILL.md
│       └── commit/
│           └── SKILL.md
├── .claude/skills/
│   ├── pdf -> ../../.skillman/skills/pdf       # relative symlink
│   └── commit -> ../../.skillman/skills/commit
└── .cursor/skills/
    ├── pdf -> ../../.skillman/skills/pdf
    └── commit -> ../../.skillman/skills/commit
```

### config.yml

```yaml
skills:
  - name: pdf
    source: github.com/anthropics/skills
    ref: main
    commit: abc123def456
  - name: my-skill
    source: local
    path: /path/to/my-skill
```

## How It Works

1. **Install** clones a GitHub repo (or reads a local path), discovers `SKILL.md` files, copies selected skills into `.skillman/skills/`, and creates relative symlinks in each agent's skill directory
2. **Update** re-fetches the skill from its source, overwrites `.skillman/skills/{name}/`, and updates the config — existing symlinks continue to work
3. **Remove** deletes the skill from `.skillman/skills/`, removes agent symlinks, and cleans up the config
4. **Config** lets you toggle which agents have symlinks for your skills

## License

MIT License - see [LICENSE](LICENSE) for details.
