#!/usr/bin/env node

const fs = require('fs-extra');
const cheerio = require('cheerio');

class SVGStacker {
    constructor() {
        this.diagrams = new Map();
        this.svgWidth = 800;
        this.svgHeight = 600;
    }

    async createStackedSVG() {
        console.log('📚 Creating self-contained stacked SVG...');
        
        // Load all SVG files
        await this.loadDiagrams();
        
        // Create the master SVG with layers
        const stackedSVG = this.buildStackedSVG();
        
        // Write the result
        await fs.writeFile('output/stacked-c4.svg', stackedSVG);
        console.log('✅ Created output/stacked-c4.svg');
        console.log('📖 Open this file directly in any SVG viewer or browser');
    }

    async loadDiagrams() {
        const svgFiles = [
            'output/01-context.svg',
            'output/02-container.svg', 
            'output/03-component.svg',
            'output/04-code.svg'
        ];

        for (const file of svgFiles) {
            if (await fs.pathExists(file)) {
                const content = await fs.readFile(file, 'utf8');
                const $ = cheerio.load(content, { xmlMode: true });
                
                const level = this.extractLevel(file);
                const $svg = $('svg');
                
                // Parse dimensions
                const viewBox = $svg.attr('viewBox') || '0 0 400 300';
                const width = parseFloat($svg.attr('width')) || 400;
                const height = parseFloat($svg.attr('height')) || 300;
                
                // Calculate aspect ratio and container fit
                const aspectRatio = width / height;
                
                this.diagrams.set(level, {
                    content: $svg.html(),
                    viewBox: viewBox,
                    width: width,
                    height: height,
                    aspectRatio: aspectRatio
                });
                
                console.log(`  📄 Loaded ${level} level (${width}x${height}, ratio: ${aspectRatio.toFixed(2)})`);
            }
        }
    }

    extractLevel(filename) {
        if (filename.includes('01-context')) return 'context';
        if (filename.includes('02-container')) return 'container';
        if (filename.includes('03-component')) return 'component';
        if (filename.includes('04-code')) return 'code';
        return 'unknown';
    }

    buildStackedSVG() {
        const levels = ['context', 'container', 'component', 'code'];
        
        return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" 
     xmlns:xlink="http://www.w3.org/1999/xlink"
     width="100%" 
     height="100%" 
     viewBox="0 0 ${this.svgWidth} ${this.svgHeight}"
     preserveAspectRatio="xMidYMid meet"
     style="background: #f8f9fa; display: block; min-height: 100vh;">
     
  <title>Stacked C4 Architecture Diagrams</title>
  
  <!-- Navigation Header -->
  <rect x="0" y="0" width="${this.svgWidth}" height="60" fill="#2c3e50"/>
  <text x="20" y="25" font-family="Arial, sans-serif" font-size="16" font-weight="bold" fill="white">
    🏗️ Stacked C4 Architecture
  </text>
  <text x="20" y="45" font-family="Arial, sans-serif" font-size="12" fill="#ecf0f1" id="breadcrumb">
    Context Level
  </text>
  
  <!-- Navigation Buttons -->
  ${levels.map((level, index) => `
  <rect x="${20 + index * 90}" y="70" width="80" height="25" rx="3" 
        fill="#3498db" stroke="#2980b9" stroke-width="1" 
        style="cursor:pointer" onclick="showLevel('${level}')" 
        id="nav-${level}"/>
  <text x="${30 + index * 90}" y="86" font-family="Arial, sans-serif" font-size="11" 
        fill="white" style="cursor:pointer; user-select: none" 
        onclick="showLevel('${level}')">
    ${level.charAt(0).toUpperCase() + level.slice(1)}
  </text>
  `).join('')}
  
  <!-- Diagram Layers -->
  ${levels.map(level => this.createDiagramLayer(level)).join('')}
  
  <!-- Navigation Script -->
  <script type="text/ecmascript"><![CDATA[
    let currentLevel = 'context';
    
    function resizeContainers() {
      // Get the viewport dimensions from the SVG's viewBox
      const viewBoxWidth = 800;
      const viewBoxHeight = 600;
      
      // Calculate available space in viewBox coordinates (subtract header and navigation)
      const containerWidth = viewBoxWidth - 40; // 20px padding each side
      const containerHeight = viewBoxHeight - 130; // header + nav + padding
      
      // Diagram dimensions and scaling strategy
      const diagramData = {
        'context': { width: 486, height: 550, ratio: 0.88 },
        'container': { width: 872, height: 820, ratio: 1.06 },
        'component': { width: 652, height: 918, ratio: 0.71 },
        'code': { width: 934, height: 670, ratio: 1.39 }
      };
      
      // Update all container rectangles and diagrams
      const levels = ['context', 'container', 'component', 'code'];
      levels.forEach(level => {
        const container = document.getElementById('container-' + level);
        const diagram = document.getElementById('diagram-' + level);
        const data = diagramData[level];
        
        if (container && diagram && data) {
          // Algorithm: Scale to fit container, but maintain minimum readable size
          const minScale = 0.7; // Don't scale below 70% of original
          const maxScale = 1.5;  // Don't scale above 150% of original
          
          // Calculate scale to fit container
          const scaleToFitWidth = containerWidth / data.width;
          const scaleToFitHeight = containerHeight / data.height;
          const scaleToFit = Math.min(scaleToFitWidth, scaleToFitHeight);
          
          // Apply scale constraints
          const finalScale = Math.max(minScale, Math.min(maxScale, scaleToFit));
          
          // Calculate final diagram size
          const finalWidth = data.width * finalScale;
          const finalHeight = data.height * finalScale;
          
          // For tall diagrams, expand the SVG canvas to allow native browser scrolling
          if (finalHeight > containerHeight) {
            // Extend the main SVG height to accommodate tall diagrams
            const mainSVG = document.documentElement;
            const newSVGHeight = 130 + finalHeight + 40; // header + diagram + padding
            mainSVG.setAttribute('viewBox', '0 0 ' + viewBoxWidth + ' ' + newSVGHeight);
          }
          
          // Container size - fill available space or fit diagram
          const containerFinalWidth = containerWidth;
          const containerFinalHeight = Math.max(containerHeight, finalHeight + 20);
          
          // Update container rectangle
          container.setAttribute('width', containerFinalWidth);
          container.setAttribute('height', containerFinalHeight);
          
          // Update diagram SVG - centered horizontally, top-aligned for tall diagrams
          const diagramX = Math.max(0, (containerFinalWidth - finalWidth) / 2);
          const diagramY = 10; // Small top padding
          
          diagram.setAttribute('x', 20 + diagramX);
          diagram.setAttribute('y', 110 + diagramY);
          diagram.setAttribute('width', finalWidth);
          diagram.setAttribute('height', finalHeight);
        }
      });
    }
    
    function showLevel(level) {
      // Hide all layers
      const layers = ['context', 'container', 'component', 'code'];
      layers.forEach(l => {
        const layer = document.getElementById('layer-' + l);
        if (layer) {
          layer.style.display = 'none';
        }
        
        // Update button styles
        const btn = document.getElementById('nav-' + l);
        if (btn) {
          btn.setAttribute('fill', l === level ? '#e74c3c' : '#3498db');
        }
      });
      
      // Show selected layer
      const targetLayer = document.getElementById('layer-' + level);
      if (targetLayer) {
        targetLayer.style.display = 'block';
      }
      
      // Update breadcrumb
      const breadcrumb = document.getElementById('breadcrumb');
      if (breadcrumb) {
        breadcrumb.textContent = level.charAt(0).toUpperCase() + level.slice(1) + ' Level';
      }
      
      currentLevel = level;
      
      // Resize containers after showing layer
      setTimeout(resizeContainers, 10);
    }
    
    // Initialize - show context level and setup resize
    showLevel('context');
    resizeContainers();
    
    // Resize on window resize
    window.addEventListener('resize', resizeContainers);
    
    // Add click handlers for diagram elements to navigate between levels
    function navigateDown() {
      const levelOrder = ['context', 'container', 'component', 'code'];
      const currentIndex = levelOrder.indexOf(currentLevel);
      if (currentIndex < levelOrder.length - 1) {
        showLevel(levelOrder[currentIndex + 1]);
      }
    }
    
    function navigateUp() {
      const levelOrder = ['context', 'container', 'component', 'code'];
      const currentIndex = levelOrder.indexOf(currentLevel);
      if (currentIndex > 0) {
        showLevel(levelOrder[currentIndex - 1]);
      }
    }
  ]]></script>
  
  <!-- Instructions -->
  <text x="450" y="86" font-family="Arial, sans-serif" font-size="10" fill="#7f8c8d">
    Click buttons or diagram elements to navigate • Self-contained SVG
  </text>
  
</svg>`;
    }

    createDiagramLayer(level) {
        const diagram = this.diagrams.get(level);
        if (!diagram) {
            return `
  <!-- ${level} layer (not found) -->
  <g id="layer-${level}" style="display:none">
    <rect x="50" y="120" width="700" height="450" fill="#ecf0f1" stroke="#bdc3c7"/>
    <text x="400" y="350" text-anchor="middle" font-family="Arial" font-size="16" fill="#7f8c8d">
      ${level.charAt(0).toUpperCase() + level.slice(1)} diagram not found
    </text>
  </g>`;
        }

        // Clean and embed the diagram content
        const cleanContent = this.cleanDiagramContent(diagram.content, level);
        
        return `
  <!-- ${level} layer -->
  <g id="layer-${level}" style="display:none">
    <rect x="20" y="110" width="760" height="470" fill="white" stroke="#ddd" stroke-width="1" rx="5" id="container-${level}"/>
    <svg x="30" y="120" width="740" height="450" viewBox="${diagram.viewBox}" preserveAspectRatio="xMidYMid meet" id="diagram-${level}">
      ${cleanContent}
    </svg>
  </g>`;
    }

    cleanDiagramContent(content, level) {
        const $ = cheerio.load(content, { xmlMode: true });
        
        // Remove scripts and convert links to click handlers
        $('script').remove();
        
        // Convert <a> tags to click handlers
        $('a').each((i, elem) => {
            const $elem = $(elem);
            const $parent = $elem.parent();
            
            // Move the click handler to the parent element and remove the <a> tag
            $parent.attr('onclick', 'navigateDown()');
            $parent.attr('style', ($parent.attr('style') || '') + ' cursor:pointer;');
            
            // Replace the <a> tag with its contents
            $elem.replaceWith($elem.html());
        });
        
        return $.html();
    }
}

// Run if called directly
if (require.main === module) {
    const stacker = new SVGStacker();
    stacker.createStackedSVG().catch(console.error);
}

module.exports = SVGStacker;