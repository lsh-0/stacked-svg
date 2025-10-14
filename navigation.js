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

  const links = currentLayer.querySelectorAll('g.link');

  links.forEach(link => {
    // Add mouseenter handler
    link.addEventListener('mouseenter', function() {
      // Bring this link to front by moving it to end of parent
      const parent = link.parentNode;
      parent.appendChild(link);

      // Dim all other links
      links.forEach(otherLink => {
        if (otherLink !== link) {
          otherLink.classList.add('dimmed');
        }
      });
    });

    // Add mouseleave handler
    link.addEventListener('mouseleave', function() {
      // Remove dimming from all links
      links.forEach(otherLink => {
        otherLink.classList.remove('dimmed');
      });
    });
  });
}
