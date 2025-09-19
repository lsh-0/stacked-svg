# Stacked C4 SVG

Generate self-contained, navigable C4 architecture diagrams from PlantUML with embedded JavaScript navigation.

## Features

- **Self-contained**: Single SVG file with all diagrams and navigation
- **No dependencies**: Works in any SVG-capable viewer or browser  
- **Responsive**: Adapts to viewport size with intelligent scaling
- **Native browser integration**: Supports scrolling and zoom
- **Clickable navigation**: Click diagram elements to drill down between C4 levels

## Quick Start

1. **Install dependencies**:
   ```bash
   npm install
   ```

2. **Add your PlantUML C4 diagrams** to `examples/` directory

3. **Generate the stacked SVG**:
   ```bash
   node src/svg-stacker.js
   ```

4. **View the result**: Open `output/stacked-c4.svg` in your browser

## Requirements

- PlantUML installed (configured in `/home/user/bin/plantuml`)
- Node.js with `cheerio` and `fs-extra` packages
- PlantUML C4 library for diagram generation

## Project Structure

- `examples/` - PlantUML source files (.puml)
- `src/svg-stacker.js` - Main generator script
- `output/` - Generated SVG files and demo viewers
- `CLAUDE.md` - Development guidance for Claude Code

## How It Works

1. Extracts content from PlantUML-generated SVG files
2. Creates layered SVG structure with embedded JavaScript
3. Adds responsive navigation controls and scaling logic
4. Produces single self-contained file for easy sharing

## Example C4 Levels

- **Context**: System overview with external actors
- **Container**: Major application containers  
- **Component**: Components within containers
- **Code**: Implementation details

Generated from example banking system architecture.