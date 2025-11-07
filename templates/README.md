# Templates

This folder contains reusable templates for project documentation.

## 📄 Available Templates

### [PULL_REQUEST_TEMPLATE.md](./PULL_REQUEST_TEMPLATE.md)
**Use when**: Creating a detailed pull request document for a feature branch

**How to use**:
1. Copy the template to root directory
2. Rename to `PULL_REQUEST_[FEATURE_NAME].md`
3. Fill in all sections
4. Remove sections marked as "N/A" if not applicable
5. Include in the PR when merging to main

**Example**: `PULL_REQUEST_BULK_ALLOCATION.md`

---

### [RELEASE_NOTES_TEMPLATE.md](./RELEASE_NOTES_TEMPLATE.md)
**Use when**: Preparing release notes for a new version

**How to use**:
1. Copy the template to root directory
2. Rename to `RELEASE_NOTES_X.X.X_DRAFT.md` (keep DRAFT suffix while working)
3. Fill in all sections as you develop features
4. When ready to release, review and finalize
5. Use the content for GitHub release notes
6. Delete the file after creating the GitHub release

**Example**: `RELEASE_NOTES_1.3.1_DRAFT.md` → Copy to GitHub release → Delete file

**Note**: Release notes are published on GitHub, not stored in the repository

---

## 📝 Usage Guidelines

### Pull Request Documents

**When to create**:
- Feature branches with multiple changes
- New major features
- Significant refactoring
- When merging to main branch

**Don't create for**:
- Small bug fixes (can be in commit message)
- Documentation-only changes
- Minor tweaks

**Lifecycle**:
1. Create when feature work begins (optional, can create at end)
2. Update as you work on the feature
3. Finalize before merging PR
4. Can be removed after merge (or keep for reference)

### Release Notes

**When to create**:
- Any version release (even patch versions)
- Start early - update as features are added
- Keep in DRAFT mode until release

**Required sections**:
- What's New (user-facing)
- Technical Details (developer-facing)
- Installation & Upgrade (operational)
- Summary (TL;DR)

**Optional sections**:
- Use Cases (helpful for significant features)
- Breaking Changes (only if applicable)
- Performance Improvements (if measured)
- Contributors (if accepting contributions)

---

## 🎯 Template Philosophy

**Comprehensive but flexible**:
- Templates include all possible sections
- Not every section is required for every release
- Remove sections that don't apply
- Add sections if needed for specific situations

**User-focused**:
- Write for users first, developers second
- Include examples and screenshots
- Explain "why" not just "what"
- Provide use cases and scenarios

**Maintainable**:
- Keep templates up to date as project evolves
- Add sections based on actual needs
- Remove sections that are never used

---

## 🔄 Template Maintenance

### Updating Templates

**When to update**:
- After major releases (incorporate learnings)
- When project structure changes
- When adding new documentation patterns
- When removing outdated sections

**How to update**:
1. Edit template file directly
2. Commit changes with explanation
3. Update this README if structure changes significantly

### Version History

Track significant template changes:
- **2025-01-07**: Initial templates created based on IBKR bulk allocation PR and v1.3.1 release notes

---

## 💡 Tips

### For Pull Requests
- Start with the summary - write it last when you know what you built
- Include screenshots for any UI changes
- Document "why" decisions were made, not just "what" changed
- List testing performed - this builds confidence

### For Release Notes
- Update as you go, don't wait until release day
- Include use cases - they help users understand value
- Be honest about limitations and known issues
- Celebrate improvements but don't oversell

### General
- Markdown supports collapsible sections: `<details><summary>Title</summary>Content</details>`
- Include code examples with syntax highlighting
- Link to related documentation
- Use emojis sparingly for visual breaks

---

## 📚 Related Documentation

- **CLAUDE.md** - References these templates for release process
- **todo/TODO.md** - Tracks what needs documenting
- **docs/CONTRIBUTING.md** - May reference PR template usage

---

**Last Updated**: 2025-01-07
