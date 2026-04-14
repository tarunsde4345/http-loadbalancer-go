const POLL_INTERVAL_MS = 2000;
const HISTORY_LIMIT = 60;
const TRAFFIC_CHART_HEIGHT = 190;
const LATENCY_CHART_HEIGHT = 190;
const X_AXIS_LABEL_INTERVAL_SECONDS = 20;
const TRAFFIC_POINT_RADIUS = 2.4;

const els = {
  uptime: document.getElementById("uptime"),
  totalRequests: document.getElementById("total-requests"),
  totalErrors: document.getElementById("total-errors"),
  aliveBackends: document.getElementById("alive-backends"),
  deadBackends: document.getElementById("dead-backends"),
  refreshPill: document.getElementById("refresh-pill"),
  backendTableBody: document.getElementById("backend-table-body"),
  trafficChart: document.getElementById("traffic-chart"),
  latencyChart: document.getElementById("latency-chart"),
};

const history = [];
let latestMetrics = null;
let previousSnapshot = null;

async function fetchMetrics() {
  const response = await fetch("/metrics", {
    headers: {
      Accept: "application/json",
    },
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  return response.json();
}

function aggregateLatency(backends) {
  if (!backends || backends.length === 0) {
    return { avg: 0, p99: 0 };
  }

  const totalAvg = backends.reduce((sum, item) => sum + Number(item.average_latency_ms || 0), 0);
  const maxP99 = backends.reduce((max, item) => Math.max(max, Number(item.p99_latency_ms || 0)), 0);

  return {
    avg: totalAvg / backends.length,
    p99: maxP99,
  };
}

function pushHistory(metrics) {
  const now = new Date();
  const totalRequests = Number(metrics.total_requests || 0);
  const totalErrors = Number(metrics.total_errors || 0);
  if (!previousSnapshot) {
    previousSnapshot = {
      totalRequests,
      totalErrors,
    };
    return;
  }

  const deltaRequests = Math.max(0, totalRequests - previousSnapshot.totalRequests);
  const deltaErrors = Math.max(0, totalErrors - previousSnapshot.totalErrors);

  history.push({
    label: now.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }),
    requests: deltaRequests,
    errors: deltaErrors,
  });

  if (history.length > HISTORY_LIMIT) {
    history.shift();
  }

  previousSnapshot = {
    totalRequests,
    totalErrors,
  };
}

function renderSummary(metrics) {
  els.uptime.textContent = metrics.uptime_seconds ?? "--";
  els.totalRequests.textContent = String(metrics.total_requests ?? 0);
  els.totalErrors.textContent = String(metrics.total_errors ?? 0);
  els.aliveBackends.textContent = String(metrics.alive_backends ?? 0);
  els.deadBackends.textContent = String(metrics.dead_backends ?? 0);
  els.refreshPill.textContent = `Updated ${new Date().toLocaleTimeString()}`;
}

function normalizeHostLabel(url) {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

function renderBackendTable(backends) {
  if (!backends || backends.length === 0) {
    els.backendTableBody.innerHTML = `
      <tr>
        <td colspan="6" class="table-placeholder">No backend metrics available.</td>
      </tr>
    `;
    return;
  }

  els.backendTableBody.innerHTML = backends
    .map((backend) => {
      const state = String(backend.circuit_breaker_state || "closed").toLowerCase();
      const badgeClass =
        state === "open"
          ? "badge--open"
          : state === "half-open"
            ? "badge--half-open"
            : "badge--closed";

      return `
        <tr>
          <td>
            <div class="backend-name">
              <span class="backend-dot"></span>
              <span>${normalizeHostLabel(backend.url)}</span>
            </div>
          </td>
          <td>${backend.requests ?? 0}</td>
          <td>${backend.errors ?? 0}</td>
          <td>${formatMs(backend.average_latency_ms)}</td>
          <td>${formatMs(backend.p99_latency_ms)}</td>
          <td><span class="badge ${badgeClass}">${state}</span></td>
        </tr>
      `;
    })
    .join("");
}

function formatMs(value) {
  const number = Number(value || 0);
  return `${number.toFixed(number < 10 ? 1 : 0)} ms`;
}

function drawTrafficChart() {
  const canvas = els.trafficChart;
  const ctx = canvas.getContext("2d");
  const cssWidth = Math.max(canvas.clientWidth, 320);
  const cssHeight = TRAFFIC_CHART_HEIGHT;
  const dpr = window.devicePixelRatio || 1;
  canvas.style.height = `${cssHeight}px`;
  canvas.width = Math.floor(cssWidth * dpr);
  canvas.height = Math.floor(cssHeight * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  const padding = { top: 26, right: 18, bottom: 36, left: 48 };
  const chartWidth = cssWidth - padding.left - padding.right;
  const chartHeight = cssHeight - padding.top - padding.bottom;

  ctx.clearRect(0, 0, cssWidth, cssHeight);

  const labels = history.map((point) => point.label);
  const requestValues = history.map((point) => point.requests);
  const errorValues = history.map((point) => point.errors);
  const maxValue = Math.max(1, ...requestValues, ...errorValues);
  const yTicks = 4;

  drawChartFrame(ctx, cssWidth, cssHeight, padding, yTicks, maxValue);

  plotLine(ctx, requestValues, maxValue, {
    padding,
    chartWidth,
    chartHeight,
    color: "#2d7fe0",
    fillColor: "rgba(45, 127, 224, 0.16)",
  });

  plotLine(ctx, errorValues, maxValue, {
    padding,
    chartWidth,
    chartHeight,
    color: "#ef404f",
    fillColor: "rgba(239, 64, 79, 0.14)",
  });

  ctx.fillStyle = "#53647c";
  ctx.font = "12px Segoe UI";
  ctx.textAlign = "center";

  const labelInterval = Math.max(1, Math.round((X_AXIS_LABEL_INTERVAL_SECONDS * 1000) / POLL_INTERVAL_MS));

  labels.forEach((label, index) => {
    const isLast = index === labels.length - 1;
    const shouldDraw = index % labelInterval === 0 || isLast;
    if (!shouldDraw) {
      return;
    }
    const x = padding.left + (labels.length <= 1 ? chartWidth / 2 : (index * chartWidth) / (labels.length - 1));
    ctx.fillText(label, x, cssHeight - 10);
  });

  drawLegend(ctx, [
    { label: "Requests", color: "#2d7fe0", x: cssWidth - 200 },
    { label: "Errors", color: "#ef404f", x: cssWidth - 96 },
  ]);
}

function drawLatencyChart(metrics) {
  const canvas = els.latencyChart;
  const ctx = canvas.getContext("2d");
  const cssWidth = Math.max(canvas.clientWidth, 280);
  const cssHeight = LATENCY_CHART_HEIGHT;
  const dpr = window.devicePixelRatio || 1;
  canvas.style.height = `${cssHeight}px`;
  canvas.width = Math.floor(cssWidth * dpr);
  canvas.height = Math.floor(cssHeight * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  const padding = { top: 26, right: 18, bottom: 46, left: 48 };
  const chartWidth = cssWidth - padding.left - padding.right;
  const chartHeight = cssHeight - padding.top - padding.bottom;

  ctx.clearRect(0, 0, cssWidth, cssHeight);

  const latency = aggregateLatency(metrics.backends || []);
  const values = [latency.avg, latency.p99];
  const labels = ["Avg Latency", "P99 Latency"];
  const colors = ["#7cb0e3", "#2d7fe0"];
  const maxValue = Math.max(10, ...values) * 1.25;
  const yTicks = 4;

  drawChartFrame(ctx, cssWidth, cssHeight, padding, yTicks, maxValue);

  const barWidth = Math.min(70, chartWidth / 5);
  const gap = Math.max(22, chartWidth / 8.5);

  values.forEach((value, index) => {
    const x = padding.left + gap + index * (barWidth + gap);
    const barHeight = (value / maxValue) * chartHeight;
    const y = padding.top + chartHeight - barHeight;

    const gradient = ctx.createLinearGradient(0, y, 0, padding.top + chartHeight);
    gradient.addColorStop(0, colors[index]);
    gradient.addColorStop(1, index === 0 ? "#a7cbef" : "#2158a6");

    ctx.fillStyle = gradient;
    ctx.fillRect(x, y, barWidth, barHeight);

    ctx.fillStyle = "#102844";
    ctx.font = "bold 16px Segoe UI";
    ctx.textAlign = "center";
    ctx.fillText(formatMsLabel(value), x + barWidth / 2, y - 10);

    ctx.font = "bold 14px Segoe UI";
    ctx.fillText(labels[index], x + barWidth / 2, cssHeight - 14);
  });
}

function formatMsLabel(value) {
  const rounded = Number(value || 0);
  return `${rounded.toFixed(rounded < 10 ? 1 : 0)} ms`;
}

function drawChartFrame(ctx, width, height, padding, yTicks, maxValue) {
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  ctx.strokeStyle = "#d7e2f1";
  ctx.lineWidth = 1;

  for (let i = 0; i <= yTicks; i += 1) {
    const y = padding.top + (i * chartHeight) / yTicks;
    ctx.beginPath();
    ctx.moveTo(padding.left, y);
    ctx.lineTo(width - padding.right, y);
    ctx.stroke();

    const tickValue = Math.round(maxValue - (i * maxValue) / yTicks);
    ctx.fillStyle = "#586981";
    ctx.font = "12px Segoe UI";
    ctx.textAlign = "right";
    ctx.fillText(String(tickValue), padding.left - 10, y + 4);
  }

  ctx.strokeStyle = "#213450";
  ctx.lineWidth = 1.3;
  ctx.beginPath();
  ctx.moveTo(padding.left, padding.top);
  ctx.lineTo(padding.left, height - padding.bottom);
  ctx.lineTo(width - padding.right, height - padding.bottom);
  ctx.stroke();
}

function plotLine(ctx, values, maxValue, options) {
  const { padding, chartWidth, chartHeight, color, fillColor } = options;
  if (!values.length) {
    return;
  }

  ctx.beginPath();
  values.forEach((value, index) => {
    const x = padding.left + (values.length <= 1 ? chartWidth / 2 : (index * chartWidth) / (values.length - 1));
    const y = padding.top + chartHeight - (value / maxValue) * chartHeight;
    if (index === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  });

  ctx.strokeStyle = color;
  ctx.lineWidth = 2.5;
  ctx.stroke();

  ctx.lineTo(padding.left + chartWidth, padding.top + chartHeight);
  ctx.lineTo(padding.left, padding.top + chartHeight);
  ctx.closePath();
  ctx.fillStyle = fillColor;
  ctx.fill();

  values.forEach((value, index) => {
    const x = padding.left + (values.length <= 1 ? chartWidth / 2 : (index * chartWidth) / (values.length - 1));
    const y = padding.top + chartHeight - (value / maxValue) * chartHeight;
    ctx.beginPath();
    ctx.fillStyle = color;
    ctx.arc(x, y, TRAFFIC_POINT_RADIUS, 0, Math.PI * 2);
    ctx.fill();
  });
}

function drawLegend(ctx, items) {
  ctx.font = "bold 13px Segoe UI";
  ctx.textAlign = "left";

  items.forEach((item) => {
    ctx.fillStyle = item.color;
    ctx.fillRect(item.x, 18, 16, 4);
    ctx.fillStyle = "#25354f";
    ctx.fillText(item.label, item.x + 22, 24);
  });
}

function renderAll(metrics) {
  pushHistory(metrics);
  renderSummary(metrics);
  renderBackendTable(metrics.backends || []);
  drawTrafficChart();
  drawLatencyChart(metrics);
}

async function boot() {
  try {
    latestMetrics = await fetchMetrics();
    previousSnapshot = {
      totalRequests: Number(latestMetrics.total_requests || 0),
      totalErrors: Number(latestMetrics.total_errors || 0),
    };
    renderAll(latestMetrics);
  } catch (error) {
    els.refreshPill.textContent = `Initial load failed: ${error.message}`;
    els.refreshPill.style.background = "rgba(225, 70, 85, 0.12)";
    els.refreshPill.style.color = "#b92e3c";
  }

  setInterval(async () => {
    try {
      latestMetrics = await fetchMetrics();
      els.refreshPill.style.background = "rgba(45, 127, 224, 0.1)";
      els.refreshPill.style.color = "#2158a6";
      renderAll(latestMetrics);
    } catch (error) {
      els.refreshPill.textContent = `Refresh failed: ${error.message}`;
      els.refreshPill.style.background = "rgba(225, 70, 85, 0.12)";
      els.refreshPill.style.color = "#b92e3c";
    }
  }, POLL_INTERVAL_MS);
}

window.addEventListener("resize", () => {
  if (latestMetrics) {
    drawTrafficChart();
    drawLatencyChart(latestMetrics);
  }
});

boot();
