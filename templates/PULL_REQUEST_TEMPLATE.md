# Pull Request: [Feature Name]

## Summary

Brief description of what this PR adds or changes (2-3 sentences).

**Key Changes**:
1. Feature/change 1
2. Feature/change 2
3. Feature/change 3

---

## Changes Overview

- **X files changed**: Brief description of what types of files
- **New features**: Yes/No
- **Database changes**: Yes/No (if yes, mention migration file)
- **Breaking changes**: Yes/No
- **Version**: What version this targets (e.g., 1.3.1)

---

## 🎯 Key Features

### 1. [Feature Name]

**What it does**: One-line description

#### User Interface
- Describe UI changes
- Include screenshots if visual changes

#### Backend Changes
- API endpoints added/modified
- Service layer changes
- Database changes

#### Features
- ✅ Feature point 1
- ✅ Feature point 2
- ✅ Feature point 3

### 2. [Another Feature if applicable]

[Repeat structure as needed]

---

## 📁 Files Changed

### Frontend

#### `frontend/src/pages/ComponentName.js` (~X lines added/modified)

**New State Variables**:
```javascript
// List any new state variables
```

**New Functions**:
- `functionName()` - Description
- `anotherFunction()` - Description

**Updated Functions**:
- `existingFunction()` - What changed

**UI Changes**:
- List visible changes
- Component additions/modifications

#### `frontend/src/pages/ComponentName.css` (~X lines added)

**New Styles**:
- `.class-name` - Purpose
- `.another-class` - Purpose

**Updated Styles**:
- Modified existing classes

### Backend

#### `backend/app/routes/route_name.py` (~X lines added/modified)

**New Endpoints**:
- `POST /api/path` - Description
- `GET /api/path` - Description

**Updated Endpoints**:
- `POST /api/existing` - What changed

**New Functions**:
- `function_name()` - Purpose

### Database (if applicable)

#### Migration: `migrations/versions/X_description.py`

**Changes**:
- Added table/column: Description
- Modified: Description
- Removed: Description

**Backwards Compatible**: Yes/No

---

## 📚 Documentation Changes

### New Documentation
- `docs/NEW_DOC.md` - Purpose

### Updated Documentation
- `README.md` - What changed
- `docs/ARCHITECTURE.md` - What changed
- `docs/EXISTING_DOC.md` - What changed

---

## 🧪 Testing

### Manual Testing Performed
- [ ] Tested scenario 1
- [ ] Tested scenario 2
- [ ] Tested edge case handling
- [ ] Tested error scenarios
- [ ] Browser compatibility (Chrome/Firefox/Safari)

### Test Results
- Feature 1: ✅ Works as expected
- Feature 2: ✅ Works as expected
- Edge cases: ✅ Handled correctly

### Known Issues
- None / List any known issues

---

## 🔧 Technical Details

### Architecture Changes
- Describe any architectural patterns introduced
- Service layer changes
- State management changes

### API Changes

**New Endpoints**:

**`POST /api/endpoint`**
```json
Request:
{
    "field": "value"
}

Response:
{
    "result": "value"
}
```

### Performance Considerations
- Any performance impacts (positive or negative)
- Optimization opportunities

### Security Considerations
- Any security implications
- Validation added
- Authentication/authorization changes

---

## 📊 Statistics

- **Lines Added**: ~X
- **Lines Removed**: ~Y
- **Files Changed**: Z
- **New Functions/Components**: Count
- **Documentation Added**: X lines

---

## ✅ Checklist

- [ ] Code follows project style guidelines
- [ ] Pre-commit checks pass (`pre-commit run --all-files`)
- [ ] Documentation updated
- [ ] No console errors in browser
- [ ] No Python warnings/errors
- [ ] Manual testing completed
- [ ] Ready to merge

---

## 📸 Screenshots (if UI changes)

### Before
[Screenshot or N/A]

### After
[Screenshot]

---

## 🔗 Related Issues/PRs

- Closes #X (if applicable)
- Related to #Y
- Depends on #Z

---

## 📝 Additional Notes

Any additional context, gotchas, or things to be aware of.

---

**Branch**: `feature/branch-name`
**Target**: `main`
**Reviewer**: @username (if applicable)
