import ApexCharts from 'apexcharts';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import './ValueChart.css';
import { useFormat } from '../context/FormatContext';
import { useTheme } from '../context/ThemeContext';

/**
 * ValueChart component - Interactive chart with zoom, pan, and mobile support
 *
 * Built on ApexCharts (vanilla, to support React 19 Strict Mode), supports:
 * - Desktop: Selection zoom, scroll wheel zoom, pan
 * - Mobile: Pinch-to-zoom, swipe-to-pan (native ApexCharts touch support)
 * - Fullscreen mode for mobile devices (CSS-based, preserves chart state)
 * - Dynamic metric toggling (value, cost, gains/losses)
 * - Time range presets (1M, 3M, 1Y, All)
 * - Peak value reference line annotation
 * - Auto-loading of historical data via onZoomChange callback
 * - Responsive design with orientation change support
 *
 * @param {Object} props - Component props object
 * @param {Array} props.data - Chart data array with date and value properties
 * @param {number} [props.height=400] - Chart height in pixels (desktop mode)
 * @param {Array} [props.lines=[]] - Line configurations with dataKey, name, color, strokeWidth, etc.
 * @param {string|boolean} [props.timeRange=false] - Current time range selection ('1M', 'ALL', etc.)
 * @param {Function} props.onTimeRangeChange - Callback when time range changes
 * @param {boolean} [props.showTimeRangeButtons=false] - Whether to show time range selection buttons
 * @param {Object|boolean} [props.visibleMetrics=false] - Object controlling which metrics are visible
 * @param {Function} props.setVisibleMetrics - Callback to update visible metrics
 * @param {number} [props.defaultZoomDays=null] - Number of days to show by default (e.g., 365 for last year)
 * @param {Function} props.onZoomChange - Callback for zoom state changes (index-based, for useChartData)
 * @param {Function} props.onLoadAllData - Callback to load all available historical data
 * @param {Object} [props.totalDataRange=null] - Information about the total data range available
 * @returns {JSX.Element} Interactive chart component with zoom and pan capabilities
 *
 * @example
 * <ValueChart
 *   data={chartData}
 *   lines={[
 *     { dataKey: 'totalValue', name: 'Value', color: '#1976d2', strokeWidth: 2 },
 *     { dataKey: 'totalCost', name: 'Cost', color: '#ff6b35', strokeWidth: 2 }
 *   ]}
 *   visibleMetrics={{ value: true, cost: true, realizedGain: false }}
 *   setVisibleMetrics={setVisibleMetrics}
 *   defaultZoomDays={365}
 *   onLoadAllData={handleLoadAllData}
 * />
 */
const ValueChart = ({
  data,
  height = 400,
  lines = [],
  timeRange = false,
  onTimeRangeChange,
  showTimeRangeButtons = false,
  visibleMetrics = false,
  setVisibleMetrics,
  defaultZoomDays = null,
  onZoomChange,
  onLoadAllData,
  totalDataRange = null,
}) => {
  const { formatCurrency } = useFormat();
  const { isDark } = useTheme();

  // containerRef: the DOM element ApexCharts renders into
  // chartInstanceRef: the vanilla ApexCharts instance
  const containerRef = useRef(null);
  const chartInstanceRef = useRef(null);
  const initialZoomApplied = useRef(false);
  const onZoomChangeRef = useRef(onZoomChange);
  const dataRef = useRef(data);

  const [isMobile, setIsMobile] = useState(false);
  const [isFullScreen, setIsFullScreen] = useState(false);
  const [showMetricControls, setShowMetricControls] = useState(false);
  const [showZoomControls, setShowZoomControls] = useState(false);

  // Keep refs fresh to avoid stale closures in chart events
  useEffect(() => {
    onZoomChangeRef.current = onZoomChange;
  }, [onZoomChange]);

  useEffect(() => {
    dataRef.current = data;
  }, [data]);

  // Mobile detection
  useEffect(() => {
    const check = () => {
      setIsMobile(
        /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
          navigator.userAgent,
        ) || window.innerWidth <= 768,
      );
    };
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, []);

  // Prevent body scroll in fullscreen
  useEffect(() => {
    if (isFullScreen) {
      const originalOverflow = document.body.style.overflow;
      const originalPosition = document.body.style.position;
      document.body.style.overflow = 'hidden';
      document.body.style.position = 'fixed';
      document.body.style.width = '100%';
      return () => {
        document.body.style.overflow = originalOverflow;
        document.body.style.position = originalPosition;
        document.body.style.width = '';
      };
    }
  }, [isFullScreen]);

  // Convert data + lines to ApexCharts series format
  const series = useMemo(() => {
    if (!data?.length || !lines?.length) return [];
    return lines.map((line) => ({
      name: line.name,
      data: data.map((point) => [
        new Date(point.date).getTime(),
        point[line.dataKey] != null ? point[line.dataKey] : null,
      ]),
    }));
  }, [data, lines]);

  // Peak value for annotation (only for totalValue or price series)
  const peakValue = useMemo(() => {
    if (!data?.length) return null;
    const priorityKey = ['totalValue', 'price'].find((key) =>
      lines.some((line) => line.dataKey === key),
    );
    if (!priorityKey) return null;
    const values = data.map((p) => p[priorityKey]).filter((v) => v != null);
    if (!values.length) return null;
    const max = Math.max(...values);
    return Number.isFinite(max) ? max : null;
  }, [data, lines]);

  // Y-axis label formatter
  const formatYAxis = useCallback(
    (value) => {
      if (value == null) return '';
      const rounded = Math.round(value);
      if (Math.abs(rounded) >= 1000000) return `${formatCurrency(rounded / 1000000, 0)}M`;
      if (Math.abs(rounded) >= 1000) return `${formatCurrency(rounded / 1000, 0)}k`;
      return formatCurrency(rounded, 0);
    },
    [formatCurrency],
  );

  // Per-series colors (with opacity support via rgba)
  const colors = useMemo(
    () =>
      lines.map((l) => {
        if (l.opacity && l.opacity < 1 && l.color?.startsWith('#') && l.color.length === 7) {
          const r = parseInt(l.color.slice(1, 3), 16);
          const g = parseInt(l.color.slice(3, 5), 16);
          const b = parseInt(l.color.slice(5, 7), 16);
          return `rgba(${r},${g},${b},${l.opacity})`;
        }
        return l.color || '#8884d8';
      }),
    [lines],
  );

  // Build the full options object (memo'd to minimize updateOptions calls)
  const options = useMemo(
    () => ({
      chart: {
        type: 'line',
        height,
        animations: { enabled: false },
        background: 'transparent',
        zoom: {
          enabled: true,
          type: 'x',
          autoScaleYaxis: true,
        },
        toolbar: {
          show: true,
          tools: {
            download: false,
            selection: true,
            zoom: true,
            zoomin: true,
            zoomout: true,
            pan: true,
            reset: false,
          },
        },
        events: {
          zoomed: (_chartContext, { xaxis }) => {
            const handler = onZoomChangeRef.current;
            const currentData = dataRef.current;
            if (!handler || !currentData?.length) return;
            const startIndex = currentData.findIndex(
              (d) => new Date(d.date).getTime() >= xaxis.min,
            );
            let endIndex = currentData.length - 1;
            for (let i = currentData.length - 1; i >= 0; i--) {
              if (new Date(currentData[i].date).getTime() <= xaxis.max) {
                endIndex = i;
                break;
              }
            }
            if (startIndex !== -1 && endIndex > startIndex) {
              handler({ isZoomed: true, xDomain: [startIndex, endIndex] });
            }
          },
        },
      },
      colors,
      stroke: {
        curve: 'smooth',
        width: lines.map((l) => l.strokeWidth || 2),
        dashArray: lines.map((l) => {
          if (!l.strokeDasharray) return 0;
          return parseInt(l.strokeDasharray.split(' ')[0], 10) || 0;
        }),
      },
      xaxis: {
        type: 'datetime',
        labels: {
          datetimeUTC: false,
          datetimeFormatter: {
            year: 'yyyy',
            month: "MMM 'yy",
            day: 'dd MMM',
            hour: 'HH:mm',
          },
        },
      },
      yaxis: {
        labels: { formatter: formatYAxis },
      },
      tooltip: {
        shared: true,
        x: { format: 'yyyy-MM-dd' },
        y: {
          formatter: (val) => (val != null ? formatCurrency(val) : 'N/A'),
        },
      },
      legend: {
        show: true,
        position: 'bottom',
        fontSize: '12px',
      },
      grid: {
        strokeDashArray: 3,
      },
      annotations: {
        yaxis: peakValue
          ? [
              {
                y: peakValue,
                borderColor: '#ff6b35',
                strokeDashArray: 8,
                borderWidth: 2,
                label: {
                  borderColor: '#ff6b35',
                  style: { color: '#fff', background: '#ff6b35', fontSize: '11px' },
                  text: `Peak: ${formatCurrency(peakValue)}`,
                  position: 'right',
                  offsetX: -10,
                },
              },
            ]
          : [],
      },
      noData: {
        text: 'No data available',
        align: 'center',
        verticalAlign: 'middle',
      },
      theme: {
        mode: isDark ? 'dark' : 'light',
      },
    }),
    [colors, lines, height, formatYAxis, formatCurrency, peakValue, isDark],
  );

  // Mount vanilla ApexCharts once; safe cleanup handles React 19 Strict Mode double-invoke
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const instance = new ApexCharts(el, { ...options, series });
    instance.render().catch(() => {});
    chartInstanceRef.current = instance;

    return () => {
      chartInstanceRef.current = null;
      try {
        instance.destroy();
      } catch (_) {
        // Safe: React 19 Strict Mode may call cleanup on an already-destroyed instance
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options, series]); // Intentionally empty: chart is mounted once; series/options use update effects below

  // Update series when data or lines change (after initial mount)
  const isFirstSeriesUpdate = useRef(true);
  useEffect(() => {
    if (isFirstSeriesUpdate.current) {
      isFirstSeriesUpdate.current = false;
      return;
    }
    if (chartInstanceRef.current) {
      chartInstanceRef.current.updateSeries(series, false);
    }
  }, [series]);

  // Update options when visual config changes (after initial mount)
  const isFirstOptionsUpdate = useRef(true);
  useEffect(() => {
    if (isFirstOptionsUpdate.current) {
      isFirstOptionsUpdate.current = false;
      return;
    }
    if (chartInstanceRef.current) {
      chartInstanceRef.current.updateOptions(options, false, false);
    }
  }, [options]);

  // Apply initial zoom once data is available
  useEffect(() => {
    if (
      data?.length &&
      defaultZoomDays &&
      !initialZoomApplied.current &&
      !totalDataRange?.isFullDataset
    ) {
      const instance = chartInstanceRef.current;
      if (!instance) return;
      const lastDate = new Date(data[data.length - 1].date).getTime();
      const startDate = lastDate - defaultZoomDays * 24 * 60 * 60 * 1000;
      instance.updateOptions({ xaxis: { min: startDate, max: lastDate } }, false, false);
      initialZoomApplied.current = true;
    }
  }, [data, defaultZoomDays, totalDataRange]);

  // Resize chart when fullscreen toggles (CSS moves wrapper; chart height must follow)
  useEffect(() => {
    const instance = chartInstanceRef.current;
    if (!instance) return;
    const chartHeight = isFullScreen ? window.innerHeight - 100 : height;
    instance.updateOptions({ chart: { height: chartHeight } }, false, false);
  }, [isFullScreen, height]);

  // Period zoom handlers
  const handleZoomToPeriod = useCallback(
    (days) => {
      if (!data?.length) return;
      const instance = chartInstanceRef.current;
      if (!instance) return;
      const lastDate = new Date(data[data.length - 1].date).getTime();
      const startDate = lastDate - days * 24 * 60 * 60 * 1000;
      instance.updateOptions({ xaxis: { min: startDate, max: lastDate } }, false, true);
    },
    [data],
  );

  const handleZoomReset = useCallback(() => {
    const instance = chartInstanceRef.current;
    if (instance) {
      instance.updateOptions({ xaxis: { min: undefined, max: undefined } }, false, true);
    }
    if (onLoadAllData && (!totalDataRange || !totalDataRange.isFullDataset)) {
      onLoadAllData();
    }
  }, [onLoadAllData, totalDataRange]);

  const renderMetricToggles = (inFullscreen = false) => {
    if (!visibleMetrics) return null;
    const metrics = [
      { key: 'value', label: 'Value' },
      { key: 'cost', label: 'Cost' },
      { key: 'realizedGain', label: 'Realized Gain/Loss' },
      { key: 'unrealizedGain', label: 'Unrealized Gain/Loss' },
      { key: 'totalGain', label: 'Total Gain/Loss' },
    ];
    return (
      <div className={`chart-controls ${inFullscreen ? 'fullscreen-metric-panel' : ''}`}>
        {metrics.map(({ key, label }) => (
          <button
            type="button"
            key={key}
            className={`transaction-button ${visibleMetrics[key] ? 'active' : ''}`}
            onClick={() => setVisibleMetrics((prev) => ({ ...prev, [key]: !prev[key] }))}
          >
            {label}
          </button>
        ))}
      </div>
    );
  };

  const renderZoomButtons = (inFullscreen = false) => (
    <div className={`chart-controls ${inFullscreen ? 'fullscreen-zoom-panel' : 'zoom-controls'}`}>
      <button type="button" className="zoom-button" onClick={handleZoomReset} title="Show All Data">
        All
      </button>
      <button
        type="button"
        className="zoom-button"
        onClick={() => handleZoomToPeriod(365)}
        title="Last Year"
      >
        1Y
      </button>
      <button
        type="button"
        className="zoom-button"
        onClick={() => handleZoomToPeriod(90)}
        title="Last 3 Months"
      >
        3M
      </button>
      <button
        type="button"
        className="zoom-button"
        onClick={() => handleZoomToPeriod(30)}
        title="Last Month"
      >
        1M
      </button>
    </div>
  );

  // The chart container is always in the DOM (never moved between normal/fullscreen).
  // Fullscreen is achieved by CSS: .chart-wrapper.chart-fullscreen-active goes position:fixed.
  return (
    <div
      className={`chart-wrapper ${isMobile ? 'mobile-chart-wrapper' : ''} ${isFullScreen ? 'chart-fullscreen-active' : ''}`}
    >
      {/* Fullscreen controls (overlay on top of chart) */}
      {isMobile && isFullScreen && (
        <>
          <button
            type="button"
            className="chart-fullscreen-close"
            onClick={() => setIsFullScreen(false)}
            aria-label="Close fullscreen"
          >
            ×
          </button>
          <div className="fullscreen-toggle-buttons">
            <button
              type="button"
              className={`control-toggle-button ${showMetricControls ? 'active' : ''}`}
              onClick={() => setShowMetricControls((v) => !v)}
              title="Toggle Metrics"
            >
              ⚙️
            </button>
            <button
              type="button"
              className={`control-toggle-button ${showZoomControls ? 'active' : ''}`}
              onClick={() => setShowZoomControls((v) => !v)}
              title="Toggle Zoom"
            >
              🔍
            </button>
          </div>
          {showMetricControls && renderMetricToggles(true)}
          {showZoomControls && renderZoomButtons(true)}
        </>
      )}

      {/* Mobile fullscreen launch button (normal mode only) */}
      {isMobile && !isFullScreen && visibleMetrics && (
        <button
          type="button"
          className="mobile-minimal-fullscreen-btn"
          onClick={() => setIsFullScreen(true)}
          aria-label="Open fullscreen chart"
        >
          ⛶
        </button>
      )}

      {/* Desktop controls */}
      {!isMobile && renderMetricToggles()}
      {!isMobile && renderZoomButtons()}

      {showTimeRangeButtons && (
        <div className="time-range-buttons">
          <button
            type="button"
            className={`time-range-button ${timeRange === '1M' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('1M')}
          >
            Last Month
          </button>
          <button
            type="button"
            className={`time-range-button ${timeRange === 'ALL' ? 'active' : ''}`}
            onClick={() => onTimeRangeChange('ALL')}
          >
            All Time
          </button>
        </div>
      )}

      {/* Chart container - single DOM element, never unmounted */}
      <div className={`chart-container ${isFullScreen ? 'chart-container-fullscreen' : ''}`}>
        <div ref={containerRef} />
      </div>
    </div>
  );
};

export default ValueChart;
