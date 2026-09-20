# TODO

A structured capture of deferred work, ideas, and nice-to-haves.
Each item is separated by `---` and has key-value metadata followed
by optional free-text context.

---
title: Replace the Claude integration with a skill
added: 2026-09-20
effort: medium
tags: cli, prompt, claude
summary: Move C4 diagram generation out of the `prompt` subcommand and into a Claude Code skill

The `prompt` subcommand currently locates the Claude CLI and invokes it
directly. A skill would keep the binary free of that dependency.
---
