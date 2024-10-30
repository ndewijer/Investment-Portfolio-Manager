import React, { useState } from 'react';
import './CollapsibleInfo.css';

const CollapsibleInfo = ({ title, children }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div className="collapsible-info">
      <button 
        className={`collapsible-header ${isExpanded ? 'expanded' : ''}`}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {title}
        <span className="toggle-icon">{isExpanded ? '−' : '+'}</span>
      </button>
      {isExpanded && (
        <div className="collapsible-content">
          {children}
        </div>
      )}
    </div>
  );
};

export default CollapsibleInfo; 