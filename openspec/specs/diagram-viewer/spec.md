# Diagram Viewer

## Purpose

Defines the runtime behaviour of the script embedded in the generated document: the features a reader can use in a browser and the guarantees about how the script prepares each layer.

## Requirements

### Requirement: Level switching
The viewer SHALL show one layer at a time, starting with the first level present. Clicking a navigation button SHALL show that level's layer and hide all others. The active button SHALL be visually distinguished. Clicking an element the converter marked with `navigateDown()` SHALL show the next level present, and SHALL do nothing on the last level.

#### Scenario: Initial state
- **WHEN** the document is opened
- **THEN** only the first level's layer is visible and its button is marked active

#### Scenario: Drill-down
- **WHEN** the reader clicks a marked element on the context layer
- **THEN** the container layer becomes visible and the context layer is hidden

#### Scenario: Drill-down on last level
- **WHEN** the reader clicks a marked element on the last level present
- **THEN** the visible layer does not change

### Requirement: Sizing modes
The viewer SHALL offer two sizing modes toggled by one button. In native mode the document SHALL expand to the current diagram's recorded width and height and rely on browser scrolling. In auto-scale mode the document SHALL fit the browser viewport and the diagram SHALL scale to the available space. The mode SHALL persist across level switches and the document SHALL re-lay out on browser resize.

#### Scenario: Toggle to auto-scale
- **WHEN** the reader clicks the sizing button while in native mode
- **THEN** the button reads "Auto Scale" and the diagram fits within the viewport

#### Scenario: Mode persists across levels
- **WHEN** the reader is in auto-scale mode and switches level
- **THEN** the new level is shown in auto-scale mode

### Requirement: Note toggling
The viewer SHALL offer a button that hides and shows all elements the converter tagged with class `note` and `note-link`, on every layer at once. The button text SHALL reflect the action it will perform next.

#### Scenario: Hide notes
- **WHEN** the reader clicks "Hide Notes"
- **THEN** every `note` and `note-link` element on every layer is hidden and the button reads "Show Notes"

### Requirement: Path highlighting
Hovering a link's label SHALL bring that link to the front, highlight its path, and show a background behind the label. Leaving the label SHALL remove the highlight unless the link is pinned. No browser tooltip SHALL appear over links.

#### Scenario: Hover
- **WHEN** the pointer enters a link label
- **THEN** the link gains the highlighted state and its label background becomes visible

#### Scenario: Leave unpinned
- **WHEN** the pointer leaves the label of a link that is not pinned
- **THEN** the highlighted state is removed

### Requirement: Pinned selection
Clicking a link label SHALL pin that link's highlight and unpin any other. Clicking a pinned link's label SHALL unpin it. Clicking with Ctrl (or Cmd on macOS) SHALL toggle the link in a multi-selection without affecting others. Pressing Escape SHALL unpin every link.

#### Scenario: Single pin replaces previous
- **WHEN** link A is pinned and the reader clicks link B's label without a modifier
- **THEN** B is pinned and A is not

#### Scenario: Multi-select
- **WHEN** link A is pinned and the reader Ctrl-clicks link B's label
- **THEN** both A and B are pinned

#### Scenario: Escape clears
- **WHEN** links are pinned and the reader presses Escape
- **THEN** no link is pinned or highlighted

### Requirement: Enhancement happens once per layer
The viewer SHALL add label hitboxes and backgrounds to a layer exactly once, when that layer is first shown. Switching to a level already shown SHALL not add elements or listeners. The Escape handler SHALL be registered exactly once for the document.

#### Scenario: Repeated switching
- **WHEN** the reader switches between two levels ten times
- **THEN** each link on each layer has exactly one hitbox and one background rectangle

#### Scenario: Single Escape handler
- **WHEN** the document has been loaded and any number of levels shown
- **THEN** exactly one keydown listener for Escape exists on the document
