# C4 Architecture Diagram Specification

Reusable specification for generating C4 architecture diagrams (3 levels) with optional code-level diagrams (4th level).

## PlantUML Configuration

All diagrams must include this base configuration:

```plantuml
!include <C4/C4_Context>
' or: !include <C4/C4_Container>
' or: !include <C4/C4_Component>

skinparam classAttributeIconSize 0
hide stereotype
```

**Rationale:**
- Uses PlantUML's **built-in C4 macros** (no remote dependencies)
- `hide stereotype` removes `<<person>>`, `<<system>>`, `<<container>>`, `<<component>>` labels
- `skinparam classAttributeIconSize 0` removes icon padding for compact layout

---

## Four Diagram Levels

### Level 1: System Context (01-system-context.puml)
**C4 Type:** System Context Diagram
**Scope:** External systems, actors, and high-level relationships
**PlantUML Include:** `!include <C4/C4_Context>`

**Key elements:**
- External actors (people, systems)
- Main system(s) under study
- External systems/dependencies
- High-level interactions

**Notes:**
- Should be understandable by non-technical stakeholders
- Include drill-down link: `$link="02-container-diagram.svg"`

---

### Level 2: Container Diagram (02-container-diagram.puml)
**C4 Type:** Container Diagram
**Scope:** Major containers (services, applications, databases) within system boundary
**PlantUML Include:** `!include <C4/C4_Container>`

**Key elements:**
- System boundaries (e.g., Kubernetes clusters, AWS regions)
- Containers/services (e.g., microservices, databases, caches)
- External systems
- Container-to-container communication

**Notes:**
- Show deployment boundaries
- Include drill-down links: `$link="01-system-context.svg"` and `$link="03-component-diagram.svg"`
- Add explanatory notes for complex relationships (bootstrapping, dependencies)

---

### Level 3: Component Diagram (03-component-diagram.puml)
**C4 Type:** Component Diagram
**Scope:** Detailed components within key containers
**PlantUML Include:** `!include <C4/C4_Component>`

**Key elements:**
- Components within important containers (e.g., API handlers, services, data access layers)
- Component responsibilities
- Inter-component interactions
- External dependencies

**Notes:**
- Detail sufficient to understand workflows and data flows
- Include drill-down links: `$link="02-container-diagram.svg"` and (optionally) `$link="04-code.svg"`
- Add workflow notes explaining complex interactions

---

### Level 4: Code Diagram (04-code.puml) [OPTIONAL]
**Type:** UML Class Diagram or Entity-Relationship Diagram
**Scope:** Actual code-level constructs (data models, schemas, API structures)
**PlantUML Include:** `!include <C4/C4_Component>` or standard UML

**When to generate:**
- **Only if possible/practical** - not all architectures have useful code diagrams
- Focus on significant data models or database schemas
- Skip if the system is already well-documented elsewhere

**Best candidates:**
- **UML Class Diagrams**: Data transfer objects, domain models, API request/response structures
- **ER Diagrams**: Database schemas, table relationships, key relationships
- Avoid: Implementation details, getters/setters, private methods

**Key elements:**
- Primary entities/classes
- Attributes (data types)
- Relationships (cardinality, foreign keys)
- Key constraints

**Notes:**
- Include drill-down link: `$link="03-component-diagram.svg"`
- Add notes explaining key relationships or constraints
- Focus on **what matters** for understanding the system, not exhaustive documentation

**Example scope:**
For a container registry system:
- Image model (name, tags, created_at, size)
- Repository model (name, owner, visibility)
- Relationship: Repository 1---* Image
- User model (username, role, credentials)
- Relationship: User 1---* Repository

---

## Drill-Down Navigation

Use PlantUML's `$link` parameter to create clickable drill-down links between diagram levels:

```plantuml
System(harbor, "Harbor Registry", "Full description", $link="02-container-diagram.svg")
Container(nginx, "nginx", "Reverse Proxy", "Description", $link="03-component-diagram.svg")
Component(auth, "Auth Manager", "Go", "Description", $link="04-code.svg")
```

**Convention:**
- Each diagram links forward to the next level
- Level 1 → Level 2, Level 2 → Level 3, Level 3 → Level 4 (if it exists)
- Consider back-links where helpful: Level 2 → Level 1, Level 3 → Level 2

---

## Explanatory Notes

Use PlantUML `note` blocks to explain complex interactions:

```plantuml
note right of system_name
  **Key Relationship:**
  Describe any non-obvious dependencies,
  bootstrap requirements, or circular
  relationships here.
end note
```

**When to add notes:**
- Bootstrap dependencies (e.g., "ECR must be deployed before Harbor")
- Circular relationships or complex data flows
- Alternative pathways or conditional logic
- Rate-limiting or caching strategies
- Authentication/credential flows

---

## File Naming Convention

```
01-system-context.puml
02-container-diagram.puml
03-component-diagram.puml
04-code.puml                 [OPTIONAL]
```

All files should be placed in the same directory (typically `docs/c4/`).

---

## Validation

After generating each `.puml` file, validate its syntax:

```bash
plantuml -checkonly -failfast2 <filename>.puml
```

If validation fails, fix the syntax errors in the generated file and re-validate until all files pass.

---

## Prompt Template for Claude

When requesting C4 diagrams from Claude, use this template:

```
Generate C4 architecture diagrams:

**System:** [System Name]

**Key Components/Systems:**
- [Component 1]
- [Component 2]
- [etc.]

**Important Relationships to Highlight:**
- [Relationship 1 - e.g., bootstrap dependency]
- [Relationship 2 - e.g., circular reference]

**Optional Code Diagram:**
[Yes/No] - If yes, focus on:
- [Data models to model]
- [Key relationships]

**Output:**
- 01-system-context.puml (System Context)
- 02-container-diagram.puml (Container)
- 03-component-diagram.puml (Component)
- [04-code.puml] (Optional)
```

