# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This project creates navigable C4 (Context, Containers, Components, Code) architecture diagrams using PlantUML with SVG output and built-in linking capabilities.

## Commands

- **Generate diagrams**: `/home/user/bin/plantuml -tsvg examples/*.puml`
- **View demo**: Open `output/index.html` in a browser
- **Test PlantUML**: `/home/user/bin/plantuml -version`

## Project Structure

- `examples/`: PlantUML source files (.puml) with C4 diagrams
- `output/`: Generated SVG files and HTML viewer
- `test.puml`: Simple test file for PlantUML verification

## PlantUML C4 Navigation

Uses PlantUML's built-in `$link` parameter for clickable navigation:
```puml
System(banking_system, "Internet Banking System", "Main banking system", $link="02-container.svg")
```

## Current Implementation

- Context diagram links to container diagram
- Container diagram links to component diagram  
- Component diagram links to code diagram
- SVG format required for clickable links (PNG doesn't support links)
- Links work when SVGs are opened directly in browsers

## Self-Contained Stacked SVG Solution

- **Generate stacked SVG**: `node src/svg-stacker.js`
- **View result**: Open `output/stacked-c4.svg` directly in any browser or SVG viewer
- **Features**: 
  - Single self-contained file
  - No web server required
  - JavaScript-based layer switching
  - Built-in navigation controls
  - Works in any SVG-capable viewer

## Limitations of Built-in PlantUML Navigation

- Requires separate SVG files for each diagram
- No seamless drill-down in single interface  
- Manual link management between diagram levels
- Limited to simple file-based navigation

## Dependencies

- Node.js with `cheerio` and `fs-extra` for SVG processing