- #memory

  ## Project Context
  - This is a Flask + React project using SQLite (which stores naive datetimes)
  - ALWAYS use datetime.now() instead of datetime.now(UTC) for database comparisons
  - Follow existing code patterns: service layer (routes → services → models), UUID primary keys, comprehensive logging
  - Database location: backend/data/db/portfolio_manager.db (can query directly)

  ## Development Workflow
  - Run `pre-commit run --all-files` after completing tasks
  - Use TodoWrite tool for multi-step tasks to track progress
  - ALWAYS keep project documentation in context when making changes
  - Update documentation as architecture and code changes (README, ARCHITECTURE, MODELS, CONTRIBUTING, DEVELOPMENT, DATA, CSS docs)

  ## Pull Request Process
  - When completing a feature branch, create a detailed PR document (e.g., PULL_REQUEST_X.md)
  - **Template**: Copy from `templates/PULL_REQUEST_TEMPLATE.md`
  - Include in PR document:
    - Summary of changes
    - Technical details (files changed, new functions, API endpoints)
    - Testing performed
    - Documentation updates made
    - Screenshots/examples if UI changes
  - Can be removed after merge (or kept for reference)

  ## Release Management

  ### Before Release (Checklist)
  When user says "let's release version X.X.X", check:

  1. **Identify All PRs in Release**: ALWAYS check what PRs are included
     - Find previous release tag (e.g., `git tag --sort=-version:refname | head -5`)
     - List commits since last release: `git log refs/tags/X.X.X..HEAD --oneline`
     - Fetch merged PRs from GitHub: Check https://github.com/ndewijer/Investment-Portfolio-Manager/compare/X.X.X...main
     - Or use WebFetch on: https://github.com/ndewijer/Investment-Portfolio-Manager/pulls?q=is%3Apr+is%3Amerged+merged%3A%3E[DATE]
     - **IMPORTANT**: Include ALL merged PRs in release notes and CHANGELOG, not just the ones user mentions
     - Link to each PR: `[#XX PR Title](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/XX)`

  2. **Draft Release Notes**: Is there a RELEASE_NOTES_X.X.X_DRAFT.md file?
     - If no, offer to create from `templates/RELEASE_NOTES_TEMPLATE.md`
     - If yes, review with user and finalize
     - Remove "DRAFT" from filename when releasing

  3. **TODO.md Updates**:
     - Move completed items from "In Progress" to "Recently Completed"
     - Update "Current Version" at bottom
     - Update "Last Updated" date
     - Mark version as released in version history

  4. **Documentation Review**:
     - Check README.md has correct version
     - Verify ARCHITECTURE.md reflects current state
     - Ensure any new features are documented (docs/ folder)
     - Update CHANGELOG if it exists (include ALL PRs from step 1)

  5. **Version Numbers**:
     - Backend version metadata
     - Frontend package.json (if versioned)
     - Database schema version (if schema changed)

  6. **Testing Reminder**: Ask user if final testing is complete

  ### After Release
  1. Update todo/TODO.md:
     - Move version from "In Progress" to "Recently Completed"
     - Clear old items from "Recently Completed" if too many

  2. Clean up:
     - Remove draft documents
     - Update any "TBD" or "Coming Soon" references

  ## Documentation Maintenance

  ### When Adding Features
  - Update relevant docs immediately (don't defer):
    - User-facing: Add to docs/FEATURES.md or create feature-specific doc
    - Technical: Update ARCHITECTURE.md with new patterns
    - API changes: Note in docs/ (and eventually API_REFERENCE.md when generated)

  ### When Changing Architecture
  - Update ARCHITECTURE.md
  - Update MODELS.md if database changes
  - Update DEVELOPMENT.md if dev setup changes

  ### Keep LLM Context Fresh
  - Ensure documentation accurately reflects codebase
  - Remove outdated information
  - Add examples where helpful
  - Link related documents

  ## Version Planning

  ### Current Version Tracking
  - **Always check**: todo/TODO.md for current version
  - **When user mentions version**: Verify it's the next logical version
  - **Schema changes**: Require version bump (e.g., 1.3.0 → 1.4.0)
  - **Feature additions**: Minor version (e.g., 1.3.0 → 1.3.1)
  - **Bug fixes only**: Patch version (e.g., 1.3.1 → 1.3.2)

  ### Release Triggers
  When user says phrases like:
  - "Let's release this"
  - "Ready to merge to main"
  - "Time to ship X.X.X"
  - "Let's finalize the release"

  **RESPOND WITH**:
  "Before we release version X.X.X, let me check:
  1. Draft release notes status
  2. Documentation updates needed
  3. TODO.md updates
  4. Any remaining testing

  Should I prepare the release checklist?"

  ## Project Philosophy
  - Feature complete for current needs
  - Only add features when actually needed (personal workflow, IBKR integration, or specific requests)
  - Maintain simplicity - this is a 1-person side project
  - Documentation is critical for LLM context and future reference
