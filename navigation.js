// Embedded navigation JavaScript for stacked C4 diagrams
let currentLevel = 'context';
let fitToWidth = false; // false = native size (free zoom), true = auto-scale (constrained)

function resizeContainers() {
  // Get the actual browser viewport dimensions
  const viewBoxWidth = window.innerWidth;
  const viewBoxHeight = window.innerHeight;

  // Calculate available space in viewport coordinates (subtract header and navigation)
  const containerWidth = viewBoxWidth - 10; // Minimal padding
  const containerHeight = viewBoxHeight - 160; // header + nav + padding
  
  // Diagram dimensions are provided by the Go program
  // diagramData will be injected before this script
  
  // Update all container rectangles and diagrams
  const levels = ['context', 'container', 'component', 'code'];
  levels.forEach(level => {
    const container = document.getElementById('container-' + level);
    const diagram = document.getElementById('diagram-' + level);
    const data = diagramData[level];
    
    if (container && diagram && data) {
      // Algorithm: Prioritize readability by using available space more effectively
      const minScale = 0.4; // Allow smaller scale for very large diagrams
      const maxScale = 2.0;  // Allow more scaling up for small diagrams
      
      // Calculate scale to fit container with more aggressive space usage
      const scaleToFitWidth = containerWidth / data.width;
      const scaleToFitHeight = containerHeight / data.height;
      const scaleToFit = Math.min(scaleToFitWidth, scaleToFitHeight);
      
      // Choose scaling strategy based on fit mode
      let finalScale;
      if (fitToWidth) {
        // Auto-scale mode - fit to available viewport (constrained)
        if (scaleToFit < 0.8) {
          // Large diagram - use available space but ensure no overflow
          finalScale = Math.max(minScale, scaleToFit * 0.98); // Use 98% to prevent overflow
        } else {
          // Smaller diagram - scale up to use space efficiently
          finalScale = Math.min(maxScale, scaleToFit * 0.95); // Slight margin to prevent overflow
        }
      } else {
        // Native size mode - no scaling, show at original diagram size for free zoom
        finalScale = 1.0;
      }
      
      // Calculate final diagram size
      const finalWidth = data.width * finalScale;
      const finalHeight = data.height * finalScale;
      
      // Container sizing based on mode
      let containerFinalWidth, containerFinalHeight;
      
      if (fitToWidth) {
        // Auto-scale mode - container fits viewport (constrained)
        containerFinalWidth = containerWidth;
        containerFinalHeight = containerHeight;
        
        // For tall diagrams in auto-scale mode, extend height
        if (finalHeight > containerHeight) {
          containerFinalHeight = finalHeight + 20;
          const mainSVG = document.documentElement;
          const newSVGHeight = 160 + finalHeight + 40;
          mainSVG.style.minHeight = newSVGHeight + 'px';
        }
      } else {
        // Native mode - make container large enough for full diagram + some padding
        containerFinalWidth = Math.max(containerWidth, finalWidth + 40);
        containerFinalHeight = Math.max(containerHeight, finalHeight + 40);

        // Extend SVG canvas to accommodate large diagrams
        const mainSVG = document.documentElement;
        const newSVGHeight = 160 + finalHeight + 80; // header + diagram + padding
        const newSVGWidth = Math.max(viewBoxWidth, finalWidth + 80);
        mainSVG.style.minHeight = newSVGHeight + 'px';
        mainSVG.style.minWidth = newSVGWidth + 'px';
      }
      
      // Update container rectangle
      container.setAttribute('width', containerFinalWidth);
      container.setAttribute('height', containerFinalHeight);
      
      // Update diagram SVG - centered horizontally with minimal padding
      const diagramX = Math.max(0, (containerFinalWidth - finalWidth) / 2);
      const diagramY = 5; // Minimal top padding

      diagram.setAttribute('x', 5 + diagramX); // Minimal left offset
      diagram.setAttribute('y', 140 + diagramY);
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

function toggleFitMode() {
  fitToWidth = !fitToWidth;

  // Update button text
  const toggleText = document.getElementById('fit-text');

  if (fitToWidth) {
    toggleText.textContent = 'Auto Scale';
  } else {
    toggleText.textContent = 'Native Size';
  }

  // Reapply scaling with new mode
  resizeContainers();
}