import React, { useEffect, useRef } from 'react';
import DatePicker from 'react-datepicker';
import './FilterPopup.css';

/**
 * FilterPopup component - Flexible filter UI with multiple input types
 *
 * A versatile popup component that renders different filter controls based on type:
 * - 'date': Date range picker (from/to dates)
 * - 'datetime': DateTime range picker with time selection (UTC)
 * - 'multiselect': Multi-select dropdown (using react-select or native)
 * - 'text': Simple text input filter
 *
 * Features:
 * - Click outside to close
 * - Escape key to close
 * - Positioned absolutely based on trigger element
 * - Supports custom component injection (e.g., react-select)
 * - Auto-focuses text inputs
 *
 * The popup appears next to the element that triggered it and handles
 * various filter scenarios common in data tables.
 *
 * @param {Object} props - Component props object
 * @param {string} props.type - Filter type: 'date', 'datetime', 'multiselect', or 'text'
 * @param {boolean} props.isOpen - Whether the popup is currently visible
 * @param {Function} props.onClose - Callback when popup is closed
 * @param {Object} props.position - Position object with top and left properties
 * @param {any} props.value - Current filter value (type depends on filter type)
 * @param {Function} props.onChange - Callback when value changes
 * @param {Array} [props.options=[]] - Options array for multiselect (objects with label/value)
 * @param {Date} props.fromDate - Start date for date/datetime filters
 * @param {Date} props.toDate - End date for date/datetime filters
 * @param {Function} props.onFromDateChange - Callback when start date changes
 * @param {Function} props.onToDateChange - Callback when end date changes
 * @param {React.Component} props.Component - Custom component for rendering (e.g., react-select)
 * @param {boolean} props.isMulti - Whether multiselect allows multiple selections
 * @returns {JSX.Element|null} Filter popup or null if not open
 *
 * @example
 * <FilterPopup
 *   type="date"
 *   isOpen={showFilter}
 *   onClose={() => setShowFilter(false)}
 *   position={{ top: 100, left: 50 }}
 *   fromDate={startDate}
 *   toDate={endDate}
 *   onFromDateChange={setStartDate}
 *   onToDateChange={setEndDate}
 * />
 */
const FilterPopup = ({
  type,
  isOpen,
  onClose,
  position,
  value,
  onChange,
  options = [],
  fromDate,
  toDate,
  onFromDateChange,
  onToDateChange,
  Component,
  isMulti,
}) => {
  const popupRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (popupRef.current && !popupRef.current.contains(event.target)) {
        onClose();
      }
    };

    const handleEscapeKey = (event) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscapeKey);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscapeKey);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const renderContent = () => {
    switch (type) {
      case 'date':
        return (
          <div className="date-range-inputs">
            <div className="date-input">
              <label>From:</label>
              <DatePicker
                selected={fromDate}
                onChange={onFromDateChange}
                dateFormat="yyyy-MM-dd"
                isClearable
              />
            </div>
            <div className="date-input">
              <label>To:</label>
              <DatePicker
                selected={toDate}
                onChange={onToDateChange}
                dateFormat="yyyy-MM-dd"
                minDate={fromDate}
                isClearable
              />
            </div>
          </div>
        );

      case 'multiselect':
        if (Component) {
          return (
            <Component
              options={options}
              value={value}
              onChange={onChange}
              isMulti={isMulti}
              isClearable={true}
              closeMenuOnSelect={false}
              styles={{
                control: (base) => ({
                  ...base,
                  minWidth: '200px',
                }),
                menu: (base) => ({
                  ...base,
                  position: 'relative',
                  zIndex: 2,
                }),
                clearIndicator: (base) => ({
                  ...base,
                  cursor: 'pointer',
                  padding: '6px',
                  ':hover': {
                    color: '#666',
                  },
                }),
              }}
              classNamePrefix="react-select"
            />
          );
        }
        return (
          <select
            multiple
            value={Array.isArray(value) ? value : []}
            onChange={(e) => {
              const selectedOptions = Array.from(
                e.target.selectedOptions,
                (option) => option.value
              );
              onChange(selectedOptions);
            }}
          >
            {options.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        );

      case 'text':
        return (
          <input
            type="text"
            value={value || ''}
            onChange={(e) => onChange(e.target.value)}
            placeholder="Filter..."
            autoFocus
          />
        );

      case 'datetime':
        return (
          <div className="date-range-inputs">
            <div className="date-input">
              <label>From (UTC):</label>
              <DatePicker
                selected={fromDate}
                onChange={onFromDateChange}
                showTimeSelect
                timeFormat="HH:mm"
                timeIntervals={15}
                dateFormat="yyyy-MM-dd HH:mm"
                //timeCaption="Time (UTC)"
                isClearable
                //utcOffset={0}
              />
            </div>
            <div className="date-input">
              <label>To (UTC):</label>
              <DatePicker
                selected={toDate}
                onChange={onToDateChange}
                showTimeSelect
                timeFormat="HH:mm"
                timeIntervals={15}
                dateFormat="yyyy-MM-dd HH:mm"
                //timeCaption="Time (UTC)"
                minDate={fromDate}
                isClearable
                //utcOffset={0}
              />
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div
      className="filter-popup"
      ref={popupRef}
      style={{
        top: position.top,
        left: position.left,
      }}
    >
      {renderContent()}
    </div>
  );
};

export default FilterPopup;
