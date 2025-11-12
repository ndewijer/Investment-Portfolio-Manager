# Release Notes - Version X.X.X

**Release Date**: YYYY-MM-DD
**Version**: X.X.X
**Previous Version**: X.X.X

[Brief one-line description of this release]

---

## 🌟 What's New

### 1. [Feature Name]

[Brief description of the feature and why it's valuable]

#### Key Features

- **Feature 1**: Description
- **Feature 2**: Description
- **Feature 3**: Description
- **Feature 4**: Description

### 2. [Another Feature]

[Repeat as needed]

---

## 🚀 Features Added

### 1. [Feature Name in Detail]

**Location**: Where users find this feature

**Functionality**:
- What it does
- How it works
- Key capabilities

**When to Use**:
- Use case 1
- Use case 2

**Example**:
```
Concrete example of usage
```

### 2. [Another Feature]

[Repeat structure as needed]

---

## 💡 Use Cases

### Scenario 1: [Descriptive Name]
**Problem**: What problem does this solve?

**Solution**:
1. Step 1
2. Step 2
3. Step 3

**Result**: What's the outcome

**Time saved**: If applicable (e.g., "5 minutes → 30 seconds")

### Scenario 2: [Another Use Case]

[Repeat as needed]

---

## 📊 Technical Details

### Database Changes

**Migration**: `X.X.X_migration_name.py`

**New Tables/Columns**:
- `table.column` - Type, Purpose

**Modified**:
- Description of changes

**Backwards Compatible**: Yes/No

### API Changes

**New Endpoints**:
- `POST /api/endpoint` - Description
- `GET /api/endpoint` - Description

**Modified Endpoints**:
- `POST /api/existing` - What changed

**Deprecated Endpoints**:
- None / List any deprecated endpoints

### Frontend Changes

**New Components**:
- Component name - Purpose

**Updated Components**:
- Component name - Changes made

**CSS/Styling**:
- Major styling changes
- New themes/modes

---

## 🔧 Installation & Upgrade

### Fresh Installation

```bash
git clone [repo-url]
cd investment-portfolio-manager
cp .env.example .env
# Edit .env with your settings
docker compose up -d
docker compose exec backend flask db upgrade
```

### Upgrading from X.X.X

```bash
# 1. Backup database
docker compose exec backend cp /app/data/db/portfolio_manager.db \
  /app/data/db/portfolio_manager.db.backup

# 2. Pull latest changes
git checkout main  # or version tag
git pull

# 3. Run migration (if database changes)
docker compose exec backend flask db upgrade

# 4. Restart containers
docker compose restart

# 5. Verify version
# Check in UI or via API
```

**Note**: [Any special upgrade notes]

---

## 📚 Documentation

### New Documentation
- **`docs/NEW_DOC.md`** - Purpose and contents

### Updated Documentation
- **`README.md`** - What was updated
- **`docs/EXISTING_DOC.md`** - What was updated

---

## 🐛 Bug Fixes

- Fixed: [Bug description] (#issue-number if applicable)
- Fixed: [Another bug]
- Improved: [Error handling/messages]

---

## ⚙️ Configuration

### Environment Variables

**New Variables**:
- `VARIABLE_NAME` - Purpose, default value

**Modified Variables**:
- `EXISTING_VARIABLE` - What changed

### Settings

**New Settings**:
- Location: Where found
- Purpose: What it does
- Default: Default value

---

## 🔄 Breaking Changes

**None** / List any breaking changes

If breaking changes:
- What breaks
- How to migrate
- Code examples

---

## 🎯 Known Limitations

### Current Limitations

1. **Limitation description**
2. **Another limitation**

### Workarounds

Solutions or workarounds for limitations.

---

## ⚠️ Deprecation Notices

**None** / List any deprecations

- Feature/endpoint being deprecated
- Replacement
- Timeline for removal

---

## 📈 Performance Improvements

- Improvement 1: Description and impact
- Improvement 2: Description and impact
- Benchmark: Before vs After (if measured)

---

## 🔒 Security

### Enhancements

- Security improvement 1
- Security improvement 2

### Best Practices

- Recommended security practices
- Configuration recommendations

---

## 🧪 Testing

### Tested Scenarios

- ✅ Scenario 1
- ✅ Scenario 2
- ✅ Edge cases
- ✅ Error handling
- ✅ Performance under load
- ✅ Browser compatibility

### Recommended Testing

When deploying to your environment:

1. **Test core functionality**
   - Action to test
   - Expected result

2. **Test new features**
   - Feature to test
   - How to verify

---

## 🛠️ Troubleshooting

### Issue: [Common issue]

**Problem**: Description
**Cause**: Why it happens
**Solution**: How to fix

### Issue: [Another issue]

[Repeat as needed]

---

## 📞 Support

### Documentation

- Setup Guide: `docs/SETUP.md`
- User Guide: `docs/USER_GUIDE.md`
- Architecture: `docs/ARCHITECTURE.md`

### Getting Help

- **GitHub Issues**: [Report issues or questions](https://github.com/ndewijer/Investment-Portfolio-Manager/issues)
- **Pull Requests**: Link to relevant PRs (e.g., [#XX Feature Name](https://github.com/ndewijer/Investment-Portfolio-Manager/pull/XX))
- **This Release**: [vX.X.X](https://github.com/ndewijer/Investment-Portfolio-Manager/releases/tag/vX.X.X)

---

## 🎊 Summary

[2-3 paragraph summary of the release]

**Key Highlights**:
1. **Feature 1**: Brief description and benefit
2. **Feature 2**: Brief description and benefit
3. **Feature 3**: Brief description and benefit

**Impact**:
- User benefit 1
- User benefit 2
- Performance/quality improvement

---

## 📦 Release Assets

- **Source Code**: Available on GitHub
- **Docker Images**: If applicable
- **Documentation**: Updated on main branch

---

## 👏 Contributors

- @username - Contribution description
- Or note if solo project: "Solo developed by @username"

---

## 📅 Next Steps

**Looking ahead to version X.X.X**:
- Planned feature 1
- Planned feature 2
- See `todo/TODO.md` for full roadmap

---

**Version**: X.X.X
**Previous Version**: X.X.X
**Release Date**: YYYY-MM-DD
**Git Tag**: `vX.X.X`
