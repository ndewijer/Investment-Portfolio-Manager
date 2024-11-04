import React, { useEffect, useRef } from 'react';
import DatePicker from 'react-datepicker';
import './FilterPopup.css';

const FilterPopup = ({ 
  type, 
  isOpen, 
  onClose, 
  position, 
  value, 
  onChange,
  options = [],
  dateRange = false,
  fromDate,
  toDate,
  onFromDateChange,
  onToDateChange,
  Component
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
              labelledBy="Select options"
              hasSelectAll={true}
              disableSearch={true}
            />
          );
        }
        return (
          <select
            multiple
            value={Array.isArray(value) ? value : []}
            onChange={(e) => {
              const selectedOptions = Array.from(e.target.selectedOptions, option => option.value);
              onChange(selectedOptions);
            }}
          >
            {options.map(option => (
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
        left: position.left
      }}
    >
      {renderContent()}
    </div>
  );
};

export default FilterPopup; 