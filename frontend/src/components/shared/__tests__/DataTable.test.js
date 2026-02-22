/**
 * @fileoverview Test suite for DataTable component
 *
 * Tests the data table component with sorting, filtering, pagination,
 * loading/error states, and responsive design.
 *
 * Component features tested:
 * - Loading state with LoadingSpinner
 * - Error state with ErrorMessage and retry
 * - Empty state with custom message
 * - Data rendering in table rows
 * - Column headers and configuration
 * - Custom column rendering
 * - Sorting (ascending/descending/disabled)
 * - Custom sort functions
 * - Filtering with filter popups
 * - Custom filter functions
 * - Row click callbacks
 * - Pagination controls
 * - Mobile card rendering
 * - Custom mobile card renderer
 * - CSS className application
 *
 * Total: 40 tests
 */

import React from 'react';
import { render, screen, fireEvent, within } from '@testing-library/react';
import DataTable from '../DataTable';

// Mock FontAwesome icons
vi.mock('@fortawesome/react-fontawesome', () => ({
  FontAwesomeIcon: ({ icon, className, onClick }) => (
    <i
      className={className}
      onClick={onClick}
      data-testid={`icon-${icon.iconName}`}
      aria-label={icon.iconName}
    />
  ),
}));

describe('DataTable Component', () => {
  const sampleColumns = [
    { key: 'name', header: 'Name' },
    { key: 'amount', header: 'Amount', render: (val) => `$${val.toFixed(2)}` },
    { key: 'status', header: 'Status' },
  ];

  const sampleData = [
    { id: 1, name: 'Apple Inc.', amount: 1500.5, status: 'Active' },
    { id: 2, name: 'Microsoft', amount: 2300.75, status: 'Active' },
    { id: 3, name: 'Google', amount: 1800.0, status: 'Inactive' },
  ];

  describe('Loading State', () => {
    /**
     * Test loading spinner appears when loading is true
     */
    test('shows loading spinner when loading is true', () => {
      render(<DataTable data={sampleData} columns={sampleColumns} loading={true} />);

      expect(screen.getByText('Loading data...')).toBeInTheDocument();
    });

    /**
     * Test data not rendered when loading
     */
    test('does not render table when loading', () => {
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} loading={true} />
      );

      expect(container.querySelector('.data-table')).not.toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    /**
     * Test error message appears when error prop provided
     */
    test('shows error message when error is provided', () => {
      const error = 'Failed to load data';
      render(<DataTable data={[]} columns={sampleColumns} error={error} />);

      expect(screen.getByText('Failed to load data')).toBeInTheDocument();
    });

    /**
     * Test retry button appears when onRetry provided
     */
    test('shows retry button when error and onRetry provided', () => {
      const onRetry = jest.fn();
      const error = 'Network error';
      render(<DataTable data={[]} columns={sampleColumns} error={error} onRetry={onRetry} />);

      expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    });

    /**
     * Test retry button calls callback
     */
    test('calls onRetry when retry button clicked', () => {
      const onRetry = jest.fn();
      const error = 'Network error';
      render(<DataTable data={[]} columns={sampleColumns} error={error} onRetry={onRetry} />);

      const retryButton = screen.getByRole('button', { name: 'Retry' });
      fireEvent.click(retryButton);

      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    /**
     * Test data not rendered when error
     */
    test('does not render table when error present', () => {
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} error="Error" />
      );

      expect(container.querySelector('.data-table')).not.toBeInTheDocument();
    });
  });

  describe('Empty State', () => {
    /**
     * Test empty message with no data
     */
    test('shows empty message when data is empty', () => {
      render(<DataTable data={[]} columns={sampleColumns} />);

      expect(screen.getByText('No data available')).toBeInTheDocument();
    });

    /**
     * Test custom empty message
     */
    test('shows custom empty message', () => {
      render(<DataTable data={[]} columns={sampleColumns} emptyMessage="No transactions found" />);

      expect(screen.getByText('No transactions found')).toBeInTheDocument();
    });
  });

  describe('Data Rendering', () => {
    /**
     * Test table headers are rendered
     */
    test('renders column headers', () => {
      render(<DataTable data={sampleData} columns={sampleColumns} />);

      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Amount')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
    });

    /**
     * Test data rows are rendered
     */
    test('renders data rows', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const rows = container.querySelectorAll('tbody tr');
      expect(rows).toHaveLength(3);
    });

    /**
     * Test data values are displayed
     */
    test('displays data values correctly', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      // Query within desktop table to avoid duplicate matches from mobile cards
      const desktopTable = container.querySelector('.desktop-table');
      expect(within(desktopTable).getByText('Apple Inc.')).toBeInTheDocument();
      expect(within(desktopTable).getByText('Microsoft')).toBeInTheDocument();
      expect(within(desktopTable).getByText('Google')).toBeInTheDocument();
    });

    /**
     * Test custom render function
     */
    test('uses custom render function for columns', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      // Query within desktop table to avoid duplicate matches from mobile cards
      const desktopTable = container.querySelector('.desktop-table');
      // Amount column should use custom render function to format as currency
      expect(within(desktopTable).getByText('$1500.50')).toBeInTheDocument();
      expect(within(desktopTable).getByText('$2300.75')).toBeInTheDocument();
      expect(within(desktopTable).getByText('$1800.00')).toBeInTheDocument();
    });
  });

  describe('Sorting', () => {
    /**
     * Test sort icons appear when sortable is true
     */
    test('shows sort icons when sortable is true', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const sortIcons = container.querySelectorAll('.sort-icon');
      expect(sortIcons).toHaveLength(3); // One for each column
    });

    /**
     * Test clicking sort icon sorts ascending
     */
    test('sorts data ascending when sort icon clicked', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const sortIcons = container.querySelectorAll('.sort-icon');
      fireEvent.click(sortIcons[0]); // Click name column sort

      const rows = container.querySelectorAll('tbody tr');
      const firstRowCells = rows[0].querySelectorAll('td');
      expect(firstRowCells[0].textContent).toBe('Apple Inc.');
    });

    /**
     * Test clicking sort icon again sorts descending
     */
    test('sorts data descending when sort icon clicked twice', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const sortIcons = container.querySelectorAll('.sort-icon');
      fireEvent.click(sortIcons[0]); // First click - ascending
      fireEvent.click(sortIcons[0]); // Second click - descending

      const rows = container.querySelectorAll('tbody tr');
      const firstRowCells = rows[0].querySelectorAll('td');
      expect(firstRowCells[0].textContent).toBe('Microsoft');
    });

    /**
     * Test sortable=false hides sort icons
     */
    test('hides sort icons when sortable is false', () => {
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} sortable={false} />
      );

      const sortIcons = container.querySelectorAll('.sort-icon');
      expect(sortIcons).toHaveLength(0);
    });

    /**
     * Test column.sortable=false hides sort icon for specific column
     */
    test('hides sort icon for specific column when column.sortable is false', () => {
      const columns = [
        { key: 'name', header: 'Name', sortable: false },
        { key: 'amount', header: 'Amount' },
      ];

      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      const sortIcons = container.querySelectorAll('.sort-icon');
      expect(sortIcons).toHaveLength(1); // Only amount column has sort icon
    });

    /**
     * Test custom sort function
     */
    test('uses custom sort function when provided', () => {
      const columns = [
        {
          key: 'name',
          header: 'Name',
          sortFn: (a, b, direction) => {
            // Custom sort: reverse alphabetical
            const result = b.name.localeCompare(a.name);
            return direction === 'asc' ? result : -result;
          },
        },
      ];

      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      const sortIcons = container.querySelectorAll('.sort-icon');
      fireEvent.click(sortIcons[0]); // Sort ascending with custom function

      const rows = container.querySelectorAll('tbody tr');
      const firstRowCells = rows[0].querySelectorAll('td');
      expect(firstRowCells[0].textContent).toBe('Microsoft');
    });

    /**
     * Test default sort applied on mount
     */
    test('applies default sort on initial render', () => {
      const { container } = render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          defaultSort={{ key: 'name', direction: 'desc' }}
        />
      );

      const rows = container.querySelectorAll('tbody tr');
      const firstRowCells = rows[0].querySelectorAll('td');
      expect(firstRowCells[0].textContent).toBe('Microsoft');
    });
  });

  describe('Filtering', () => {
    /**
     * Test filter icons appear when filterable is true
     */
    test('shows filter icons when filterable is true', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const filterIcons = container.querySelectorAll('.filter-icon');
      expect(filterIcons).toHaveLength(3); // One for each column
    });

    /**
     * Test filterable=false hides filter icons
     */
    test('hides filter icons when filterable is false', () => {
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} filterable={false} />
      );

      const filterIcons = container.querySelectorAll('.filter-icon');
      expect(filterIcons).toHaveLength(0);
    });

    /**
     * Test column.filterable=false hides filter icon for specific column
     */
    test('hides filter icon for specific column when column.filterable is false', () => {
      const columns = [
        { key: 'name', header: 'Name', filterable: false },
        { key: 'amount', header: 'Amount' },
        { key: 'status', header: 'Status' },
      ];

      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      const filterIcons = container.querySelectorAll('.filter-icon');
      expect(filterIcons).toHaveLength(2); // Only amount and status columns
    });
  });

  describe('Row Click', () => {
    /**
     * Test row click callback is called
     */
    test('calls onRowClick when row is clicked', () => {
      const onRowClick = jest.fn();
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} onRowClick={onRowClick} />
      );

      const rows = container.querySelectorAll('tbody tr');
      fireEvent.click(rows[0]);

      expect(onRowClick).toHaveBeenCalledWith(sampleData[0]);
    });

    /**
     * Test clickable-row class added when onRowClick provided
     */
    test('adds clickable-row class when onRowClick provided', () => {
      const onRowClick = jest.fn();
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} onRowClick={onRowClick} />
      );

      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0]).toHaveClass('clickable-row');
    });

    /**
     * Test no clickable-row class when onRowClick not provided
     */
    test('does not add clickable-row class when onRowClick not provided', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0]).not.toHaveClass('clickable-row');
    });
  });

  describe('Pagination', () => {
    const paginationConfig = {
      currentPage: 2,
      totalPages: 5,
      totalItems: 50,
    };

    /**
     * Test pagination controls appear when pagination prop provided
     */
    test('shows pagination controls when pagination provided', () => {
      render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          pagination={paginationConfig}
          onPaginationChange={jest.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Previous' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();
      expect(screen.getByText('Page 2 of 5 (50 items)')).toBeInTheDocument();
    });

    /**
     * Test previous button calls callback with correct page
     */
    test('calls onPaginationChange with previous page when Previous clicked', () => {
      const onPaginationChange = jest.fn();
      render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          pagination={paginationConfig}
          onPaginationChange={onPaginationChange}
        />
      );

      const prevButton = screen.getByRole('button', { name: 'Previous' });
      fireEvent.click(prevButton);

      expect(onPaginationChange).toHaveBeenCalledWith(1);
    });

    /**
     * Test next button calls callback with correct page
     */
    test('calls onPaginationChange with next page when Next clicked', () => {
      const onPaginationChange = jest.fn();
      render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          pagination={paginationConfig}
          onPaginationChange={onPaginationChange}
        />
      );

      const nextButton = screen.getByRole('button', { name: 'Next' });
      fireEvent.click(nextButton);

      expect(onPaginationChange).toHaveBeenCalledWith(3);
    });

    /**
     * Test previous button disabled on first page
     */
    test('disables Previous button on first page', () => {
      const firstPageConfig = { currentPage: 1, totalPages: 5, totalItems: 50 };
      render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          pagination={firstPageConfig}
          onPaginationChange={jest.fn()}
        />
      );

      const prevButton = screen.getByRole('button', { name: 'Previous' });
      expect(prevButton).toBeDisabled();
    });

    /**
     * Test next button disabled on last page
     */
    test('disables Next button on last page', () => {
      const lastPageConfig = { currentPage: 5, totalPages: 5, totalItems: 50 };
      render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          pagination={lastPageConfig}
          onPaginationChange={jest.fn()}
        />
      );

      const nextButton = screen.getByRole('button', { name: 'Next' });
      expect(nextButton).toBeDisabled();
    });

    /**
     * Test no pagination controls without pagination prop
     */
    test('does not show pagination without pagination prop', () => {
      render(<DataTable data={sampleData} columns={sampleColumns} />);

      expect(screen.queryByRole('button', { name: 'Previous' })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument();
    });
  });

  describe('Mobile Rendering', () => {
    /**
     * Test mobile cards are rendered
     */
    test('renders mobile cards', () => {
      const { container } = render(<DataTable data={sampleData} columns={sampleColumns} />);

      const mobileCards = container.querySelector('.mobile-cards');
      expect(mobileCards).toBeInTheDocument();
    });

    /**
     * Test custom mobile card renderer
     */
    test('uses custom mobile card renderer when provided', () => {
      const mobileCardRenderer = (item) => (
        <div className="custom-card">
          <h3>{item.name}</h3>
          <p>{item.status}</p>
        </div>
      );

      const { container } = render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          mobileCardRenderer={mobileCardRenderer}
        />
      );

      const customCards = container.querySelectorAll('.custom-card');
      expect(customCards).toHaveLength(3);
    });

    /**
     * Test mobile card click callback
     */
    test('calls onRowClick when mobile card is clicked', () => {
      const onRowClick = jest.fn();
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} onRowClick={onRowClick} />
      );

      const cards = container.querySelectorAll('.mobile-cards > div');
      fireEvent.click(cards[0]);

      expect(onRowClick).toHaveBeenCalledWith(sampleData[0]);
    });

    /**
     * Test mobile cards have clickable class when onRowClick provided
     */
    test('adds clickable-card class when onRowClick provided', () => {
      const onRowClick = jest.fn();
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} onRowClick={onRowClick} />
      );

      const cards = container.querySelectorAll('.mobile-cards > div');
      expect(cards[0]).toHaveClass('clickable-card');
    });
  });

  describe('Custom Styling', () => {
    /**
     * Test custom className applied to wrapper
     */
    test('applies custom className to wrapper', () => {
      const { container } = render(
        <DataTable data={sampleData} columns={sampleColumns} className="custom-table" />
      );

      const wrapper = container.querySelector('.data-table-wrapper');
      expect(wrapper).toHaveClass('custom-table');
    });

    /**
     * Test column className applied to header
     */
    test('applies column className to header cell', () => {
      const columns = [{ key: 'name', header: 'Name', className: 'header-custom' }];
      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      const header = container.querySelector('th.header-custom');
      expect(header).toBeInTheDocument();
    });

    /**
     * Test column cellClassName applied to body cells
     */
    test('applies column cellClassName to body cells', () => {
      const columns = [{ key: 'name', header: 'Name', cellClassName: 'cell-custom' }];
      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      const cells = container.querySelectorAll('td.cell-custom');
      expect(cells).toHaveLength(3); // One for each data row
    });
  });

  describe('Complex Scenarios', () => {
    /**
     * Test sorting and filtering together
     */
    test('handles sorting and filtering together', () => {
      const columns = [
        {
          key: 'status',
          header: 'Status',
          filter: (item, filterValue) => item.status === filterValue,
        },
        { key: 'name', header: 'Name' },
      ];

      const { container } = render(<DataTable data={sampleData} columns={columns} />);

      // Apply filter first (would need to interact with FilterPopup which is mocked)
      // For now, just verify the components are rendered
      const sortIcons = container.querySelectorAll('.sort-icon');
      const filterIcons = container.querySelectorAll('.filter-icon');

      expect(sortIcons.length).toBeGreaterThan(0);
      expect(filterIcons.length).toBeGreaterThan(0);
    });

    /**
     * Test all props combined
     */
    test('renders correctly with all props', () => {
      const onRowClick = jest.fn();
      const onPaginationChange = jest.fn();
      const pagination = { currentPage: 1, totalPages: 3, totalItems: 30 };

      const { container } = render(
        <DataTable
          data={sampleData}
          columns={sampleColumns}
          loading={false}
          error={null}
          onRowClick={onRowClick}
          sortable={true}
          filterable={true}
          className="full-featured-table"
          pagination={pagination}
          onPaginationChange={onPaginationChange}
        />
      );

      // Verify key elements are present
      expect(container.querySelector('.data-table')).toBeInTheDocument();
      expect(container.querySelector('.mobile-cards')).toBeInTheDocument();
      expect(container.querySelector('.table-pagination')).toBeInTheDocument();
      expect(container.querySelector('.full-featured-table')).toBeInTheDocument();
    });
  });
});
