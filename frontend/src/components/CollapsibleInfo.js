import React, { useState } from 'react';
import './CollapsibleInfo.css';

/**
 * CollapsibleInfo component - Expandable/collapsible content section
 *
 * A simple accordion-style component that displays a clickable header with
 * a toggle icon (+/-) and shows/hides content when clicked. Useful for
 * organizing information in a compact, user-friendly way.
 *
 * @param {Object} props - Component props object
 * @param {string} props.title - The header text to display
 * @param {React.ReactNode} props.children - The content to show/hide when expanded
 * @returns {JSX.Element} Collapsible section with header and toggleable content
 *
 * @example
 * <CollapsibleInfo title="Additional Information">
 *   <p>This content can be expanded or collapsed</p>
 *   <ul>
 *     <li>Item 1</li>
 *     <li>Item 2</li>
 *   </ul>
 * </CollapsibleInfo>
 */
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
      {isExpanded && <div className="collapsible-content">{children}</div>}
    </div>
  );
};

export default CollapsibleInfo;
