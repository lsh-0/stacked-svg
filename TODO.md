# TODO

A structured capture of deferred work, ideas, and nice-to-haves.
Each item is separated by `---` and has key-value metadata followed
by optional free-text context.

---
title: Build a Claude Code skill from the C4 diagram prompt document
added: 2026-09-20
updated: 2026-09-20
effort: low
tags: docs, claude
summary: Turn `docs/c4-diagram-prompt.md` into an invocable skill that writes the `.puml` files and runs the stacker

The `prompt` subcommand has been removed from the binary and its prompt
captured in the document. What remains is packaging that document as a
skill so it no longer has to be pasted by hand.
---
