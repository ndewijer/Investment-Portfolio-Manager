import React, { useRef, useEffect } from 'react';
import DatePicker from 'react-datepicker';
import './FilterPopup.css';
import "react-datepicker/dist/react-datepicker.css";

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
  onToDateChange
}) => {
  const popupRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (popupRef.current && !popupRef.current.contains(event.target)) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
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

      case 'select':
        return (
          <select
            value={value}
            onChange={(e) => onChange(e.target.value)}
          >
            <option value="">All</option>
            {options.map(option => (
              <option key={option} value={option}>{option}</option>
            ))}
          </select>
        );

      case 'multiselect':
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
              <option key={option} value={option}>{option}</option>
            ))}
          </select>
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