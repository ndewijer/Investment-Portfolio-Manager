import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import './ValueChart.css';
import { useFormat } from '../context/FormatContext';

const ValueChart = ({
  data,
  height = 400,
  lines = [],
  timeRange = false,
  onTimeRangeChange,
  showTimeRangeButtons = false,
  visibleMetrics = false,
  setVisibleMetrics,
  defaultZoomDays = null, // New prop: number of days to show by default (e.g., 365 for last year)
  onZoomChange, // New prop: callback for zoom state changes
  onLoadAllData, // New prop: callback to load all available data
  totalDataRange = null, // New prop: information about the total data range
}) => {
  const { formatCurrency } = useFormat();

  // Full-screen state for mobile
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [showMetricControls, setShowMetricControls] = useState(false);
  const [showZoomControls, setShowZoomControls] = useState(false);

  // Zoom state management
  const [zoomState, setZoomState] = useState({
    isZoomed: false,
    zoomLevel: 1,
    xDomain: null,
    yDomain: null,
    panOffset: { x: 0, y: 0 },
  });

  // Refs for touch handling and drag selection
  const chartRef = useRef(null);
  const touchStartRef = useRef(null);
  const lastTouchDistanceRef = useRef(null);
  const isPanningRef = useRef(false);
  const dragStartRef = useRef(null);
  const isDraggingRef = useRef(false);
  const [dragSelection, setDragSelection] = useState(null);
  const [isDragging, setIsDragging] = useState(false); // State version for render

  // Track if initial zoom has been applied
  const [initialZoomApplied, setInitialZoomApplied] = useState(false);

  // Mobile tooltip management
  const [isMobile, setIsMobile] = useState(false);
  const [showTooltip, setShowTooltip] = useState(true);
  const [tooltipTimeout, setTooltipTimeout] = useState(null);
  const [tooltipPinned, setTooltipPinned] = useState(false);

  // Track if user has interacted with the chart to hide instructions
  const [hasInteracted, setHasInteracted] = useState(false);

  // Calculate the maximum value from visible data for specific metrics only
  const getMaxValue = useCallback(() => {
    if (!data || data.length === 0) return null;

    let max = -Infinity;
    let targetDataKey = null;

    // Determine which data key to use for peak calculation
    // Priority: totalValue (for portfolios) > price (for funds)
    const priorityKeys = ['totalValue', 'price'];
    for (const key of priorityKeys) {
      const lineExists = lines.some((line) => line.dataKey === key);
      if (lineExists) {
        targetDataKey = key;
        break;
      }
    }

    // If no priority key found, don't show peak line
    if (!targetDataKey) return null;

    // Determine data range to analyze based on zoom
    let dataToAnalyze = data;
    if (zoomState.isZoomed && zoomState.xDomain) {
      const [startIndex, endIndex] = zoomState.xDomain;
      dataToAnalyze = data.slice(startIndex, endIndex + 1);
    }

    // Check only the target data key for max value
    dataToAnalyze.forEach((point) => {
      const value = point[targetDataKey];
      if (value !== null && value !== undefined) {
        max = Math.max(max, value);
      }
    });

    return max === -Infinity ? null : max;
  }, [data, lines, zoomState]);

  // Calculate Y-axis domain based on data and zoom state
  const calculateDomain = useCallback(() => {
    if (!data || data.length === 0) return [0, 0];

    // If zoomed, use the stored domain
    if (zoomState.isZoomed && zoomState.yDomain) {
      return zoomState.yDomain;
    }

    let min = Infinity;
    let max = -Infinity;

    // Determine data range to analyze based on zoom
    let dataToAnalyze = data;
    if (zoomState.isZoomed && zoomState.xDomain) {
      const [startIndex, endIndex] = zoomState.xDomain;
      dataToAnalyze = data.slice(startIndex, endIndex + 1);
    }

    // Check all lines and find min/max values
    dataToAnalyze.forEach((point) => {
      lines.forEach((line) => {
        const value = point[line.dataKey];
        if (value !== null && value !== undefined) {
          min = Math.min(min, value);
          max = Math.max(max, value);
        }
      });
    });

    // Add 10% padding to the top and bottom (increased from 5% to prevent peak line cutoff)
    const padding = (max - min) * 0.1;
    return [
      Math.max(0, Math.floor(min - padding)), // Round down and don't go below 0
      Math.ceil(max + padding), // Round up
    ];
  }, [data, lines, zoomState]);

  // Format Y-axis ticks based on value range with special formatting for max value
  const formatYAxis = (value) => {
    const domain = calculateDomain();
    const maxValue = domain[1];
    const actualMaxValue = getMaxValue();

    // Round the value to whole numbers
    const roundedValue = Math.round(value);

    // Check if this tick is close to the actual maximum value (within 1% tolerance)
    const isMaxValue =
      actualMaxValue && Math.abs(roundedValue - actualMaxValue) / actualMaxValue < 0.01;

    let formattedValue;
    if (maxValue >= 1000000) {
      formattedValue = formatCurrency(roundedValue / 1000000, 0) + 'M';
    } else if (maxValue >= 1000) {
      formattedValue = formatCurrency(roundedValue / 1000, 0) + 'k';
    } else {
      formattedValue = formatCurrency(roundedValue, 0);
    }

    // Add special notation for the maximum value
    return isMaxValue ? `${formattedValue} ★` : formattedValue;
  };

  // Format tooltip values with full precision
  const formatTooltip = useCallback(
    (value) => {
      if (!value && value !== 0) return 'N/A';
      return formatCurrency(value);
    },
    [formatCurrency]
  );

  // Zoom reset function - defined early to avoid circular dependencies
  const handleZoomReset = useCallback(() => {
    const newZoomState = {
      isZoomed: false,
      zoomLevel: 1,
      xDomain: null,
      yDomain: null,
      panOffset: { x: 0, y: 0 },
    };

    setZoomState(newZoomState);

    // Notify parent component of zoom change
    if (onZoomChange) {
      onZoomChange(newZoomState);
    }
  }, [onZoomChange]);

  // Zoom functionality
  const handleZoomIn = useCallback(() => {
    if (!data || data.length === 0) return;

    // Mark as interacted
    setHasInteracted(true);

    const newZoomLevel = Math.min(zoomState.zoomLevel * 1.5, 10);
    const dataLength = data.length;

    let newXDomain;
    if (zoomState.xDomain) {
      const [currentStart, currentEnd] = zoomState.xDomain;
      const currentRange = currentEnd - currentStart;
      const newRange = Math.max(Math.floor(currentRange / 1.5), 5);
      const center = Math.floor((currentStart + currentEnd) / 2);
      const newStart = Math.max(0, center - Math.floor(newRange / 2));
      const newEnd = Math.min(dataLength - 1, newStart + newRange);
      newXDomain = [newStart, newEnd];
    } else {
      const range = Math.max(Math.floor(dataLength / newZoomLevel), 5);
      const start = Math.floor((dataLength - range) / 2);
      newXDomain = [start, start + range];
    }

    const newZoomState = {
      ...zoomState,
      isZoomed: true,
      zoomLevel: newZoomLevel,
      xDomain: newXDomain,
      yDomain: null,
    };

    setZoomState(newZoomState);

    // Notify parent component of zoom change
    if (onZoomChange) {
      onZoomChange(newZoomState);
    }
  }, [data, zoomState, onZoomChange]);

  const handleZoomOut = useCallback(() => {
    if (!data || data.length === 0 || zoomState.zoomLevel <= 1) return;

    // Mark as interacted
    setHasInteracted(true);

    const newZoomLevel = Math.max(zoomState.zoomLevel / 1.5, 1);

    if (newZoomLevel <= 1) {
      handleZoomReset();
      return;
    }

    const dataLength = data.length;
    const [currentStart, currentEnd] = zoomState.xDomain || [0, dataLength - 1];
    const currentRange = currentEnd - currentStart;
    const newRange = Math.min(Math.floor(currentRange * 1.5), dataLength);
    const center = Math.floor((currentStart + currentEnd) / 2);
    const newStart = Math.max(0, center - Math.floor(newRange / 2));
    const newEnd = Math.min(dataLength - 1, newStart + newRange);

    const newZoomState = {
      ...zoomState,
      zoomLevel: newZoomLevel,
      xDomain: [newStart, newEnd],
      yDomain: null,
    };

    setZoomState(newZoomState);

    // Notify parent component of zoom change
    if (onZoomChange) {
      onZoomChange(newZoomState);
    }
  }, [data, zoomState, onZoomChange, handleZoomReset]); // Now includes handleZoomReset

  const handleZoomToPeriod = useCallback(
    (days) => {
      if (!data || data.length === 0) return;

      // Mark as interacted
      setHasInteracted(true);

      const dataLength = data.length;
      const targetDataPoints = Math.min(days, dataLength);

      // Start from the end (most recent data) and go back
      const startIndex = Math.max(0, dataLength - targetDataPoints);
      const endIndex = dataLength - 1;

      if (startIndex < endIndex) {
        const range = endIndex - startIndex + 1;
        const newZoomLevel = dataLength / range;

        const newZoomState = {
          isZoomed: true,
          zoomLevel: newZoomLevel,
          xDomain: [startIndex, endIndex],
          yDomain: null,
          panOffset: { x: 0, y: 0 },
        };

        setZoomState(newZoomState);

        // Don't notify parent component for period buttons to avoid triggering data loading
        // Period buttons (1Y, 3M, 1M) should only adjust the view, not load more data
      }
    },
    [data]
  );

  const handlePan = useCallback(
    (deltaX) => {
      if (!zoomState.isZoomed || !data || data.length === 0) return;

      const [currentStart, currentEnd] = zoomState.xDomain || [0, data.length - 1];
      const range = currentEnd - currentStart;
      const panSensitivity = range / 200;

      const panAmount = Math.floor(deltaX * panSensitivity);
      let newStart = currentStart - panAmount;
      let newEnd = currentEnd - panAmount;

      if (newStart < 0) {
        newStart = 0;
        newEnd = range;
      } else if (newEnd >= data.length) {
        newEnd = data.length - 1;
        newStart = newEnd - range;
      }

      setZoomState((prev) => ({
        ...prev,
        xDomain: [newStart, newEnd],
        yDomain: null,
      }));
    },
    [data, zoomState]
  );

  // Mouse wheel zoom
  const handleWheel = useCallback(
    (e) => {
      if (!e.ctrlKey && !e.metaKey) return;

      e.preventDefault();

      if (e.deltaY < 0) {
        handleZoomIn();
      } else {
        handleZoomOut();
      }
    },
    [handleZoomIn, handleZoomOut]
  );

  // Detect if device is mobile
  useEffect(() => {
    const checkIsMobile = () => {
      const isMobileDevice =
        /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
          navigator.userAgent
        ) || window.innerWidth <= 768;
      setIsMobile(isMobileDevice);
    };

    checkIsMobile();
    window.addEventListener('resize', checkIsMobile);
    return () => window.removeEventListener('resize', checkIsMobile);
  }, []);

  // Auto-hide tooltip on mobile after delay (only if not pinned)
  const scheduleTooltipHide = useCallback(() => {
    if (tooltipTimeout) {
      clearTimeout(tooltipTimeout);
    }

    // Don't auto-hide if tooltip is pinned
    if (tooltipPinned) {
      return;
    }

    const timeout = setTimeout(() => {
      setShowTooltip(false);
    }, 2000); // Hide after 2 seconds

    setTooltipTimeout(timeout);
  }, [tooltipTimeout, tooltipPinned]);

  // Show tooltip temporarily on mobile (only if not pinned)
  const showTooltipTemporarily = useCallback(() => {
    if (isMobile && !tooltipPinned) {
      setShowTooltip(true);
      scheduleTooltipHide();
    }
  }, [isMobile, scheduleTooltipHide, tooltipPinned]);

  // Touch handling for mobile
  const handleTouchStart = useCallback(
    (e) => {
      // Mark as interacted on any touch
      setHasInteracted(true);

      if (e.touches.length === 1) {
        touchStartRef.current = {
          x: e.touches[0].clientX,
          y: e.touches[0].clientY,
          time: Date.now(),
        };
        isPanningRef.current = false;

        // Only show tooltip temporarily if not pinned
        if (!tooltipPinned) {
          showTooltipTemporarily();
        }
      } else if (e.touches.length === 2) {
        const distance = Math.sqrt(
          Math.pow(e.touches[0].clientX - e.touches[1].clientX, 2) +
            Math.pow(e.touches[0].clientY - e.touches[1].clientY, 2)
        );
        lastTouchDistanceRef.current = distance;
        isPanningRef.current = false;

        // Hide tooltip during pinch zoom and unpin it
        if (isMobile) {
          setShowTooltip(false);
          setTooltipPinned(false);
        }
      }
    },
    [isMobile, showTooltipTemporarily, tooltipPinned]
  );

  const handleTouchMove = useCallback(
    (e) => {
      if (e.touches.length === 2) {
        e.preventDefault();
        const distance = Math.sqrt(
          Math.pow(e.touches[0].clientX - e.touches[1].clientX, 2) +
            Math.pow(e.touches[0].clientY - e.touches[1].clientY, 2)
        );

        if (lastTouchDistanceRef.current) {
          const scale = distance / lastTouchDistanceRef.current;
          if (scale > 1.1) {
            handleZoomIn();
            lastTouchDistanceRef.current = distance;
          } else if (scale < 0.9) {
            handleZoomOut();
            lastTouchDistanceRef.current = distance;
          }
        }
      } else if (e.touches.length === 1 && touchStartRef.current && zoomState.isZoomed) {
        const deltaX = e.touches[0].clientX - touchStartRef.current.x;

        if (Math.abs(deltaX) > 10) {
          isPanningRef.current = true;
          handlePan(deltaX);
          touchStartRef.current.x = e.touches[0].clientX;

          // Hide tooltip during panning and unpin it
          if (isMobile) {
            setShowTooltip(false);
            setTooltipPinned(false);
          }
        }
      }
    },
    [zoomState.isZoomed, handleZoomIn, handleZoomOut, handlePan, isMobile]
  );

  const handleTouchEnd = useCallback(() => {
    // Check if this was a tap (not a pan/swipe)
    if (touchStartRef.current && !isPanningRef.current && isMobile) {
      const touchDuration = Date.now() - touchStartRef.current.time;
      // Consider it a tap if it was quick (less than 200ms) and didn't involve panning
      if (touchDuration < 200) {
        // Toggle tooltip pinned state on tap
        if (tooltipPinned) {
          setTooltipPinned(false);
          setShowTooltip(false);
          if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            setTooltipTimeout(null);
          }
        } else {
          setTooltipPinned(true);
          setShowTooltip(true);
          // Clear any existing timeout since tooltip is now pinned
          if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            setTooltipTimeout(null);
          }
        }
      }
    }

    touchStartRef.current = null;
    lastTouchDistanceRef.current = null;
    isPanningRef.current = false;
  }, [isMobile, tooltipPinned, tooltipTimeout]);

  // Handle chart click/tap to toggle tooltip on mobile
  const handleChartClick = useCallback(
    (e) => {
      if (isMobile && e) {
        // If tooltip is currently hidden, show it temporarily
        if (!showTooltip) {
          showTooltipTemporarily();
        } else {
          // If tooltip is showing, hide it
          setShowTooltip(false);
          if (tooltipTimeout) {
            clearTimeout(tooltipTimeout);
            setTooltipTimeout(null);
          }
        }
      }
    },
    [isMobile, showTooltip, showTooltipTemporarily, tooltipTimeout]
  );

  // Custom tooltip component that respects mobile visibility state
  const renderCustomTooltip = useCallback(
    ({ active, payload, label }) => {
      // Don't show tooltip on mobile if showTooltip is false
      if (isMobile && !showTooltip) {
        return null;
      }

      // Don't show tooltip if not active or no payload
      if (!active || !payload || !payload.length) {
        return null;
      }

      return (
        <div className="custom-tooltip">
          <p className="tooltip-label">{`Date: ${label}`}</p>
          {payload.map((entry, index) => (
            <p key={index} className="tooltip-entry" style={{ color: entry.color }}>
              {`${entry.name}: ${formatTooltip(entry.value)}`}
            </p>
          ))}
        </div>
      );
    },
    [isMobile, showTooltip, formatTooltip]
  );

  const getVisibleData = useCallback(() => {
    if (!zoomState.isZoomed || !zoomState.xDomain || !data) {
      return data;
    }

    const [start, end] = zoomState.xDomain;
    return data.slice(start, end + 1);
  }, [data, zoomState]);

  // Mouse drag selection for desktop zoom
  // Define handleMouseUp and handleMouseMove using refs to avoid circular dependencies
  const handleMouseMoveRef = useRef();
  const handleMouseUpRef = useRef();

  // Update refs in an effect to avoid updating refs during render
  useEffect(() => {
    handleMouseMoveRef.current = (e) => {
      if (!dragStartRef.current) return;

      const rect = chartRef.current?.getBoundingClientRect();
      if (!rect) return;

      const currentX = e.clientX - rect.left;
      const startX = dragStartRef.current.x;

      // Only start dragging if moved more than 5 pixels
      if (!isDraggingRef.current && Math.abs(currentX - startX) > 5) {
        isDraggingRef.current = true;
        setIsDragging(true); // Update state for rendering
      }

      if (isDraggingRef.current) {
        // Less restrictive constraints - allow dragging to the very edge
        const yAxisWidth = 80;
        const minX = yAxisWidth;
        const maxX = rect.width - 10; // Small margin to prevent going off-screen

        const constrainedStartX = Math.max(minX, Math.min(maxX, startX));
        const constrainedCurrentX = Math.max(minX, Math.min(maxX, currentX));

        setDragSelection({
          startX: Math.min(constrainedStartX, constrainedCurrentX),
          endX: Math.max(constrainedStartX, constrainedCurrentX),
          width: Math.abs(constrainedCurrentX - constrainedStartX),
        });
      }
    };

    handleMouseUpRef.current = (e) => {
      if (!dragStartRef.current) return;

      const rect = chartRef.current?.getBoundingClientRect();
      if (!rect) return;

      const currentX = e.clientX - rect.left;
      const startX = dragStartRef.current.x;
      const dragDistance = Math.abs(currentX - startX);

      // Clean up event listeners
      const mouseMoveHandler = handleMouseMoveRef.current;
      const mouseUpHandler = handleMouseUpRef.current;
      if (mouseMoveHandler) {
        document.removeEventListener('mousemove', mouseMoveHandler);
      }
      if (mouseUpHandler) {
        document.removeEventListener('mouseup', mouseUpHandler);
      }

      // If dragged enough distance, zoom to selection
      if (isDraggingRef.current && dragDistance > 20 && data && data.length > 0) {
        // Account for Y-axis width and Recharts internal padding
        // These values were empirically tuned for best accuracy in center selections
        // Note: Edge selections (far left/right) may be slightly less accurate due to
        // Recharts' internal margin handling
        const yAxisWidth = 80;
        const leftPadding = 5; // Recharts internal left margin
        const rightPadding = 7; // Recharts internal right margin
        const chartStartX = yAxisWidth + leftPadding;
        const chartEndX = rect.width - rightPadding;
        const chartWidth = chartEndX - chartStartX;

        // Convert pixel positions to ratios
        const leftX = Math.min(startX, currentX);
        const rightX = Math.max(startX, currentX);

        // Calculate ratios based on actual chart area
        const startRatio = Math.max(0, Math.min(1, (leftX - chartStartX) / chartWidth));
        const endRatio = Math.max(0, Math.min(1, (rightX - chartStartX) / chartWidth));

        // Convert ratios to data indices
        const dataLength = data.length;
        // Add adjustments to align better across different zoom levels
        // At higher zoom (more pixels per day), we need slightly different alignment
        const pixelsPerDay = chartWidth / (dataLength - 1);
        const startAdjustment = pixelsPerDay > 2 ? 0.5 : 0; // Adjust when zoomed in
        const endAdjustment = pixelsPerDay > 2 ? 0.6 : 0; // Slightly more for end

        const startIndex = Math.max(0, Math.floor(startRatio * (dataLength - 1) + startAdjustment));
        const endIndex = Math.min(
          dataLength - 1,
          Math.floor(endRatio * (dataLength - 1) + endAdjustment)
        );

        // Ensure we have a meaningful selection
        if (endIndex > startIndex) {
          const range = endIndex - startIndex + 1;
          const newZoomLevel = Math.min(dataLength / range, 10);

          setZoomState((prev) => ({
            ...prev,
            isZoomed: true,
            zoomLevel: newZoomLevel,
            xDomain: [startIndex, endIndex],
            yDomain: null,
          }));
        }
      }

      // Reset drag state
      setDragSelection(null);
      setIsDragging(false); // Update state for rendering
      dragStartRef.current = null;
      isDraggingRef.current = false;
    };
  }, [data]); // Re-create handlers when data changes

  const handleMouseDown = useCallback((e) => {
    if (e.button !== 0) return; // Only left mouse button
    if (e.ctrlKey || e.metaKey) return; // Don't interfere with wheel zoom

    const rect = chartRef.current?.getBoundingClientRect();
    if (!rect) return;

    const x = e.clientX - rect.left;

    // More lenient boundary check - allow starting anywhere in the chart container
    const yAxisWidth = 80;
    if (x < yAxisWidth) return; // Only prevent starting on Y-axis

    // Mark as interacted on mouse interaction
    setHasInteracted(true);

    dragStartRef.current = { x, startTime: Date.now() };
    isDraggingRef.current = false;

    // Add global mouse event listeners using current ref values
    const mouseMoveHandler = handleMouseMoveRef.current;
    const mouseUpHandler = handleMouseUpRef.current;
    if (mouseMoveHandler) {
      document.addEventListener('mousemove', mouseMoveHandler);
    }
    if (mouseUpHandler) {
      document.addEventListener('mouseup', mouseUpHandler);
    }
  }, []); // No dependencies needed - uses refs

  // Apply initial zoom when data is loaded - use a flag to avoid ESLint warning
  const shouldApplyInitialZoom = useRef(false);

  useEffect(() => {
    if (data && data.length > 0 && defaultZoomDays && !initialZoomApplied) {
      shouldApplyInitialZoom.current = true;

      // Check if we should apply initial zoom
      const shouldApplyZoom = !(totalDataRange && totalDataRange.isFullDataset);

      // Schedule state updates for next render cycle to avoid cascading renders
      queueMicrotask(() => {
        if (!shouldApplyInitialZoom.current) return;

        if (shouldApplyZoom) {
          const dataLength = data.length;

          // Calculate how many data points represent the default zoom period
          const targetDataPoints = Math.min(defaultZoomDays, dataLength);

          // Start from the end (most recent data) and go back
          const startIndex = Math.max(0, dataLength - targetDataPoints);
          const endIndex = dataLength - 1;

          if (startIndex < endIndex) {
            const range = endIndex - startIndex + 1;
            const newZoomLevel = dataLength / range;

            // Batch state updates together to minimize re-renders
            setZoomState({
              isZoomed: true,
              zoomLevel: newZoomLevel,
              xDomain: [startIndex, endIndex],
              yDomain: null,
              panOffset: { x: 0, y: 0 },
            });
          }
        }

        // Always mark as applied
        setInitialZoomApplied(true);
        shouldApplyInitialZoom.current = false;
      });
    }
  }, [data, defaultZoomDays, initialZoomApplied, totalDataRange]);

  useEffect(() => {
    const chartElement = chartRef.current;
    if (chartElement) {
      chartElement.addEventListener('wheel', handleWheel, { passive: false });
      chartElement.addEventListener('mousedown', handleMouseDown);

      return () => {
        chartElement.removeEventListener('wheel', handleWheel);
        chartElement.removeEventListener('mousedown', handleMouseDown);
        // Clean up any remaining global listeners using refs
        const mouseMoveHandler = handleMouseMoveRef.current;
        const mouseUpHandler = handleMouseUpRef.current;
        if (mouseMoveHandler) {
          document.removeEventListener('mousemove', mouseMoveHandler);
        }
        if (mouseUpHandler) {
          document.removeEventListener('mouseup', mouseUpHandler);
        }
      };
    }
  }, [handleWheel, handleMouseDown]);

  // Cleanup tooltip timeout on unmount
  useEffect(() => {
    return () => {
      if (tooltipTimeout) {
        clearTimeout(tooltipTimeout);
      }
    };
  }, [tooltipTimeout]);

  // Prevent body scroll when fullscreen is active
  useEffect(() => {
    if (isFullScreen) {
      // Save current body overflow and position
      const originalOverflow = document.body.style.overflow;
      const originalPosition = document.body.style.position;

      // Prevent scrolling
      document.body.style.overflow = 'hidden';
      document.body.style.position = 'fixed';
      document.body.style.width = '100%';

      return () => {
        // Restore original values
        document.body.style.overflow = originalOverflow;
        document.body.style.position = originalPosition;
        document.body.style.width = '';
      };
    }
  }, [isFullScreen]);

  // Handle orientation changes in fullscreen to fix chart overflow
  useEffect(() => {
    if (isFullScreen) {
      const handleOrientationChange = () => {
        // Force a small state update to trigger re-render and recalculate chart dimensions
        // This fixes the chart overflow issue when rotating device
        setZoomState((prev) => ({ ...prev }));
      };

      // Listen for orientation changes
      window.addEventListener('orientationchange', handleOrientationChange);
      // Also listen for resize as a fallback
      window.addEventListener('resize', handleOrientationChange);

      return () => {
        window.removeEventListener('orientationchange', handleOrientationChange);
        window.removeEventListener('resize', handleOrientationChange);
      };
    }
  }, [isFullScreen]);

  // Render chart content (reusable for normal and fullscreen modes)
  const renderChartContent = (isFullScreenMode = false) => {
    const chartHeight = isFullScreenMode ? '100%' : height;

    const zoomButtons = (
      <>
        <button
          className="zoom-button"
          onClick={handleZoomIn}
          disabled={zoomState.zoomLevel >= 10}
          title="Zoom In (Ctrl + Mouse Wheel)"
        >
          🔍+
        </button>
        <button
          className="zoom-button"
          onClick={handleZoomOut}
          disabled={zoomState.zoomLevel <= 1}
          title="Zoom Out (Ctrl + Mouse Wheel)"
        >
          🔍-
        </button>
        <button
          className="zoom-button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            handleZoomReset();
            if (onLoadAllData && (!totalDataRange || !totalDataRange.isFullDataset)) {
              onLoadAllData();
            }
          }}
          title="Show All Data"
          type="button"
        >
          All
        </button>
        <button className="zoom-button" onClick={() => handleZoomToPeriod(365)} title="Last Year">
          1Y
        </button>
        <button
          className="zoom-button"
          onClick={() => handleZoomToPeriod(90)}
          title="Last 3 Months"
        >
          3M
        </button>
        <button className="zoom-button" onClick={() => handleZoomToPeriod(30)} title="Last Month">
          1M
        </button>
      </>
    );

    return (
      <>
        {/* Fullscreen: Toggle buttons for controls */}
        {isFullScreenMode && visibleMetrics && (
          <>
            {/* Toggle buttons - always visible */}
            <div className="fullscreen-toggle-buttons">
              <button
                className={`control-toggle-button ${showMetricControls ? 'active' : ''}`}
                onClick={() => setShowMetricControls(!showMetricControls)}
                title="Toggle Metrics"
              >
                ⚙️
              </button>
              <button
                className={`control-toggle-button ${showZoomControls ? 'active' : ''}`}
                onClick={() => setShowZoomControls(!showZoomControls)}
                title="Toggle Zoom"
              >
                🔍
              </button>
            </div>

            {/* Metric Controls - shown when toggled */}
            {showMetricControls && (
              <div className="chart-controls fullscreen-metric-panel">
                <button
                  className={`transaction-button ${visibleMetrics.value ? 'active' : ''}`}
                  onClick={() => setVisibleMetrics((prev) => ({ ...prev, value: !prev.value }))}
                >
                  Value
                </button>
                <button
                  className={`transaction-button ${visibleMetrics.cost ? 'active' : ''}`}
                  onClick={() => setVisibleMetrics((prev) => ({ ...prev, cost: !prev.cost }))}
                >
                  Cost
                </button>
                <button
                  className={`transaction-button ${visibleMetrics.realizedGain ? 'active' : ''}`}
                  onClick={() =>
                    setVisibleMetrics((prev) => ({
                      ...prev,
                      realizedGain: !prev.realizedGain,
                    }))
                  }
                >
                  Realized Gain/Loss
                </button>
                <button
                  className={`transaction-button ${visibleMetrics.unrealizedGain ? 'active' : ''}`}
                  onClick={() =>
                    setVisibleMetrics((prev) => ({
                      ...prev,
                      unrealizedGain: !prev.unrealizedGain,
                    }))
                  }
                >
                  Unrealized Gain/Loss
                </button>
                <button
                  className={`transaction-button ${visibleMetrics.totalGain ? 'active' : ''}`}
                  onClick={() =>
                    setVisibleMetrics((prev) => ({
                      ...prev,
                      totalGain: !prev.totalGain,
                    }))
                  }
                >
                  Total Gain/Loss
                </button>
              </div>
            )}

            {/* Zoom Controls - shown when toggled */}
            {showZoomControls && (
              <div className="chart-controls fullscreen-zoom-panel">{zoomButtons}</div>
            )}
          </>
        )}

        {visibleMetrics && !isFullScreenMode && (
          <div className="chart-controls">
            <div className="metric-toggles">
              <button
                className={`transaction-button ${visibleMetrics.value ? 'active' : ''}`}
                onClick={() => setVisibleMetrics((prev) => ({ ...prev, value: !prev.value }))}
              >
                Value
              </button>
              <button
                className={`transaction-button ${visibleMetrics.cost ? 'active' : ''}`}
                onClick={() => setVisibleMetrics((prev) => ({ ...prev, cost: !prev.cost }))}
              >
                Cost
              </button>
              <button
                className={`transaction-button ${visibleMetrics.realizedGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    realizedGain: !prev.realizedGain,
                  }))
                }
              >
                Realized Gain/Loss
              </button>
              <button
                className={`transaction-button ${visibleMetrics.unrealizedGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    unrealizedGain: !prev.unrealizedGain,
                  }))
                }
              >
                Unrealized Gain/Loss
              </button>
              <button
                className={`transaction-button ${visibleMetrics.totalGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    totalGain: !prev.totalGain,
                  }))
                }
              >
                Total Gain/Loss
              </button>
            </div>
          </div>
        )}

        {/* Zoom Controls - separate in normal mode */}
        {!isFullScreenMode && <div className="zoom-controls">{zoomButtons}</div>}

        {showTimeRangeButtons && (
          <div className="time-range-buttons">
            <button
              className={`time-range-button ${timeRange === '1M' ? 'active' : ''}`}
              onClick={() => onTimeRangeChange('1M')}
            >
              Last Month
            </button>
            <button
              className={`time-range-button ${timeRange === 'ALL' ? 'active' : ''}`}
              onClick={() => onTimeRangeChange('ALL')}
            >
              All Time
            </button>
          </div>
        )}

        <div
          className={`chart-container ${isDragging ? 'dragging' : ''}`}
          ref={chartRef}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        >
          <ResponsiveContainer width="100%" height={chartHeight}>
            <LineChart data={getVisibleData()} onClick={handleChartClick}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} interval="preserveStartEnd" />
              <YAxis
                domain={calculateDomain()}
                tick={{ fontSize: 12 }}
                tickFormatter={formatYAxis}
                width={80}
              />
              <Tooltip
                content={renderCustomTooltip}
                cursor={{ stroke: '#1976d2', strokeWidth: 1 }}
              />
              <Legend />
              {/* Reference line for maximum value */}
              {getMaxValue() && (
                <ReferenceLine
                  y={getMaxValue()}
                  stroke="#ff6b35"
                  strokeDasharray="8 8"
                  strokeWidth={2}
                  label={{
                    value: `Peak: ${formatCurrency(getMaxValue())}`,
                    position: 'top',
                    offset: 10,
                    style: {
                      textAnchor: 'middle',
                      fontSize: '12px',
                      fontWeight: 'bold',
                      fill: '#ff6b35',
                    },
                  }}
                />
              )}
              {lines.map((line) => (
                <Line
                  key={line.dataKey}
                  type="monotone"
                  dataKey={line.dataKey}
                  name={line.name}
                  stroke={line.color}
                  dot={false}
                  strokeWidth={line.strokeWidth || 2}
                  strokeDasharray={line.strokeDasharray}
                  opacity={line.opacity}
                  connectNulls={true}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>

          {/* Drag selection overlay */}
          {dragSelection && (
            <div
              className="drag-selection-overlay"
              style={{
                left: `${dragSelection.startX}px`,
                width: `${dragSelection.width}px`,
                top: '0px',
                height: '100%',
                position: 'absolute',
                backgroundColor: 'rgba(25, 118, 210, 0.2)',
                border: '2px solid #1976d2',
                borderRadius: '4px',
                pointerEvents: 'none',
                zIndex: 10,
              }}
            />
          )}

          {!hasInteracted && zoomState.isZoomed && !isFullScreenMode && (
            <div className="zoom-instructions">
              <p>
                💡 Desktop: Drag to select area, Hold Ctrl/Cmd + scroll to zoom | Mobile: Pinch to
                zoom, swipe to pan, tap chart to toggle tooltip
              </p>
            </div>
          )}

          {!hasInteracted && !zoomState.isZoomed && !isFullScreenMode && (
            <div className="zoom-instructions">
              <p>
                💡 Desktop: Click and drag to zoom to selection, Hold Ctrl/Cmd + scroll to zoom |
                Mobile: Pinch to zoom, tap chart to show values
              </p>
            </div>
          )}
        </div>
      </>
    );
  };

  return (
    <>
      {/* Normal Mode */}
      <div className={`chart-wrapper ${isMobile ? 'mobile-chart-wrapper' : ''}`}>
        {/* Mobile: Minimalist - just show chart, small fullscreen button in corner */}
        {isMobile && visibleMetrics && (
          <button className="mobile-minimal-fullscreen-btn" onClick={() => setIsFullScreen(true)}>
            ⛶
          </button>
        )}

        {/* Desktop: show all metric toggles normally */}
        {!isMobile && visibleMetrics && (
          <div className="chart-controls">
            <div className="metric-toggles">
              <button
                className={`transaction-button ${visibleMetrics.value ? 'active' : ''}`}
                onClick={() => setVisibleMetrics((prev) => ({ ...prev, value: !prev.value }))}
              >
                Value
              </button>
              <button
                className={`transaction-button ${visibleMetrics.cost ? 'active' : ''}`}
                onClick={() => setVisibleMetrics((prev) => ({ ...prev, cost: !prev.cost }))}
              >
                Cost
              </button>
              <button
                className={`transaction-button ${visibleMetrics.realizedGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    realizedGain: !prev.realizedGain,
                  }))
                }
              >
                Realized Gain/Loss
              </button>
              <button
                className={`transaction-button ${visibleMetrics.unrealizedGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    unrealizedGain: !prev.unrealizedGain,
                  }))
                }
              >
                Unrealized Gain/Loss
              </button>
              <button
                className={`transaction-button ${visibleMetrics.totalGain ? 'active' : ''}`}
                onClick={() =>
                  setVisibleMetrics((prev) => ({
                    ...prev,
                    totalGain: !prev.totalGain,
                  }))
                }
              >
                Total Gain/Loss
              </button>
            </div>
          </div>
        )}

        {/* Desktop: Show zoom controls */}
        {!isMobile && (
          <div className="zoom-controls">
            <button
              className="zoom-button"
              onClick={handleZoomIn}
              disabled={zoomState.zoomLevel >= 10}
              title="Zoom In (Ctrl + Mouse Wheel)"
            >
              🔍+
            </button>
            <button
              className="zoom-button"
              onClick={handleZoomOut}
              disabled={zoomState.zoomLevel <= 1}
              title="Zoom Out (Ctrl + Mouse Wheel)"
            >
              🔍-
            </button>
            <button
              className="zoom-button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                handleZoomReset();
                if (onLoadAllData && (!totalDataRange || !totalDataRange.isFullDataset)) {
                  onLoadAllData();
                }
              }}
              title="Show All Data"
              type="button"
            >
              All
            </button>
            <button
              className="zoom-button"
              onClick={() => handleZoomToPeriod(365)}
              title="Last Year"
            >
              1Y
            </button>
            <button
              className="zoom-button"
              onClick={() => handleZoomToPeriod(90)}
              title="Last 3 Months"
            >
              3M
            </button>
            <button
              className="zoom-button"
              onClick={() => handleZoomToPeriod(30)}
              title="Last Month"
            >
              1M
            </button>
          </div>
        )}

        {showTimeRangeButtons && (
          <div className="time-range-buttons">
            <button
              className={`time-range-button ${timeRange === '1M' ? 'active' : ''}`}
              onClick={() => onTimeRangeChange('1M')}
            >
              Last Month
            </button>
            <button
              className={`time-range-button ${timeRange === 'ALL' ? 'active' : ''}`}
              onClick={() => onTimeRangeChange('ALL')}
            >
              All Time
            </button>
          </div>
        )}

        <div
          className={`chart-container ${isDragging ? 'dragging' : ''}`}
          ref={chartRef}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        >
          <ResponsiveContainer width="100%" height={height}>
            <LineChart data={getVisibleData()} onClick={handleChartClick}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} interval="preserveStartEnd" />
              <YAxis
                domain={calculateDomain()}
                tick={{ fontSize: 12 }}
                tickFormatter={formatYAxis}
                width={80}
              />
              <Tooltip
                content={renderCustomTooltip}
                cursor={{ stroke: '#1976d2', strokeWidth: 1 }}
              />
              <Legend />
              {/* Reference line for maximum value */}
              {getMaxValue() && (
                <ReferenceLine
                  y={getMaxValue()}
                  stroke="#ff6b35"
                  strokeDasharray="8 8"
                  strokeWidth={2}
                  label={{
                    value: `Peak: ${formatCurrency(getMaxValue())}`,
                    position: 'top',
                    offset: 10,
                    style: {
                      textAnchor: 'middle',
                      fontSize: '12px',
                      fontWeight: 'bold',
                      fill: '#ff6b35',
                    },
                  }}
                />
              )}
              {lines.map((line) => (
                <Line
                  key={line.dataKey}
                  type="monotone"
                  dataKey={line.dataKey}
                  name={line.name}
                  stroke={line.color}
                  dot={false}
                  strokeWidth={line.strokeWidth || 2}
                  strokeDasharray={line.strokeDasharray}
                  opacity={line.opacity}
                  connectNulls={true}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>

          {/* Drag selection overlay */}
          {dragSelection && (
            <div
              className="drag-selection-overlay"
              style={{
                left: `${dragSelection.startX}px`,
                width: `${dragSelection.width}px`,
                top: '0px',
                height: '100%',
                position: 'absolute',
                backgroundColor: 'rgba(25, 118, 210, 0.2)',
                border: '2px solid #1976d2',
                borderRadius: '4px',
                pointerEvents: 'none',
                zIndex: 10,
              }}
            />
          )}

          {!hasInteracted && zoomState.isZoomed && (
            <div className="zoom-instructions">
              <p>
                💡 Desktop: Drag to select area, Hold Ctrl/Cmd + scroll to zoom | Mobile: Pinch to
                zoom, swipe to pan, tap chart to toggle tooltip
              </p>
            </div>
          )}

          {!hasInteracted && !zoomState.isZoomed && (
            <div className="zoom-instructions">
              <p>
                💡 Desktop: Click and drag to zoom to selection, Hold Ctrl/Cmd + scroll to zoom |
                Mobile: Pinch to zoom, tap chart to show values
              </p>
            </div>
          )}
        </div>
      </div>

      {/* True Full-Screen Experience (Mobile Only) */}
      {isMobile && isFullScreen && (
        <div className="chart-fullscreen-overlay">
          {/* Close button overlay */}
          <button
            className="chart-fullscreen-close"
            onClick={() => setIsFullScreen(false)}
            aria-label="Close fullscreen"
          >
            ×
          </button>

          {/* Fullscreen chart content */}
          <div className="chart-fullscreen-container">{renderChartContent(true)}</div>
        </div>
      )}
    </>
  );
};

export default ValueChart;
