// Embedded navigation JavaScript for stacked C4 diagrams
let currentLevel = 'context';
let fitToWidth = false; // false = native size (free zoom), true = auto-scale (constrained)
let notesVisible = true; // true = show notes, false = hide notes
let selectedLinks = []; // Track multiple selected links (Ctrl+click to multi-select)
const enhancedLevels = new Set(); // layers whose link labels already have hitboxes; enhancement needs layout, so it runs on first show

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

  // Link labels get their hitboxes the first time the layer is visible (getBBox needs layout)
  if (!enhancedLevels.has(level)) {
    enhancedLevels.add(level);
    setupLinkHoverEnhancements();
  }
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

  // The converter tags note groups with `note` and the links attached to them with `note-link`
  document.querySelectorAll('g.note, g.note-link').forEach(el => {
    el.style.display = notesVisible ? '' : 'none';
  });
}

// Drop every pinned link back to its normal state
function clearSelection() {
  selectedLinks.forEach(prevLink => {
    const prevBgRect = prevLink.querySelector('.text-bg');
    if (prevBgRect) {
      prevBgRect.setAttribute('fill-opacity', '0');
    }
    prevLink.classList.remove('highlighted');
  });
  selectedLinks = [];
}

// Escape clears the pinned selection; registered once for the document
document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape' && selectedLinks.length > 0) {
    clearSelection();
  }
});

// Progressive enhancement: JavaScript-enhanced link hovering
function setupLinkHoverEnhancements() {
  const currentLayer = document.getElementById('layer-' + currentLevel);
  if (!currentLayer) return;

  const links = currentLayer.querySelectorAll('g.link');

  links.forEach(link => {
    // Find text elements within this link (the labels)
    const textElements = link.querySelectorAll('text');

    // Let the hitbox underneath handle all mouse events
    textElements.forEach(textEl => {
      textEl.style.pointerEvents = 'none';
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

      // Add hover handler to hitbox (only if not selected)
      hitbox.addEventListener('mouseenter', function(e) {
        e.preventDefault();

        // Only highlight on hover if this link is not selected
        if (!selectedLinks.includes(link)) {
          // Bring this link to front
          const parent = link.parentNode;
          parent.appendChild(link);

          // Add highlight class
          link.classList.add('highlighted');

          // Show background
          bgRect.setAttribute('fill-opacity', '0.9');
        }
      });

      hitbox.addEventListener('mouseleave', function() {
        // Only remove highlight if this link is not selected
        if (!selectedLinks.includes(link)) {
          link.classList.remove('highlighted');
          bgRect.setAttribute('fill-opacity', '0');
        }
      });

      // Add click handler for persistent selection (Ctrl+click for multi-select)
      hitbox.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();

        const isMultiSelect = e.ctrlKey || e.metaKey; // Ctrl on Windows/Linux, Cmd on Mac

        if (isMultiSelect) {
          // Toggle this link in/out of selection
          const index = selectedLinks.indexOf(link);
          if (index > -1) {
            // Deselect
            selectedLinks.splice(index, 1);
            link.classList.remove('highlighted');
            bgRect.setAttribute('fill-opacity', '0');
          } else {
            // Add to selection
            selectedLinks.push(link);
            const parent = link.parentNode;
            parent.appendChild(link);
            link.classList.add('highlighted');
            bgRect.setAttribute('fill-opacity', '0.9');
          }
        } else {
          // Single select - deselect all others
          const wasOnlySelection = selectedLinks.length === 1 && selectedLinks[0] === link;
          clearSelection();

          // If clicking the same link again, leave it deselected
          if (wasOnlySelection) {
            // nothing more to do
          } else {
            // Select new link
            selectedLinks = [link];
            const parent = link.parentNode;
            parent.appendChild(link);
            link.classList.add('highlighted');
            bgRect.setAttribute('fill-opacity', '0.9');
          }
        }
      });
    }
  });
}
