# CSS Architecture

## Overview
The application implements a centralized CSS architecture focusing on maintainability, reusability, and consistency. The system uses CSS variables for theming and follows a component-based approach while maintaining global styles for common elements.

## Color System

### Core Colors
```css
:root {
  --primary-color: #1976d2;    /* Standard buttons, primary actions */
  --add-color: #4caf50;        /* Add/create actions */
  --edit-color: #ff9800;       /* Edit/modify actions */
  --delete-color: #f44336;     /* Delete/remove actions */
  --archive-color: #9e9e9e;    /* Archive/disable actions */
  --positive-color: #4caf50;   /* Positive values/changes */
  --negative-color: #f44336;   /* Negative values/changes */
}
```

## Component Structure

### Common Components
1. **Summary Cards**
   - Responsive grid layout
   - Consistent spacing and shadows
   - Standardized typography
   ```css
   .summary-cards {
     display: grid;
     grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
     gap: 20px;
   }
   ```

2. **Action Buttons**
   - Color-coded by action type
   - Consistent hover states
   - Standardized padding and borders
   ```css
   .action-buttons button {
     margin: 4px;
     padding: 8px 12px;
     border-radius: 3px;
   }
   ```

3. **Chart Sections**
   - Unified container styling
   - Consistent spacing
   - Standard control layouts

4. **Modals**
   - Two reusable modal components: `Modal` (base) and `FormModal` (form-specific)
   - Click outside to close (configurable)
   - Escape key to close
   - Scrollable content for tall modals
   - Responsive sizing with viewport constraints
   - Multiple size variants (small, medium, large, xlarge)
   - Dark mode support

   **Usage Examples:**
   ```jsx
   // Basic Modal
   <Modal
     isOpen={isOpen}
     onClose={handleClose}
     title="Modal Title"
     size="medium"
     closeOnOverlayClick={true}
   >
     <div>Modal content here</div>
   </Modal>

   // Form Modal (preferred for forms)
   <FormModal
     isOpen={isOpen}
     onClose={handleClose}
     title="Add Item"
     onSubmit={handleSubmit}
     loading={submitting}
     submitText="Create"
     size="medium"
   >
     <FormField
       label="Name"
       type="text"
       value={name}
       onChange={setName}
       required
     />
   </FormModal>
   ```

   **Modal Features:**
   - **Base Modal** (`components/Modal.js`):
     - Basic modal wrapper with overlay
     - Header with title and close button
     - Scrollable body
     - Props: `isOpen`, `onClose`, `title`, `size`, `closeOnOverlayClick`, `className`

   - **Form Modal** (`components/shared/FormModal.js`):
     - Extends base Modal with form handling
     - Built-in submit/cancel buttons
     - Loading states and error display
     - Form field components included
     - Props: All Modal props plus `onSubmit`, `submitText`, `cancelText`, `loading`, `error`, `submitDisabled`

   **Size Variants:**
   - `small`: 400px max-width
   - `medium`: 600px max-width (default)
   - `large`: 900px max-width
   - `xlarge`: 1200px max-width

   **Best Practices:**
   - Use `FormModal` for all form-based modals (create, edit operations)
   - Use base `Modal` for view-only or custom content
   - Set `closeOnOverlayClick={false}` for forms to prevent accidental data loss
   - Use appropriate size based on content (most forms work with `medium`)
   - Leverage built-in `FormField` component for consistent form styling

## File Organization

```
frontend/src/
├── styles/
│   └── common.css          # Global styles and variables
├── pages/
│   ├── FundDetail.css      # Page-specific styles
│   ├── Overview.css
│   └── PortfolioDetail.css
└── components/
    └── ValueChart.css      # Component-specific styles
```

## Style Guidelines

### 1. Component Styling
- Use BEM-like naming for component classes
- Keep component-specific styles in their CSS files
- Import common styles where needed

### 2. Common Elements
- Use predefined variables for colors
- Maintain consistent spacing using common classes
- Follow standard button styling patterns

### 3. Responsive Design
- Use CSS Grid for layouts
- Implement flexible components
- Use relative units where appropriate

### 4. Interactive States
```css
/* Standard hover effect */
button:hover {
  opacity: 0.9;
}

/* Active state for toggles */
.button.active {
  background-color: var(--primary-color);
  color: white;
}
```

## Best Practices

1. **CSS Variables**
   - Use for colors and theming
   - Maintain in common.css
   - Use semantic naming

2. **Component Isolation**
   - Namespace component styles
   - Avoid deep nesting
   - Use specific selectors

3. **Performance**
   - Minimize selector specificity
   - Use efficient selectors
   - Avoid redundant styles

4. **Maintenance**
   - Document complex styles
   - Keep related styles together
   - Use consistent formatting

## Common Patterns

### 1. Card Layout
```css
.card {
  background-color: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}
```

### 2. Button Styles
```css
.button {
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s;
}
```

### 3. Form Elements
```css
.form-group {
  margin-bottom: 15px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}
```

## Future Considerations

1. **Potential Improvements**
   - Migration to CSS Modules
   - Implementation of CSS-in-JS
   - Theme system expansion

2. **Maintenance Tasks**
   - Regular audit of unused styles
   - Performance optimization
   - Documentation updates

3. **Style Guide Evolution**
   - Component library development
   - Design system documentation
   - Pattern library creation
