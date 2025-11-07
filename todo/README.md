# TODO & Planning Folder

This folder contains all planning documents, feature roadmaps, and task lists for the Investment Portfolio Manager project.

## 📂 Files

### [TODO.md](./TODO.md)
**Main task list and project roadmap**

Contains:
- Recently completed features (v1.3.x)
- Current work in progress
- Planned features by priority (High/Medium/Low)
- Technical debt items
- Known issues and bugs
- Version planning (1.3.1 → 2.0.0)
- Success metrics

**Use this file to**:
- Track overall project progress
- Understand what's been done
- See what's coming next
- Plan future work

---

### [API_DOCUMENTATION_GENERATION_PLAN.md](./API_DOCUMENTATION_GENERATION_PLAN.md)
**Comprehensive plan for automated API documentation**

Contains:
- Current state analysis (60+ endpoints, inconsistent docs)
- Proposed solution (Python AST-based parser)
- 5-phase implementation plan
- Script design and architecture
- Documentation output format examples
- Maintenance plan
- Future enhancements (OpenAPI, interactive docs)
- Route standardization recommendations

**Status**: Planning phase complete
**Target Version**: 1.5.0+

---

### [CURRENCY_CONVERSION_PLAN.md](./CURRENCY_CONVERSION_PLAN.md)
**Detailed plan for multi-currency support implementation**

Contains:
- Current limitations (single currency per portfolio)
- Proposed architecture (conversion at transaction time)
- Database schema changes
- Exchange rate API integration
- UI/UX changes
- Migration strategy
- Performance considerations

**Status**: Planning phase complete
**Target Version**: 1.4.0

---

## 🎯 How to Use This Folder

### For Developers

1. **Starting new work**: Check TODO.md to see what's prioritized
2. **Planning features**: Review existing plans before creating new ones
3. **Updating progress**: Mark items as complete in TODO.md when done
4. **Adding new plans**: Create detailed plan docs (like the two examples) for complex features

### For Contributors

1. Read TODO.md to understand project direction
2. Look for "High Priority" or "In Progress" items to contribute to
3. Check plans for technical details before implementing large features
4. Update TODO.md when work is completed

### For Project Maintainers

1. Review TODO.md monthly to update priorities
2. Create detailed plan docs for major features before starting work
3. Link plan docs in TODO.md for reference
4. Update version planning based on progress

---

## 📝 Document Standards

### Planning Documents Should Include:

1. **Overview**: What problem does this solve?
2. **Current State**: What exists today?
3. **Proposed Solution**: How will we solve it?
4. **Implementation Plan**: Phased approach with timeline
5. **Technical Details**: Database, API, UI changes
6. **Risks & Mitigation**: What could go wrong?
7. **Success Metrics**: How do we measure success?

### TODO.md Updates:

- Mark completed items with `[x]` checkbox
- Add new items under appropriate priority section
- Update version planning when versions are released
- Link to detailed plan docs for complex features
- Update "Last Updated" date when making changes

---

## 🔄 Version Control

All files in this folder should be:
- ✅ Committed to git
- ✅ Included in pull requests for related features
- ✅ Updated when plans change
- ✅ Reviewed during release planning

---

## 📅 Review Schedule

- **TODO.md**: Review monthly or after each release
- **Plan Documents**: Review before starting implementation
- **Folder**: Periodic cleanup to archive completed plans

---

**Last Updated**: 2025-01-07
