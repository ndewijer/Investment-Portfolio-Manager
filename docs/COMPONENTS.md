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

## ValueChart

Interactive time-series chart for portfolio performance visualisation. Built on vanilla ApexCharts (v5) with manual React lifecycle management to support React 19 Strict Mode.

### Features
- Multi-line datetime chart: value, cost, realized/unrealized gains, individual fund lines
- Built-in zoom (selection, scroll wheel), pan, and pinch-to-zoom on mobile
- Period shortcut buttons: All, 1Y, 3M, 1M
- Peak value annotation (dashed reference line at the highest total value)
- Dark/light theme via `useTheme()` — ApexCharts `theme.mode` is updated reactively
- Mobile fullscreen mode (CSS-based, preserves zoom state)
- Progressive data loading: `onZoomChange` bridges ApexCharts timestamp events to the index-based format expected by `useChartData`

### Usage Example

```jsx
<ValueChart
  data={chartData}           // [{date: 'YYYY-MM-DD', totalValue, totalCost, ...}]
  lines={chartLines}         // [{dataKey, name, color, strokeWidth, strokeDasharray?}]
  visibleMetrics={visibleMetrics}
  setVisibleMetrics={setVisibleMetrics}
  defaultZoomDays={365}
  onZoomChange={onZoomChange}
  onLoadAllData={loadAllData}
  totalDataRange={totalDataRange}
/>
```

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `data` | `Array` | required | Chart data points with `date` (YYYY-MM-DD) and metric fields |
| `lines` | `Array` | `[]` | Series config: `{ dataKey, name, color, strokeWidth, strokeDasharray?, opacity? }` |
| `height` | `number` | `400` | Chart height in pixels (desktop) |
| `visibleMetrics` | `Object\|false` | `false` | `{ value, cost, realizedGain, unrealizedGain, totalGain }` — controls metric toggle buttons |
| `setVisibleMetrics` | `Function` | — | Callback to update `visibleMetrics` |
| `defaultZoomDays` | `number` | `null` | Days to show on initial load (e.g. `365`) |
| `onZoomChange` | `Function` | — | Called with `{ isZoomed, xDomain: [startIdx, endIdx] }` on zoom — used by `useChartData` for progressive loading |
| `onLoadAllData` | `Function` | — | Called when the "All" button is pressed and full dataset is not yet loaded |
| `totalDataRange` | `Object` | `null` | `{ isFullDataset: boolean }` — prevents re-applying initial zoom after full data is loaded |
| `showTimeRangeButtons` | `boolean` | `false` | Show Last Month / All Time legacy buttons |
| `timeRange` | `string\|false` | `false` | Active time range for legacy buttons |
| `onTimeRangeChange` | `Function` | — | Callback for legacy time range buttons |

### Line Configuration

Each entry in the `lines` array:

| Property | Type | Description |
|----------|------|-------------|
| `dataKey` | `string` | Property name in the data array (e.g. `'totalValue'`, `'fund_1_value'`) |
| `name` | `string` | Legend label |
| `color` | `string` | Hex colour (e.g. `'#8884d8'`) |
| `strokeWidth` | `number` | Line thickness (default `2`) |
| `strokeDasharray` | `string` | SVG dash pattern e.g. `'5 5'` for dashed lines |
| `opacity` | `number` | `0`–`1`; converted to rgba for ApexCharts |

### Implementation Notes

- Uses **vanilla `ApexCharts`** (not `react-apexcharts`) to avoid React 19 Strict Mode destroy/remount errors
- Chart is initialised once in `useEffect([], [])` and updated via `instance.updateSeries()` / `instance.updateOptions()`
- Fullscreen is CSS-only: `.chart-wrapper.chart-fullscreen-active` uses `position: fixed` so the chart DOM element never moves (preserves zoom state)
- `onZoomChange` receives index-based state for compatibility with `useChartData`'s progressive-loading boundary detection

---

## More Components Coming Soon

Component documentation will be added here as the project evolves.

---

**Last Updated**: 2026-03-27 (Version 2.0.0)
**Maintained By**: @ndewijer
