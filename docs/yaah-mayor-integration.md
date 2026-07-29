# yaah × Gas Town: Mayor Integration Requirements

This document specifies the features yaah must support to act as a **mayor**
in Gas Town. The mayor is the primary orchestrator that runs in a tmux session,
dispatches crew sub-agents, manages rigs, and serves as the town's "overseer"
for human-directed work.

**Status**: This doc is based on yaah `v0.x` source code at
`C:\Code\Personal\agentic\yaah`.

## Reference: How Other Agents Integrate

Gas Town already supports these agents as mayors/crew:

| Agent | Preset | ACP | Hooks | Process Names | Resume |
|---|---|---|---|---|---|
| Claude Code | `claude` | via `claude-agent-acp` | `.claude/` settings + hooks | `node`, `claude` | `--resume` / `--continue` |
| OpenCode | `opencode` | `opencode acp` | `.opencode/plugins/*.js` | `opencode`, `node`, `bun` | internal sessions |
| Codex | `codex` | no | none | `codex` | `resume` subcommand |
| Gemini | `gemini` | `gemini --acp` | none | `gemini` | `--resume` |
| Copilot | `copilot` | no | `.github/hooks/*.json` | `copilot` | `--resume` / `--continue` |

The mayor integration has **two modes**:

1. **tmux mode** (default): yaah runs inside a tmux pane. Gas Town manages
   the tmux session lifecycle (start, stop, restart, attach). yaah must be
   interactive but capable of autonomous multi-turn operation.
2. **MCP serve mode** (yaah-specific): yaah runs as an MCP tool server over
   stdio or HTTP. Gas Town's `acp.Proxy` (or a custom MCP client) communicates
   with yaah via JSON-RPC. This mode enables tighter integration than ACP
   because yaah already implements the MCP spec natively.

---

## Required Features

### 1. CLI Invocation

Gas Town spawns yaah as a subprocess. It needs to know:

| Field | Requirement | Example |
|---|---|---|
| `command` | Binary name or full path | `yaah` |
| `args` | Default args for autonomous mode | `[]` (no args needed) |
| `env` | Env vars to set at spawn time | `{"YAHC_APPROVAL": "allow"}` |

**yaah CLI modes:**

| Command | Purpose |
|---|---|
| `yaah` | Interactive REPL (tmux mode) |
| `yaah "prompt"` | One-shot prompt, prints response, exits |
| `yaah serve` | MCP tool server over stdio (MCP mode) |
| `yaah serve --http :7333` | MCP tool server over Streamable HTTP |
| `yaah session list` | List recent sessions |
| `yaah session show <id>` | Show session messages |

**Constraints:**
- yaah accepts prompts as positional CLI args (one-shot) or via MCP `prompt` tool.
- yaah must not require a TTY for non-interactive runs (tmux will allocate a
  pseudo-TTY, but MCP mode uses stdio pipes).
- yaah exits cleanly on SIGINT/SIGTERM so tmux session teardown works.

### 2. Process Detection

Gas Town monitors the tmux pane to detect if yaah is alive, crashed, or
waiting for input. It checks `pane_current_command` in tmux.

| Field | Requirement |
|---|---|
| `process_names` | List of process names yaah appears as in `ps`/`tmux` |

**Expected values for yaah:**
- Primary: `yaah`

yaah is a single static Go binary — it does not shell out to Node.js or Bun.
Gas Town uses the process name in `tmux.IsAgentRunning()` to decide whether to
respawn a dead session.

### 3. Session Resume

The mayor may be restarted (container reboot, `gt mayor attach`, crash
recovery). Gas Town needs to resume the previous conversation.

| Field | Requirement |
|---|---|
| `resume_flag` | Flag to resume a specific session | `--resume <id>` |
| `continue_flag` | Flag to resume most recent session | _(not implemented)_ |
| `resume_style` | `"flag"` |
| `session_id_env` | Env var that yaah sets with its session ID | _(not implemented — see below)_ |

**yaah session storage:**
- Sessions are stored in a SQLite database at `~/.yaah/memory.db` (or
  `$YAAH_HOME/memory.db`).
- Session IDs are generated as `sess-<unix-nano>` (e.g., `sess-1715000000000000000`).
- Resume: `yaah --resume <session-id>` restores the full conversation history
  including compacted summaries.
- Session list: `yaah session list` shows recent sessions with timestamps,
  models, and token counts.
- No "continue" flag exists — users must specify a session ID explicitly.

**Gas Town integration note:**
- yaah does **not** export its session ID via an environment variable.
- Gas Town can read the session ID from the SQLite database directly, or
  capture it from yaah's stdout on startup.
- Alternatively, Gas Town can pass a known session ID via `--resume` and let
  yaah create it if it doesn't exist.

### 4. Hooks / Lifecycle Integration

Gas Town injects startup context (prime formula, nudge queue, escalation
rules) via hooks or startup files. yaah provides **JSONL hook events** — a
passive event log, not an active hook system like Claude Code's.

#### Option A: JSONL hook events (best effort)

yaah emits structured events to a configurable directory:

| Field | Requirement |
|---|---|
| `supports_hooks` | `true` |
| `hooks_provider` | `"yaah"` |
| `hooks_dir` | Relative path for hook files, e.g. `.yaah/hooks` |
| `hooks_settings_file` | Not applicable — yaah uses `config.yaml` |

**Events yaah emits:**

| Event | When |
|---|---|
| `session.start` | Session begins |
| `session.end` | Session ends |
| `turn.start` | Each user turn |
| `tool.start` | Tool call begins |
| `tool.end` | Tool call completes |
| `conflict.check` | Conflict detection runs |
| `conflict.detect` | Conflict detected |
| `context.prune` | Context compaction runs |

**Limitations:**
- Events are **write-only** — yaah reads from a directory.
- Gas Town cannot inject behavior via these files; it can only observe.
- The hook directory is set in `config.yaml` under `hooks.dir`.

#### Option B: AGENTS.md / instructions file (recommended)

yaah reads `AGENTS.md`, `CLAUDE.md`, or `CONTEXT.md` from the working
directory and injects them into the system prompt. This is the primary
mechanism for Gas Town to deliver startup context.

| Field | Requirement |
|---|---|
| `instructions_file` | File name to read for startup context | `"AGENTS.md"` |

**Behavior:**
- yaah walks up from `cwd` to the worktree root, collecting instruction files.
- `AGENTS.md` is preferred; `CLAUDE.md` is accepted for cross-tool compat.
- Files are injected into the system prompt at session start.
- Changes to instruction files are picked up on the next session start (not
  dynamically during a running session).

### 5. Non-Interactive / Headless Mode

The mayor runs unattended for long periods. yaah must support running without
a human at the keyboard.

| Field | Requirement |
|---|---|
| `non_interactive.subcommand` | `"serve"` (MCP mode) or one-shot `"yaah \"prompt\""` |
| `non_interactive.output_flag` | Not needed — MCP mode returns structured JSON |
| `non_interactive.prompt_flag` | Not needed — prompt is a positional arg |

**Expected behavior:**
- `yaah "prompt here"` executes one or more turns and exits.
- `yaah serve` starts an MCP server that accepts `prompt` tool calls over
  stdio/HTTP. Conversation state persists across calls.
- `yaah serve --http :7333` same thing over HTTP.

If yaah only supports interactive TUI mode, Gas Town can still drive it via
tmux input simulation, but MCP mode is strongly preferred.

### 6. Permission / Approval Model

The mayor runs in a sandboxed container with `IS_SANDBOX=1`. yaah must be
configurable to auto-approve tool calls without human intervention.

| Requirement | Details |
|---|---|
| Auto-approve all tools | yaah supports `approval: allow` in config |
| No IDE integration required | The mayor runs in a container, not an IDE |
| Tool allowlist | yaah does not have a tool-level allowlist, but the `--approval` flag controls global mode |

**yaah approval modes:**
- `ask` (default): prompts for approval on each tool call.
- `allow`: auto-approves all tool calls.
- `deny`: blocks all tool calls.

**Configuration precedence:**
1. CLI flag: `--approval allow`
2. Env var: `YAAH_APPROVAL=allow`
3. Config file: `agents.default.approval: allow`

For the mayor, Gas Town should set `YAAH_APPROVAL=allow` at spawn time.

**Sub-agent note:** Sub-agents spawned via the `task` tool always run in
`allow` mode regardless of the parent's approval setting.

### 7. MCP Support (Preferred over ACP)

yaah does **not** implement ACP. Instead, it implements the **MCP (Model
Context Protocol)** server interface, which is actually better for Gas Town
integration because:

- yaah's `serve` subcommand exposes an MCP server over stdio or HTTP.
- The server provides three tools: `prompt`, `traces`, `status`.
- Conversation state persists across `prompt` calls for the lifetime of the
  server.
- Gas Town can drive yaah programmatically without tmux.

| Requirement | Details |
|---|---|
| MCP stdio transport | `yaah serve` — newline-delimited JSON-RPC 2.0 on stdin/stdout |
| MCP HTTP transport | `yaah serve --http :7333` — Streamable HTTP |
| `prompt` tool | Run a multi-turn agent prompt; state persists |
| `traces` tool | Query in-process OpenTelemetry spans |
| `status` tool | Report provider, model, session, token usage |

**Gas Town integration path:**
- Replace the tmux-based mayor with an MCP client that connects to `yaah serve`.
- Use the `prompt` tool for all mayor interactions.
- Use the `traces` tool for dashboard observability (replaces Jaeger dependency).
- Use the `status` tool for health checks.

If MCP mode is not desired, yaah works fine in tmux mode with the REPL.

### 8. Working Directory and Config Paths

| Field | Requirement |
|---|---|
| `config_dir` | Top-level config directory: `$YAAH_HOME` or `~/.yaah` |
| `config_dir_env` | Env var to override config dir: `YAAH_HOME` |
| `hooks_dir` | JSONL hook directory, relative to `config_dir` or absolute |

yaah stores config at `~/.yaah/config.yaml` by default. The `YAAH_HOME` env
var overrides this. For the mayor, Gas Town should set `YAAH_HOME` to
`<townRoot>/mayor/.yaah` so each town gets its own yaah config.

Session state (SQLite DB) lives at `<config_dir>/memory.db`.

### 9. Readiness Detection

The mayor session must be "ready" before Gas Town sends it the startup nudge.
yaah must provide a detectable signal that it's initialized.

| Requirement | Details |
|---|---|
| Ready prompt prefix | REPL mode: `yaah ❯ ` (Bold "yaah" + Cyan "❯") |
| Ready delay | MCP mode: server responds to `initialize` handshake instantly |

**REPL prompt string:**
```
yaah ❯
```
Where `yaah` is ANSI bold and `❯` is ANSI cyan. Gas Town's session readiness
detector should look for the `❯` character or the `yaah` prefix.

**MCP mode readiness:**
The `yaah serve` command completes the MCP `initialize` handshake immediately,
so readiness is determined by successful JSON-RPC communication rather than
output pattern matching.

### 10. Sub-Agent / Delegation Awareness

yaah's dual-loop executor with `spawn_subagent` is a strength. For mayor
integration, yaah already supports:

- **Role-based dispatch**: Built-in roles (analyst, developer, tester, reviewer)
  plus custom roles from `.agents/roles/*.md`.
- **Sub-agent contracts**: Structured output contracts with evidence/interpretation fields.
- **Escalation blocks**: Sub-agents can emit structured escalation blocks
  (`blocker`, `critical`, `warning`, `info`) that the parent agent sees.
- **Concurrency control**: `max_concurrency` caps parallel sub-agent spawns.
- **Timeout control**: Per-role and global timeouts with `StuckChildTimeout`
  liveness guards.
- **OTel tracing**: Sub-agent spans are emitted as `subagent: <role>` spans
  with prompt/completion token attributes.

This maps well to Gas Town's crew model. The main gap is that yaah's sub-agent
tool is called `spawn_subagent` (not `task`), so Gas Town's crew dispatch
would need to map its `task` tool calls to yaah's `spawn_subagent`.

---

## Recommended Implementation Path

### Phase 1: tmux mode (minimum viable mayor)

Add a yaah preset to `internal/config/agents.go`:

```go
AgentYaah: {
    Name:          AgentYaah,
    Command:       "yaah",
    Args:          []string{},
    ProcessNames:  []string{"yaah"},
    SessionIDEnv:  "", // yaah does not export session ID via env
    ResumeFlag:    "--resume",
    ContinueFlag:  "", // not implemented
    ResumeStyle:   "flag",
    SupportsHooks: false, // Phase 1: use AGENTS.md fallback
    PromptMode:    "arg",  // "yaah \"prompt\"" for one-shot
    InstructionsFile: "AGENTS.md",
    ReadyDelayMs:  5000, // REPL prompt detection preferred
    Env: map[string]string{
        "YAHC_APPROVAL": "allow",
    },
},
```

Then test:
```bash
docker compose exec gastown gt mayor attach --agent yaah
```

For session resume, Gas Town will need to either:
- Parse the SQLite DB at `<townRoot>/mayor/.yaah/memory.db` to find the latest
  session ID, or
- Accept that the mayor starts fresh each time (no resume in Phase 1).

### Phase 2: MCP serve mode (tight integration)

Instead of tmux, run yaah as an MCP server:

```bash
yaah serve --http 127.0.0.1:7333
```

Gas Town connects as an MCP client and uses the `prompt` tool for all
interactions. This enables:
- Programmatic control without tmux.
- Structured `traces` tool for dashboard observability.
- Clean shutdown via MCP session teardown.

Update the preset:
```go
ACPMode: "native", // yaah IS the MCP server
ACP: &ACPConfig{
    Command: "serve",
    Args:    []string{"--http", "127.0.0.1:7333"},
},
```

Note: Gas Town's ACP proxy would need an MCP transport adapter, or yaah could
add a thin ACP shim on top of its MCP server.

### Phase 3: Active hooks (optional)

If yaah wants to support active hooks (Gas Town injecting behavior, not just
observing), it would need:
- A plugin directory that yaah loads at startup.
- Hook points: `SessionStart`, `UserPromptSubmit`, `Stop`.
- Each hook is an executable or script that yaah calls with JSON payload.

This is not required for basic mayor operation.

---

## Open Questions (Answered from Source)

1. **Binary name**: `yaah` — single static Go binary.
2. **Process detection**: Runs as a single `yaah` process. No Node.js/Bun
   shell-out.
3. **Session storage**: SQLite at `~/.yaah/memory.db`. Session IDs are
   `sess-<unix-nano>`.
4. **Resume semantics**: `yaah --resume <session-id>` resumes a specific
   session. No `--continue` flag.
5. **Permission model**: `approval: allow|ask|deny` in config, `YAAH_APPROVAL`
   env var, or `--approval` CLI flag.
6. **Hooks**: Passive JSONL event log only. No active hook execution.
   `AGENTS.md` is the primary startup context mechanism.
7. **ACP**: Not implemented. yaah uses MCP (`yaah serve`) instead, which is
   actually more suitable for programmatic integration.
