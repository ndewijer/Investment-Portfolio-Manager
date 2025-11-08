# React Components Documentation

This document contains usage documentation for reusable React components in the Investment Portfolio Manager application.

---

## DataTable

Reusable table component with sorting, filtering, and responsive design for displaying tabular data.

### Features
- Desktop table view with sortable/filterable columns
- Mobile card view for small screens
- Pagination support
- Custom renderers for cells
- Default sorting configuration
- Loading and error states

### Usage Example

```jsx
<DataTable
  data={items}
  columns={[
    {
      key: 'date',
      header: 'Date',
      render: (value) => new Date(value).toLocaleDateString(),
      sortable: true,
      filterable: true,
      sortFn: (a, b, direction) => {
        const dateA = new Date(a.date);
        const dateB = new Date(b.date);
        return direction === 'asc' ? dateA - dateB : dateB - dateA;
      },
    },
    {
      key: 'amount',
      header: 'Amount',
      render: (value) => formatCurrency(value),
      sortable: true,
    },
  ]}
  loading={loading}
  emptyMessage="No data available"
  sortable={true}
  filterable={true}
  defaultSort={{ key: 'date', direction: 'desc' }}
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `data` | `Array` | `[]` | Array of items to display |
| `columns` | `Array` | `[]` | Column configuration array (see Column Configuration below) |
| `loading` | `boolean` | `false` | Show loading spinner |
| `error` | `string/object` | `null` | Display error message |
| `emptyMessage` | `string` | `'No data available'` | Message when no data |
| `sortable` | `boolean` | `true` | Enable/disable sorting globally |
| `filterable` | `boolean` | `true` | Enable/disable filtering globally |
| `defaultSort` | `object` | `null` | Initial sort configuration `{ key: string, direction: 'asc' \| 'desc' }` |
| `onRowClick` | `function` | `null` | Callback for row clicks `(item) => void` |
| `mobileCardRenderer` | `function` | `null` | Custom mobile card renderer `(item) => ReactNode` |
| `pagination` | `object` | `null` | Pagination config `{ currentPage, totalPages, totalItems }` |
| `onPaginationChange` | `function` | `null` | Pagination callback `(newPage) => void` |
| `className` | `string` | `''` | Additional CSS class names |
| `onRetry` | `function` | `null` | Retry callback for error states |

### Column Configuration

Each column object in the `columns` array supports:

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `key` | `string` | required | Data property key to display |
| `header` | `string` | required | Column header text |
| `render` | `function` | `null` | Custom render function `(value, item) => ReactNode` |
| `sortable` | `boolean` | `true` | Enable sorting for this column |
| `filterable` | `boolean` | `true` | Enable filtering for this column |
| `sortFn` | `function` | `null` | Custom sort function `(a, b, direction) => number` |
| `filter` | `function` | `null` | Custom filter function `(item, filterValue) => boolean` |
| `filterType` | `string` | `'text'` | Filter UI type ('text', 'select', 'date', etc.) |
| `filterOptions` | `Array` | `[]` | Options for select-type filters |
| `onFilterClick` | `function` | `null` | Custom filter click handler |
| `className` | `string` | `''` | Column header class name |
| `cellClassName` | `string` | `''` | Column cell class name |

### Best Practices

1. **Default Sorting**: Use `defaultSort` to set initial table sorting (e.g., newest dates first)
   ```jsx
   defaultSort={{ key: 'date', direction: 'desc' }}
   ```

2. **Custom Sort Functions**: Provide custom `sortFn` for complex data types (dates, nested objects)
   ```jsx
   sortFn: (a, b, direction) => {
     const dateA = new Date(a.date);
     const dateB = new Date(b.date);
     return direction === 'asc' ? dateA - dateB : dateB - dateA;
   }
   ```

3. **Custom Renderers**: Use `render` function for formatting (currency, dates, badges)
   ```jsx
   render: (value) => formatCurrency(value)
   ```

4. **Mobile Views**: Provide custom mobile card renderer for better mobile UX
   ```jsx
   mobileCardRenderer={(item) => (
     <div className="custom-card">
       <h3>{item.name}</h3>
       <p>{item.description}</p>
     </div>
   )}
   ```

5. **Keep Configuration Close**: Keep column configuration close to the DataTable usage for maintainability

### Examples

#### Basic Table
```jsx
<DataTable
  data={users}
  columns={[
    { key: 'name', header: 'Name' },
    { key: 'email', header: 'Email' },
  ]}
/>
```

#### Table with Custom Formatting
```jsx
<DataTable
  data={transactions}
  columns={[
    {
      key: 'date',
      header: 'Date',
      render: (value) => new Date(value).toLocaleDateString(),
    },
    {
      key: 'amount',
      header: 'Amount',
      render: (value) => formatCurrency(value),
    },
  ]}
  defaultSort={{ key: 'date', direction: 'desc' }}
/>
```

---

## More Components Coming Soon

Component documentation will be added here as the project evolves.
