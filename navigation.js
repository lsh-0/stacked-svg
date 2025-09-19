// Embedded navigation JavaScript for stacked C4 diagrams
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