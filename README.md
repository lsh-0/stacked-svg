# Stacked C4 SVG

Generate self-contained, navigable C4 architecture diagrams from PlantUML with embedded JavaScript navigation.

## Features

- **Self-contained**: Single SVG file with all diagrams and navigation
- **No dependencies**: Works in any SVG-capable viewer or browser  
- **Responsive**: Adapts to viewport size with intelligent scaling
- **Native browser integration**: Supports scrolling and zoom
- **Clickable navigation**: Click diagram elements to drill down between C4 levels

## Quick Start

1. **Build the generator**:
   ```bash
   go build -o svg-stacker
   ```

2. **Add your PlantUML C4 diagrams** to `examples/` directory

3. **Generate individual SVGs** (using PlantUML):
   ```bash
   /home/user/bin/plantuml -tsvg -o output examples/*.puml
   ```

4. **Generate the stacked SVG**:
   ```bash
   ./svg-stacker
   ```

5. **View the result**: Open `output/stacked-c4.svg` in your browser

## Requirements

- Go compiler for building the generator
- PlantUML installed (configured in `/home/user/bin/plantuml`)
- PlantUML C4 library for diagram generation

## No Runtime Dependencies

The generator is a single Go binary with no external dependencies. The generated SVG file is completely self-contained with embedded JavaScript navigation.

## Project Structure

- `examples/` - PlantUML source files (.puml)
- `main.go` - Go generator source code
- `navigation.js` - JavaScript navigation logic (embedded into final SVG)
- `svg-stacker` - Compiled Go binary (gitignored)
- `output/` - Generated SVG files (individual diagrams + final stacked SVG)
- `CLAUDE.md` - Development guidance for Claude Code

## How It Works

1. Go program extracts content from PlantUML-generated SVG files (using regex, no dependencies)
2. Creates layered SVG structure with embedded JavaScript from `navigation.js`
3. Adds responsive navigation controls and scaling logic
4. Produces single self-contained file for easy sharing

## Example C4 Levels

- **Context**: System overview with external actors
- **Container**: Major application containers  
- **Component**: Components within containers
- **Code**: Implementation details

Generated from example banking system architecture.