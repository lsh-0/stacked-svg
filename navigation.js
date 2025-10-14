// Embedded navigation JavaScript for stacked C4 diagrams
let currentLevel = 'context';
let fitToWidth = false; // false = native size (free zoom), true = auto-scale (constrained)
let notesVisible = true; // true = show notes, false = hide notes

function positionRightAlignedElements() {
  const viewBoxWidth = window.innerWidth;
  const fitToggleButton = document.getElementById('fit-toggle');
  const fitToggleText = document.getElementById('fit-text');
  const notesToggleButton = document.getElementById('notes-toggle');
  const notesToggleText = document.getElementById('notes-text');

  if (fitToggleButton && fitToggleText && notesToggleButton && notesToggleText) {
    // Position fit toggle on far right
    const fitToggleX = viewBoxWidth - 156; // 130px width + 26px margin
    const fitTextX = fitToggleX + 13;

    // Position notes toggle to the left of fit toggle
    const notesToggleX = fitToggleX - 143; // 130px button width + 13px gap
    const notesTextX = notesToggleX + 13;

    fitToggleButton.setAttribute('x', fitToggleX);
    fitToggleText.setAttribute('x', fitTextX);
    notesToggleButton.setAttribute('x', notesToggleX);
    notesToggleText.setAttribute('x', notesTextX);
  }
}

function resizeContainers() {
  // Get the actual browser viewport dimensions
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  // Get the outer SVG
  const mainSVG = document.documentElement;
  if (!mainSVG) return;

  // Get current level diagram data
  const data = diagramData[currentLevel];
  if (!data) return;

  // Get container and diagram elements
  const container = document.getElementById('container-' + currentLevel);
  const diagramGroup = document.getElementById('diagram-' + currentLevel);
  const diagramSVG = diagramGroup ? diagramGroup.querySelector('svg') : null;

  if (!container || !diagramSVG) return;

  if (fitToWidth) {
    // Auto-scale mode - SVG fits viewport, diagram scales to fit available space
    mainSVG.setAttribute('width', viewportWidth);
    mainSVG.setAttribute('height', viewportHeight);

    const availableWidth = viewportWidth - 20;
    const availableHeight = viewportHeight - 160;

    container.setAttribute('width', availableWidth);
    container.setAttribute('height', availableHeight);

    diagramSVG.setAttribute('width', availableWidth - 10);
    diagramSVG.setAttribute('height', availableHeight - 10);
  } else {
    // Native size mode - SVG expands to diagram's native size, browser provides scrollbars
    const svgWidth = Math.max(viewportWidth, data.width + 30);
    const svgHeight = Math.max(viewportHeight, 140 + data.height + 30);

    mainSVG.setAttribute('width', svgWidth);
    mainSVG.setAttribute('height', svgHeight);

    container.setAttribute('width', data.width + 20);
    container.setAttribute('height', data.height + 20);

    diagramSVG.setAttribute('width', data.width);
    diagramSVG.setAttribute('height', data.height);
  }
}

function showLevel(level) {
  // Hide all layers
  availableLevels.forEach(l => {
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

  currentLevel = level;

  // Resize containers after showing layer
  setTimeout(resizeContainers, 10);

  // Setup link hover enhancements (JavaScript progressive enhancement)
  setupLinkHoverEnhancements();
}

// Initialize - show context level and setup resize
showLevel('context');
positionRightAlignedElements();
resizeContainers();

// Resize on window resize
window.addEventListener('resize', function() {
  positionRightAlignedElements();
  resizeContainers();
});

// Add click handlers for diagram elements to navigate between levels
function navigateDown() {
  const currentIndex = availableLevels.indexOf(currentLevel);
  if (currentIndex < availableLevels.length - 1) {
    showLevel(availableLevels[currentIndex + 1]);
  }
}

function navigateUp() {
  const currentIndex = availableLevels.indexOf(currentLevel);
  if (currentIndex > 0) {
    showLevel(availableLevels[currentIndex - 1]);
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

function toggleNotes() {
  notesVisible = !notesVisible;

  // Update button text
  const notesText = document.getElementById('notes-text');
  notesText.textContent = notesVisible ? 'Hide Notes' : 'Show Notes';

  // Find all note elements (PlantUML generates notes with the distinctive yellow fill)
  // Notes are <g> elements with class="entity" containing paths with fill="#FEFFDD"
  availableLevels.forEach(level => {
    const layer = document.getElementById('layer-' + level);
    if (layer) {
      const noteElements = layer.querySelectorAll('g.entity');
      noteElements.forEach(g => {
        // Check if this group contains a note (yellow fill path)
        const notePath = g.querySelector('path[fill="#FEFFDD"]');
        if (notePath) {
          g.style.display = notesVisible ? 'block' : 'none';
        }
      });
    }
  });
}

// Progressive enhancement: JavaScript-enhanced link hovering
function setupLinkHoverEnhancements() {
  const currentLayer = document.getElementById('layer-' + currentLevel);
  if (!currentLayer) return;

  // Remove all <title> elements from the diagram to prevent tooltips
  const allTitles = currentLayer.querySelectorAll('title');
  allTitles.forEach(t => t.remove());

  const links = currentLayer.querySelectorAll('g.link');

  links.forEach(link => {
    // Remove any title elements that might trigger tooltips
    const titleElements = link.querySelectorAll('title');
    titleElements.forEach(t => t.remove());

    // Find text elements within this link (the labels)
    const textElements = link.querySelectorAll('text');

    // Remove title attributes from all text elements and make them non-interactive
    textElements.forEach(textEl => {
      textEl.removeAttribute('title');
      textEl.style.pointerEvents = 'none'; // Let hitbox underneath handle all mouse events
      let parent = textEl.parentElement;
      while (parent && parent !== link) {
        parent.removeAttribute('title');
        parent = parent.parentElement;
      }
    });

    // Create persistent background rectangle for the label area
    if (textElements.length > 0) {
      // Calculate bounding box of all text elements together
      let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
      textElements.forEach(t => {
        const bbox = t.getBBox();
        minX = Math.min(minX, bbox.x);
        minY = Math.min(minY, bbox.y);
        maxX = Math.max(maxX, bbox.x + bbox.width);
        maxY = Math.max(maxY, bbox.y + bbox.height);
      });

      // Create invisible hitbox rectangle that's always present
      const hitbox = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      hitbox.setAttribute('x', minX - 3);
      hitbox.setAttribute('y', minY - 2);
      hitbox.setAttribute('width', (maxX - minX) + 6);
      hitbox.setAttribute('height', (maxY - minY) + 4);
      hitbox.setAttribute('fill', 'transparent');
      hitbox.setAttribute('rx', '3');
      hitbox.style.cursor = 'pointer';
      hitbox.classList.add('label-hitbox');

      // Insert before first text element
      textElements[0].parentNode.insertBefore(hitbox, textElements[0]);

      // Create visible background (hidden by default)
      const bgRect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
      bgRect.setAttribute('x', minX - 3);
      bgRect.setAttribute('y', minY - 2);
      bgRect.setAttribute('width', (maxX - minX) + 6);
      bgRect.setAttribute('height', (maxY - minY) + 4);
      bgRect.setAttribute('fill', '#e74c3c');
      bgRect.setAttribute('fill-opacity', '0');
      bgRect.setAttribute('rx', '3');
      bgRect.classList.add('text-bg');
      bgRect.style.pointerEvents = 'none'; // Don't interfere with hitbox

      // Insert before first text element
      textElements[0].parentNode.insertBefore(bgRect, textElements[0]);

      // Add hover handler to hitbox
      hitbox.addEventListener('mouseenter', function(e) {
        e.preventDefault();

        // Bring this link to front
        const parent = link.parentNode;
        parent.appendChild(link);

        // Add highlight class
        link.classList.add('highlighted');

        // Show background
        bgRect.setAttribute('fill-opacity', '0.9');
      });

      hitbox.addEventListener('mouseleave', function() {
        link.classList.remove('highlighted');
        bgRect.setAttribute('fill-opacity', '0');
      });
    }
  });
}
