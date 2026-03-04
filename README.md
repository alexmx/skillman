# Skillman

A package manager for [Agent Skills](https://agentskills.io).

Skillman manages Agent Skills — install from GitHub or local paths into a central store, then link them into any workspace for any supported AI coding agent.

<p align="center">
<img width="800" alt="terminal" src="https://github.com/user-attachments/assets/cf3e3385-eac7-4bef-ab09-40c322bf357a" />
<p/>

## Features

- **Install** — Fetch skills from GitHub repos or local directories into a central store
- **Link** — Symlink skills from the store into workspace agent directories
- **Multi-agent** — Supports Claude Code, Cursor, Codex, and GitHub Copilot out of the box
- **Declarative** — Define skills in `.skillman.yml` and sync across workspaces
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
skillman install github.com/anthropics/skills
```

An interactive picker lets you choose which skills to install.

### Link skills into your workspace

```bash
cd ~/my-project
skillman link
```

Skillman detects which agents are configured in your workspace and lets you pick which ones to link. The selection is saved to `.skillman.yml`.

### Check workspace status

```bash
skillman status
```
```
Global:
  Config file:  ~/.config/skillman/config.toml
  Store path:   ~/.local/share/skillman/store
  Skills:       3 installed (run 'skillman list' to see all)

Workspace:
  Path: /Users/you/my-project
  Declared: 2 skills in .skillman.yml
  Linked: 2 skills
    pdf                  -> claude, cursor (~/.local/share/skillman/store/github.com/anthropics/skills/pdf)
    commit               -> claude (~/.local/share/skillman/store/github.com/anthropics/skills/commit)
```

## Command Reference

### Managing Skills

| Command | Description | Example |
|---------|-------------|---------|
| `install <source>` | Install skills from GitHub or a local path | `skillman install github.com/org/repo` |
| `list` | List all installed skills | `skillman list` |
| `update [name]` | Update a skill to the latest version | `skillman update pdf` |
| `remove <name>` | Remove a skill from the store | `skillman remove pdf` |

### Workspace Operations

| Command | Description | Example |
|---------|-------------|---------|
| `link [names...]` | Link skills into the current workspace | `skillman link pdf commit` |
| `unlink [names...]` | Unlink skills from the current workspace | `skillman unlink pdf` |
| `sync` | Sync workspace symlinks with `.skillman.yml` | `skillman sync` |
| `status` | Show status for the current workspace | `skillman status` |

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

When linking, Skillman detects which agents are configured in your workspace and prompts you to select which ones to link to.

## Configuration

Skillman follows the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) specification:

- **Config:** `~/.config/skillman/config.toml`
- **Store:** `~/.local/share/skillman/store`
- **Registry:** `~/.local/share/skillman/store/registry.json`

Override paths with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` environment variables, or set `store_path` in `config.toml`.

### config.toml

```toml
# Override the default store path
store_path = "/custom/path/to/store"

# Configure agents
[agents.claude]
enabled = true
skill_path = ".claude/skills"

[agents.cursor]
enabled = true
skill_path = ".cursor/skills"
```

## Workspace Config

Create a `.skillman.yml` in your project root to declare which skills should be linked:

```yaml
skills:
  - pdf
  - commit
```

Run `skillman sync` to ensure workspace symlinks match the declared list. The `link` command automatically updates this file.

## How It Works

1. **Install** clones a GitHub repo (or reads a local path), discovers `SKILL.md` files, and copies selected skills into the central store at `~/.local/share/skillman/store/`
2. **Link** creates symlinks from the store into your workspace's agent skill directories (e.g., `.claude/skills/pdf` -> `~/.local/share/skillman/store/github.com/org/repo/pdf`)
3. **Update** re-clones the source repo, compares commit SHAs, and replaces the stored copy if newer
4. **Sync** reads `.skillman.yml` and reconciles symlinks — linking declared skills and removing stale ones

Skills are stored once and shared across all workspaces via symlinks.

## License

MIT License - see [LICENSE](LICENSE) for details.
