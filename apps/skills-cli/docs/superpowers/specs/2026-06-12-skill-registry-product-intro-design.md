# Skill Registry Product Intro Deck Design

Date: 2026-06-12
Topic: Internal product-team introduction deck for `skill registry`
Format: Slidev deck
Target duration: 10 minutes
Audience: Internal product team
Primary outcome: Unified product understanding

## 1. Objective

Produce a concise Slidev presentation that helps the internal product team align on what `skill registry` is, why it exists, what problem boundary it owns, and how to describe it consistently.

The deck is not a technical architecture review, an implementation demo, or a roadmap deep dive. It should establish shared product language first.

## 2. Core Product Definition

Working one-line definition:

> `Skill Registry` is infrastructure for producing, publishing, discovering, installing, updating, and governing AI agent skills in a standardized way.

Compressed message for repetition:

> Let AI agent skills be managed like software packages.

## 3. Audience and Success Criteria

### Audience

- Internal product team members
- People who need product-level understanding, not code-level detail

### Success criteria

After the presentation, the audience should retain three conclusions:

1. `skill registry` solves standardized skill delivery, not just one-off installation.
2. The product covers the full skill lifecycle from authoring and publishing to discovery, installation, updating, and governance.
3. The product boundary is infrastructure for the skill ecosystem, not model hosting, agent runtime orchestration, or generic knowledge management.

## 4. Recommended Narrative

Use a product-first narrative:

1. Why the problem exists
2. What the product is
3. Who it serves
4. What problems it solves
5. What capabilities it includes
6. What form the product takes
7. What it is not
8. What the audience should remember

This is preferred over a workflow-first or architecture-first story because the immediate goal is shared understanding inside the product team.

## 5. Deck Structure

The deck should contain 9 slides.

### Slide 1. Cover

- Title: `Skill Registry 产品介绍`
- Subtitle: `面向 AI Agent 技能的标准化生产与分发基础设施`

### Slide 2. Why It Exists

Primary message:

> The core problem is not whether a skill can be written. The problem is how a skill is delivered, reused, and governed reliably after it exists.

Supporting points:

- Skills are scattered across repos, docs, and personal environments.
- Delivery methods are inconsistent.
- Versions and ownership are hard to track.
- Team reuse is weak.
- Enterprise and internal-network delivery adds operational complexity.

### Slide 3. What It Is

Primary message:

> `skill registry` is not a single tool. It is a complete product chain around skills.

Supporting points:

- Standardized skill definition and validation
- Publishing and release recording
- Search and discovery
- Installation and update flows
- Consistent rules and entry points

### Slide 4. Who It Serves

Primary message:

> The product connects skill authors, platform operators, and skill consumers.

Supporting points:

- Authors create and package skills
- Platform or operations roles govern publishing and distribution
- Internal teams discover, install, update, and use skills

### Slide 5. Core Problems Solved

Primary message:

> The product upgrades "people can write skills" into "the organization can manage and distribute skills at scale."

Supporting points:

- Standardized publishing
- Searchable and installable distribution
- Sustainable updating and governance

### Slide 6. Capability Landscape

Primary message:

> The product supports the full lifecycle of a skill.

Lifecycle sequence:

- Create
- Inspect or validate
- Publish
- Release or list
- Search
- Install
- Update
- Remove
- Govern

### Slide 7. Product Shape

Primary message:

> Multiple surfaces have distinct jobs, but together they form one closed product loop.

Components to present:

- `CLI`: operational entry point for day-to-day actions
- `Catalog`: metadata, validation, release workflow, and orchestration layer
- `npm-compatible registry`: package storage and distribution layer
- `Web`: browsing, portal, and human-facing entry point

The slide should explicitly show that these are complementary parts of one product rather than competing products.

### Slide 8. Boundaries and Non-goals

Primary message:

> `skill registry` solves the infrastructure problem of skill distribution, not every agent problem.

Out of scope:

- Large-model platform
- Agent runtime itself
- Generic workflow orchestrator
- Universal knowledge asset management system

### Slide 9. Closing Summary

Primary message:

> Let AI agent skills be produced, published, discovered, installed, and governed like software packages.

Close by restating the three audience takeaways from Section 3.

## 6. Visual and Tone Direction

The deck should feel product-oriented and deliberate, not like a developer demo.

### Tone

- Clear
- Internal-facing
- Confident but not inflated
- Product vocabulary first, technical nouns second

### Visual direction

- Clean, modern layout
- Strong hierarchy
- Limited color palette
- Simple diagrams over dense screenshots
- Use one lifecycle view and one component relationship view

### Avoid

- Overloaded architecture diagrams
- Long code snippets
- CLI command-heavy slides
- Excess implementation detail

## 7. Deck Component Design

### Content components

- Title slide
- Problem framing slide
- Definition slide
- Role-based slide
- Three-pillar problem-solution slide
- Lifecycle slide
- Component relationship slide
- Non-goals slide
- Summary slide

### Visual components

- Icon-backed cards for roles and capability groups
- A simple lifecycle flow for the end-to-end skill journey
- A compact relationship diagram for `CLI`, `Catalog`, `Web`, and `registry`

## 8. Content Flow

The information flow should move from abstract to concrete:

1. Business problem
2. Product definition
3. Personas
4. Value statement
5. Capability scope
6. Product surfaces
7. Boundary clarification
8. Summary

This order avoids premature detail and supports a 10-minute internal presentation.

## 9. Risks and Failure Modes

### Risk: Too technical

If the deck leans into backend internals, the audience will leave with fragmented system knowledge instead of product understanding.

Mitigation:

- Keep component descriptions to product responsibilities
- Use diagrams only to show relationships, not internals

### Risk: Too generic

If the deck becomes abstract platform language, the audience will not understand what is distinctive about `skill registry`.

Mitigation:

- Anchor every major claim to the skill lifecycle
- Name actual product surfaces: `CLI`, `Catalog`, `Web`, `npm-compatible registry`

### Risk: Boundary confusion

The audience may confuse the product with models, agents, or workflow engines.

Mitigation:

- Preserve a dedicated non-goals slide
- Repeat the product boundary in the closing summary

## 10. Implementation Notes

The Slidev implementation should:

- Use a product presentation style rather than a docs style
- Keep slide copy concise and speaker-friendly
- Include presenter notes where helpful
- Prefer diagrams, cards, and short bullets over long paragraphs
- Be structured for future export to PDF

## 11. Verification Plan

Before considering the deck complete:

1. Confirm the deck source matches the 9-slide structure above.
2. Confirm the main title, one-line definition, and non-goals are all present.
3. Run the Slidev dev server and verify the deck renders.
4. If export dependencies are available, export a PDF and confirm it is generated successfully.

## 12. Scope Guardrails

In this task, do not expand into:

- Detailed roadmap planning
- Technical deployment deep dives
- Customer-facing sales narrative
- Detailed demo scripts

The result should remain a short internal product-introduction deck.
