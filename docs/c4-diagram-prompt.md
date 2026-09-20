# C4 diagram generation prompt

The prompt that the removed `prompt` subcommand assembled and piped into Claude Code. It is kept here so the same instructions can be given to Claude Code by hand, or turned into a skill, without the binary depending on the Claude CLI.

## Prompt template

Everything between the fences is sent as one message. `{{SPEC}}` is the full contents of [`../C4-DIAGRAM-SPEC.md`](../C4-DIAGRAM-SPEC.md), inserted verbatim. The other placeholders are described below.

````text
Generate C4 architecture diagrams for this project.

IMPORTANT: Save all generated .puml files to: docs/c4/

{{SPEC}}

---

PROJECT CONTEXT
===============

Project Name: {{PROJECT_NAME}}

README.md (first 10 lines):
{{README_HEAD}}

Primary file types: {{FILE_TYPES}}
Key files/directories: {{KEY_FILES}}

---

Please analyze this project and generate appropriate C4 diagrams following the spec above.
Generate files: 01-context.puml, 02-container.puml, 03-component.puml, and optionally 04-code.puml
All files should be saved to: docs/c4/
````

## Placeholders

The subcommand derived each value from the current working directory.

| Placeholder | Derivation |
|---|---|
| `{{PROJECT_NAME}}` | Base name of the current directory. `project` when that is empty or `.`. |
| `{{README_HEAD}}` | The first 10 lines of `README.md`, or the whole file when it is shorter. The `README.md (first 10 lines):` line and this block were omitted entirely when no `README.md` existed. |
| `{{FILE_TYPES}}` | The three most frequent file extensions among non-hidden files in the current directory only, not recursed, most frequent first, comma separated. The line was omitted when no extensions were found. |
| `{{KEY_FILES}}` | Whichever of `go.mod`, `package.json`, `Dockerfile`, `Makefile`, `src/`, `cmd/`, `main.go` exist, in that order, comma separated. The line was omitted when none existed. |

## Invocation

The prompt was passed on standard input to:

```sh
claude --print --input-format text --output-format stream-json --include-partial-messages --verbose
```

The `--print` and streaming flags avoided a raw-mode TTY problem and showed progress as Claude worked.

## Follow-up

Once Claude has written the `.puml` files, stack them:

```sh
./svg-stacker ./docs/c4/ --output ./docs/c4/stacked-c4-architecture.svg --title "🏗️ {{PROJECT_NAME}} Architecture"
```

The subcommand ran this automatically after Claude exited, and printed the command for later use.
