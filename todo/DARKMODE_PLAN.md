# Dark Mode Implementation Plan

## Current Status ✅

### Completed Tasks
1. **Feature Flag System**: Created `ThemeContext` with feature flag control
2. **Developer Panel Controls**: Added dark mode toggle in Developer Panel
3. **Shared Components**: Updated all shared components to use class-based dark mode
4. **CSS Architecture**: Replaced `@media (prefers-color-scheme: dark)` with `.dark-theme` classes

### Current Implementation
- Dark mode is **DISABLED by default** via feature flag
- Can be enabled/disabled through Developer Panel
- When disabled, application stays in light mode regardless of system preference
- When enabled, users can toggle between light/dark themes
- All shared components support both themes

## Remaining Work for Full Implementation

### Phase 1: Core Application Styling 🔄

#### 1.1 Main Layout Components
- [ ] `App.css` - Main application container
- [ ] `Navigation.css` - Navigation bar and menu items
- [ ] `Modal.css` - Modal dialogs and overlays

#### 1.2 Page-Specific Components
- [ ] `Overview.css` - Dashboard and summary cards
- [ ] `Portfolios.css` - Portfolio listing and cards
- [ ] `PortfolioDetail.css` - Individual portfolio view
- [ ] `Funds.css` - Fund listing and management
- [ ] `FundDetail.css` - Individual fund view
- [ ] `LogViewer.css` - System logs interface

#### 1.3 Utility Components
- [ ] `Toast.css` - Notification toasts
- [ ] `FilterPopup.css` - Filter dropdown menus
- [ ] `CollapsibleInfo.css` - Expandable information sections
- [ ] `ValueChart.css` - Chart components

#### 1.4 Global Styles
- [ ] `common.css` - Global variables and base styles
- [ ] `index.css` - Root application styles

### Phase 2: Enhanced User Experience 🚀

#### 2.1 Theme Persistence
- [x] localStorage integration (already implemented)
- [ ] User preference API endpoint (backend)
- [ ] Sync theme across browser tabs

#### 2.2 Theme Toggle UI
- [ ] Add theme toggle button to Navigation bar
- [ ] Theme toggle in user settings/preferences
- [ ] Visual feedback for theme changes

#### 2.3 System Integration
- [x] Detect system preference (already implemented)
- [ ] Auto-switch based on time of day (optional)
- [ ] Respect system changes while app is running

### Phase 3: Advanced Features 🎨

#### 3.1 Theme Customization
- [ ] Multiple theme variants (blue-dark, green-dark, etc.)
- [ ] Custom accent colors
- [ ] High contrast mode support

#### 3.2 Component-Specific Enhancements
- [ ] Chart color schemes for dark mode
- [ ] Icon variations for better contrast
- [ ] Image/logo variants for dark backgrounds

#### 3.3 Performance Optimization
- [ ] CSS custom properties for theme switching
- [ ] Minimize layout shifts during theme changes
- [ ] Lazy load theme-specific assets

### Phase 4: Testing & Polish 🧪

#### 4.1 Cross-Browser Testing
- [ ] Chrome/Chromium compatibility
- [ ] Firefox compatibility
- [ ] Safari compatibility
- [ ] Edge compatibility

#### 4.2 Accessibility Testing
- [ ] Color contrast ratios (WCAG AA compliance)
- [ ] Screen reader compatibility
- [ ] Keyboard navigation in both themes

#### 4.3 Mobile Responsiveness
- [ ] Touch-friendly theme toggle
- [ ] Mobile-specific dark mode optimizations
- [ ] Battery usage considerations

## Implementation Guidelines

### CSS Architecture
```css
/* Use this pattern for all components */
.light-theme .component {
  /* Light theme styles */
}

.dark-theme .component {
  /* Dark theme styles */
}
```

### Color Palette
```css
:root {
  /* Light theme */
  --bg-primary: #ffffff;
  --bg-secondary: #f8f9fa;
  --text-primary: #333333;
  --text-secondary: #666666;
  --border-color: #e9ecef;
}

.dark-theme {
  /* Dark theme */
  --bg-primary: #1a1a1a;
  --bg-secondary: #2d2d2d;
  --text-primary: #ffffff;
  --text-secondary: #cccccc;
  --border-color: #404040;
}
```

### Component Update Checklist
For each component to be updated:
- [ ] Identify all color-related CSS properties
- [ ] Create light theme explicit styles
- [ ] Create dark theme styles
- [ ] Test both themes thoroughly
- [ ] Ensure proper contrast ratios
- [ ] Verify mobile responsiveness

## Testing Strategy

### Manual Testing
1. Enable dark mode feature flag
2. Toggle between light/dark themes
3. Navigate through all pages
4. Test all interactive components
5. Verify persistence across sessions

### Automated Testing
1. Add theme switching to component tests
2. Screenshot testing for visual regression
3. Accessibility testing with both themes

## Rollout Plan

### Phase 1: Internal Testing (Week 1-2)
- Complete core styling
- Internal team testing
- Bug fixes and refinements

### Phase 2: Beta Testing (Week 3)
- Enable feature flag for beta users
- Collect feedback
- Performance monitoring

### Phase 3: Full Release (Week 4)
- Enable by default for all users
- Monitor adoption metrics
- Support and bug fixes

## Success Metrics

- [ ] 100% of components support both themes
- [ ] WCAG AA contrast compliance
- [ ] No performance degradation
- [ ] Positive user feedback
- [ ] Increased user engagement (optional metric)

## Notes

- Current implementation prioritizes development experience by disabling dark mode by default
- Feature flag allows gradual rollout and easy rollback if issues arise
- Class-based approach provides better control than media queries
- All shared components are already updated and ready
