"use strict";

(() => {
  const ranges = [
    ["1h", 3600],
    ["6h", 21600],
    ["24h", 86400],
    ["7d", 604800],
  ];
  const queriesAtOnce = 6;
  const statTypes = ["stat", "gauge", "bargauge"];
  const statLookbackSeconds = 300;
  const allValue = "$__all";
  const autoValue = "auto";
  const heatmapLabelFont = "11px system-ui, sans-serif";
  const palette = [
    "#73BF69",
    "#F2CC0C",
    "#8AB8FF",
    "#FF780A",
    "#F2495C",
    "#5794F2",
    "#B877D9",
    "#705DA0",
    "#37872D",
    "#FADE2A",
    "#447EBC",
    "#C15C17",
    "#890F02",
    "#0A437C",
    "#6D1F62",
    "#584477",
    "#B7DBAB",
    "#F4D598",
    "#70DBED",
    "#F9BA8F",
  ];
  const namedColors = {
    green: "#73BF69",
    "dark-green": "#37872D",
    "semi-dark-green": "#56A64B",
    "light-green": "#96D98D",
    "super-light-green": "#C8F2C2",
    yellow: "#FADE2A",
    "dark-yellow": "#E0B400",
    "light-yellow": "#FFEE52",
    red: "#F2495C",
    "dark-red": "#C4162A",
    "semi-dark-red": "#E02F44",
    "light-red": "#FF7383",
    "super-light-red": "#FFA6B0",
    blue: "#5794F2",
    "dark-blue": "#1F60C4",
    "light-blue": "#8AB8FF",
    orange: "#FF9830",
    "dark-orange": "#FA6400",
    "light-orange": "#FFB357",
    purple: "#B877D9",
    "dark-purple": "#8F3BB8",
    "light-purple": "#CA95E5",
    text: "#CCCCDC",
  };
  const roundedIntervals = [
    [10, 1],
    [15, 10],
    [35, 20],
    [75, 50],
    [150, 100],
    [350, 200],
    [750, 500],
    [1500, 1000],
    [3500, 2000],
    [7500, 5000],
    [12500, 10000],
    [17500, 15000],
    [25000, 20000],
    [45000, 30000],
    [90000, 60000],
    [210000, 120000],
    [450000, 300000],
    [750000, 600000],
    [1050000, 900000],
    [1500000, 1200000],
    [2700000, 1800000],
    [5400000, 3600000],
    [8100000, 7200000],
    [16200000, 10800000],
    [32400000, 21600000],
    [86400000, 43200000],
    [604800000, 86400000],
    [1814400000, 604800000],
  ];

  const model = JSON.parse(document.getElementById("model").textContent);
  const themeStyle = getComputedStyle(document.documentElement);
  const theme = Object.fromEntries(
    ["text", "muted", "border"].map((name) => [
      name,
      themeStyle.getPropertyValue(`--${name}`).trim(),
    ]),
  );
  const sampleIntervalSeconds = Number(
    document.getElementById("dashboard").dataset.sampleIntervalSeconds,
  );
  const state = {
    rangeSeconds: rangeSecondsFrom(new URLSearchParams(location.search)),
    variables: [],
    end: 0,
  };
  const panels = [];
  const querySlots = querySlotsFor(queriesAtOnce);
  const descriptionTip = document.createElement("div");
  descriptionTip.className = "description-tip";

  start();

  async function start() {
    state.end = Date.now() / 1000;
    renderRanges();
    await resolveVariables();
    layOut();
    const refreshSeconds = secondsOf(model.refresh || "");
    if (refreshSeconds > 0) {
      setInterval(refresh, Math.max(refreshSeconds, sampleIntervalSeconds) * 1000);
    }
    window.addEventListener("resize", debounced(resizeCharts, 200));
    document.addEventListener("visibilitychange", refresh);
  }

  function rangeSecondsFrom(given) {
    const asked = ranges.find(([name]) => name === given.get("range"));
    return (asked || ranges[0])[1];
  }

  function querySlotsFor(slotCount) {
    let slotsTaken = 0;
    const waitingQueries = [];
    return {
      async run(query) {
        if (slotsTaken < slotCount) {
          slotsTaken++;
        } else {
          await new Promise((resolve) => waitingQueries.push(resolve));
        }
        try {
          return await query();
        } finally {
          const nextQuery = waitingQueries.shift();
          if (nextQuery) {
            nextQuery();
          } else {
            slotsTaken--;
          }
        }
      },
    };
  }

  function renderRanges() {
    const container = document.getElementById("ranges");
    container.replaceChildren(
      ...ranges.map(([name, seconds]) => {
        const button = document.createElement("button");
        button.textContent = name;
        button.className = seconds === state.rangeSeconds ? "active" : "";
        button.addEventListener("click", async () => {
          state.rangeSeconds = seconds;
          rememberInAddress();
          renderRanges();
          await resolveVariables();
          refresh();
        });
        return button;
      }),
    );
  }

  async function resolveVariables() {
    const listed = (model.templating && model.templating.list) || [];
    const resolved = [];
    for (const variable of listed) {
      if (variable.type === "datasource") {
        continue;
      }
      const options = await optionsOf(variable, resolved);
      resolved.push({
        variable,
        options,
        selected: selectedOf(variable, options),
      });
    }
    state.variables = resolved;
    renderVariables();
  }

  async function optionsOf(variable, resolved) {
    switch (variable.type) {
      case "interval": {
        const options = splitOptions(variable.query);
        return variable.auto ? [autoValue, ...options] : options;
      }
      case "custom":
        return splitOptions(variable.query);
      case "constant":
      case "textbox":
        return [variable.query || ""];
      case "query":
        return labelValuesOf(variable, resolved);
      default:
        return [];
    }
  }

  function splitOptions(text) {
    return String(text || "")
      .split(",")
      .map((option) => option.trim())
      .filter(Boolean);
  }

  async function labelValuesOf(variable, resolved) {
    const definition = typeof variable.query === "object" ? variable.query.query : variable.query;
    const parsed = /^\s*label_values\((.+),\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\)\s*$/.exec(
      definition || "",
    );
    if (!parsed) {
      return [];
    }
    const [, selector, label] = parsed;
    const asked = new URLSearchParams({ "match[]": interpolatedText(selector, {}, resolved) });
    try {
      const response = await fetch(
        `/dashboards/api/v1/label/${encodeURIComponent(label)}/values?${asked}`,
      );
      const answer = await response.json();
      return answer.status === "success" ? answer.data : [];
    } catch (failure) {
      return [];
    }
  }

  function selectedOf(variable, options) {
    const asked = new URLSearchParams(location.search).get(`var-${variable.name}`);
    if (asked !== null) {
      return asked;
    }
    const current = variable.current && variable.current.value;
    if (variable.type === "interval") {
      return current && String(current).startsWith("$__auto") ? autoValue : current || options[0];
    }
    if (variable.includeAll && (current === allValue || current === undefined)) {
      return allValue;
    }
    return Array.isArray(current) ? current[0] : current || options[0] || "";
  }

  function renderVariables() {
    const container = document.getElementById("variables");
    container.replaceChildren(
      ...state.variables
        .filter(({ variable }) => variable.hide !== 2 && variable.type !== "constant")
        .map((resolved) => {
          const label = document.createElement("label");
          label.textContent = `${resolved.variable.label || resolved.variable.name} `;
          const select = document.createElement("select");
          const choices = resolved.variable.includeAll
            ? [allValue, ...resolved.options]
            : resolved.options;
          for (const choice of choices) {
            const option = document.createElement("option");
            option.value = choice;
            option.textContent = choice === allValue ? "All" : choice;
            option.selected = choice === resolved.selected;
            select.append(option);
          }
          select.addEventListener("change", () => {
            resolved.selected = select.value;
            rememberInAddress();
            refresh();
          });
          label.append(select);
          return label;
        }),
    );
  }

  function rememberInAddress() {
    const address = new URLSearchParams();
    address.set("range", ranges.find(([, seconds]) => seconds === state.rangeSeconds)[0]);
    for (const { variable, selected } of state.variables) {
      address.set(`var-${variable.name}`, selected);
    }
    history.replaceState(null, "", `?${address}`);
  }

  function interpolatedText(text, scopedVariables, resolved = state.variables) {
    return substitutedText(text, (name) => {
      if (Object.hasOwn(scopedVariables, name)) {
        return scopedVariables[name];
      }
      const found = resolved.find(({ variable }) => variable.name === name);
      return found && valueOf(found);
    });
  }

  function substitutedText(text, valueNamed) {
    return String(text).replace(
      /\$\{(\w+)(?::\w+)?\}|\[\[(\w+)\]\]|\$(\w+)/g,
      (match, braced, bracketed, bare) => valueNamed(braced || bracketed || bare) ?? match,
    );
  }

  function valueOf({ variable, options, selected }) {
    if (variable.type === "interval") {
      return selected === autoValue ? autoIntervalOf(variable) : selected;
    }
    if (selected !== allValue) {
      return regexEscaped(selected);
    }
    if (variable.allValue) {
      return variable.allValue;
    }
    return `(${options.map(regexEscaped).join("|")})`;
  }

  function titleOf(text) {
    return substitutedText(text, (name) => {
      const found = state.variables.find(({ variable }) => variable.name === name);
      return found && (found.selected === allValue ? "All" : found.selected);
    });
  }

  function regexEscaped(value) {
    return String(value).replace(/[\\^$*+?.()|[\]{}]/g, "\\\\$&");
  }

  function autoIntervalOf(variable) {
    const count = Number(variable.auto_count) || 30;
    const floor = secondsOf(variable.auto_min || "10s") * 1000;
    return durationLiteralOf(Math.max(roundedInterval((state.rangeSeconds * 1000) / count), floor));
  }

  function roundedInterval(milliseconds) {
    const rounded = roundedIntervals.find(([limit]) => milliseconds <= limit);
    return rounded ? rounded[1] : 2592000000;
  }

  function secondsOf(text) {
    const units = { ms: 0.001, s: 1, m: 60, h: 3600, d: 86400, w: 604800, y: 31536000 };
    let seconds = 0;
    for (const [, amount, unit] of String(text).matchAll(/(\d+(?:\.\d+)?)(ms|s|m|h|d|w|y)/g)) {
      seconds += Number(amount) * units[unit];
    }
    return seconds;
  }

  function durationLiteralOf(milliseconds) {
    const units = [
      ["d", 86400000],
      ["h", 3600000],
      ["m", 60000],
      ["s", 1000],
    ];
    for (const [unit, size] of units) {
      if (milliseconds >= size && milliseconds % size === 0) {
        return `${milliseconds / size}${unit}`;
      }
    }
    return `${Math.max(1, Math.round(milliseconds))}ms`;
  }

  function layOut() {
    const grid = document.getElementById("dashboard");
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          const panel = panels.find((candidate) => candidate.element === entry.target);
          panel.visible = entry.isIntersecting;
          if (panel.visible && panel.stale) {
            draw(panel);
          }
        }
      },
      { rootMargin: "300px" },
    );
    for (const spec of flattenedPanels(model.panels || [])) {
      const element = document.createElement("section");
      const position = spec.gridPos || { x: 0, y: 0, w: 12, h: 8 };
      element.style.setProperty("--grid-column", `${position.x + 1} / span ${position.w}`);
      element.style.setProperty("--grid-row", `${position.y + 1} / span ${position.h}`);
      element.style.setProperty("--grid-height", position.h);
      element.style.order = position.y * 24 + position.x;
      if (spec.type === "row") {
        element.className = "row";
        element.textContent = spec.title || "";
        grid.append(element);
        continue;
      }
      element.className = "panel";
      const heading = document.createElement("h2");
      const title = document.createElement("span");
      title.className = "title";
      title.textContent = titleOf(spec.title || "");
      heading.append(title);
      if (spec.description) {
        heading.append(descriptionMarkerFor(spec.description));
      }
      const body = document.createElement("div");
      body.className = "body";
      element.append(heading, body);
      grid.append(element);
      const panel = {
        spec,
        element,
        title,
        body,
        chart: null,
        stale: true,
        visible: false,
        generation: 0,
      };
      panels.push(panel);
      observer.observe(element);
    }
  }

  function descriptionMarkerFor(description) {
    const marker = document.createElement("span");
    marker.className = "description-marker";
    marker.textContent = "ⓘ";
    marker.addEventListener("mouseenter", () => showDescription(marker, description));
    marker.addEventListener("mouseleave", () => descriptionTip.remove());
    return marker;
  }

  function showDescription(marker, description) {
    descriptionTip.textContent = description;
    document.body.append(descriptionTip);
    const anchor = marker.getBoundingClientRect();
    const left = Math.min(anchor.left, window.innerWidth - descriptionTip.offsetWidth - 8);
    descriptionTip.style.left = `${Math.max(8, left)}px`;
    descriptionTip.style.top = `${anchor.bottom + 4}px`;
  }

  function flattenedPanels(listed) {
    return listed.flatMap((spec) =>
      spec.type === "row" && spec.collapsed && spec.panels ? [spec, ...spec.panels] : [spec],
    );
  }

  function refresh() {
    if (document.hidden) {
      return;
    }
    state.end = Date.now() / 1000;
    for (const panel of panels) {
      panel.stale = true;
      if (panel.visible) {
        draw(panel);
      }
    }
  }

  async function draw(panel) {
    panel.stale = false;
    panel.title.textContent = titleOf(panel.spec.title || "");
    const generation = ++panel.generation;
    const isStat = statTypes.includes(panel.spec.type);
    const span = querySpanOf(panel, isStat ? statLookbackSeconds : state.rangeSeconds);
    const targets = (panel.spec.targets || []).filter((target) => target.expr && !target.hide);
    let answers;
    try {
      answers = await Promise.all(
        targets.map(async (target) => ({
          target,
          series: await rangeQueryOf(
            interpolatedText(target.expr, span.scopedVariables),
            span.start,
            span.end,
            span.step,
          ),
        })),
      );
    } catch (failure) {
      if (generation === panel.generation) {
        showFailure(panel, failure.message);
      }
      return;
    }
    if (generation !== panel.generation) {
      return;
    }
    const drawn = answers.flatMap(({ target, series }) =>
      series
        .filter((found) => found.values.size > 0)
        .map((found) => ({
          name: legendOf(target, found.metric),
          metric: found.metric,
          values: found.values,
        })),
    );
    if (drawn.length === 0) {
      showNotice(panel, "No data");
      return;
    }
    if (isStat) {
      drawStat(panel, drawn);
    } else if (panel.spec.type === "heatmap") {
      drawHeatmap(panel, drawn, span);
    } else {
      drawChart(panel, drawn, span);
    }
  }

  function querySpanOf(panel, rangeSeconds) {
    const rangeMilliseconds = rangeSeconds * 1000;
    const points = panel.spec.maxDataPoints || Math.max(100, panel.body.clientWidth || 600);
    const floor = panel.spec.interval
      ? secondsOf(interpolatedText(panel.spec.interval, {})) * 1000
      : 0;
    const interval = Math.max(
      roundedInterval(rangeMilliseconds / points),
      floor || sampleIntervalSeconds * 1000,
    );
    const step = interval / 1000;
    const end = Math.ceil(state.end / step) * step;
    const start = Math.floor((state.end - rangeSeconds) / step) * step;
    const rateInterval = Math.max(
      interval + sampleIntervalSeconds * 1000,
      4 * sampleIntervalSeconds * 1000,
    );
    return {
      start,
      end,
      step,
      scopedVariables: {
        __interval: durationLiteralOf(interval),
        __interval_ms: String(interval),
        __rate_interval: durationLiteralOf(rateInterval),
        __range: durationLiteralOf(rangeMilliseconds),
        __range_s: String(rangeSeconds),
        __range_ms: String(rangeMilliseconds),
      },
    };
  }

  function rangeQueryOf(expr, start, end, step) {
    return querySlots.run(() => seriesFromRangeQuery(expr, start, end, step));
  }

  async function seriesFromRangeQuery(expr, start, end, step) {
    const form = new URLSearchParams({ query: expr, start, end, step });
    const response = await fetch("/dashboards/api/v1/query_range", { method: "POST", body: form });
    const answer = await response.json();
    if (answer.status !== "success") {
      throw new Error(answer.error || response.statusText);
    }
    return answer.data.result.map((found) => ({
      metric: found.metric,
      values: new Map(
        (found.values || [])
          .map(([at, value]) => [Number(at), Number(value)])
          .filter(([, value]) => !Number.isNaN(value)),
      ),
    }));
  }

  function legendOf(target, metric) {
    const format = target.legendFormat;
    if (format && format !== "__auto") {
      return format.replace(/\{\{\s*(.+?)\s*\}\}/g, (match, label) => metric[label] || "");
    }
    const labels = Object.entries(metric)
      .filter(([name]) => name !== "__name__")
      .map(([name, value]) => `${name}="${value}"`);
    const name = metric.__name__ || "";
    if (labels.length === 0) {
      return name || target.expr;
    }
    return `${name}{${labels.join(", ")}}`;
  }

  function showNotice(panel, text) {
    replaceWithNotice(panel, text, "notice");
  }

  function showFailure(panel, text) {
    replaceWithNotice(panel, text, "notice failure");
  }

  function replaceWithNotice(panel, text, className) {
    destroyChart(panel);
    const notice = document.createElement("div");
    notice.className = className;
    notice.textContent = text;
    panel.body.replaceChildren(notice);
  }

  function destroyChart(panel) {
    if (panel.chart) {
      panel.chart.destroy();
      panel.chart = null;
    }
    panel.resize = null;
  }

  function drawStat(panel, drawn) {
    destroyChart(panel);
    const defaults = fieldDefaultsOf(panel.spec);
    const container = document.createElement("div");
    container.className = "stat";
    for (const { name, values } of drawn) {
      const latest = [...values.values()].pop();
      const entry = document.createElement("div");
      const value = document.createElement("div");
      value.className = "value";
      value.textContent = valueTextOf(latest, defaults.unit, defaults.decimals);
      value.style.color = thresholdColorOf(defaults.thresholds, latest);
      entry.append(value);
      if (drawn.length > 1) {
        const label = document.createElement("div");
        label.className = "name";
        label.textContent = name;
        entry.append(label);
      }
      container.append(entry);
    }
    panel.body.replaceChildren(container);
  }

  function thresholdColorOf(thresholds, value) {
    const steps = (thresholds && thresholds.steps) || [];
    let color = theme.text;
    for (const step of steps) {
      if (step.value === null || step.value === undefined || value >= step.value) {
        color = colorOf(step.color);
      }
    }
    return color;
  }

  function colorOf(name) {
    return namedColors[name] || name;
  }

  function fieldDefaultsOf(spec) {
    return (spec.fieldConfig && spec.fieldConfig.defaults) || {};
  }

  function overridesOf(spec, name) {
    const overrides = (spec.fieldConfig && spec.fieldConfig.overrides) || [];
    const properties = {};
    for (const override of overrides) {
      const matcher = override.matcher || {};
      const matches =
        (matcher.id === "byName" && matcher.options === name) ||
        (matcher.id === "byRegexp" && new RegExp(matcher.options).test(name));
      if (matches) {
        for (const property of override.properties || []) {
          properties[property.id] = property.value;
        }
      }
    }
    return properties;
  }

  function drawChart(panel, drawn, span) {
    const defaults = fieldDefaultsOf(panel.spec);
    const custom = defaults.custom || {};
    const stacked = custom.stacking && custom.stacking.mode === "normal";
    const times = timesOf(span);
    const columns = drawn.map(({ values }) => times.map((at) => values.get(at) ?? null));
    const scaleDistribution = custom.scaleDistribution || {};
    const logBase = scaleDistribution.type === "log" ? scaleDistribution.log || 10 : 0;
    const shownColumns = stacked ? stackedColumnsOf(columns) : columns;
    const plotted = logBase ? shownColumns.map(positiveOnly) : shownColumns;
    let rightUnit;
    const series = drawn.map(({ name }, index) => {
      const overrides = overridesOf(panel.spec, name);
      const unit = overrides.unit || defaults.unit;
      const onRight = overrides["custom.axisPlacement"] === "right";
      if (onRight) {
        rightUnit ??= unit;
      }
      const color =
        overrides.color && overrides.color.fixedColor
          ? colorOf(overrides.color.fixedColor)
          : palette[index % palette.length];
      const fillOpacity =
        overrides["custom.fillOpacity"] ??
        (stacked ? Math.max(custom.fillOpacity || 0, 40) : custom.fillOpacity || 0);
      return {
        label: name,
        scale: onRight ? "right" : "y",
        stroke: color,
        width: custom.lineWidth ?? 1,
        fill: fillOpacity > 0 ? `${color}${alphaHexOf(fillOpacity)}` : undefined,
        paths: custom.drawStyle === "bars" ? uPlot.paths.bars({ size: [0.8, 40] }) : undefined,
        points: { show: false },
        spanGaps: Boolean(custom.spanNulls),
        value: (chart, value, seriesIndex, at) =>
          valueTextOf(
            stacked && at !== null && at !== undefined ? columns[index][at] : value,
            unit,
            defaults.decimals,
          ),
      };
    });
    const axisStyle = {
      stroke: theme.muted,
      grid: { stroke: theme.border, width: 1 },
      ticks: { stroke: theme.border, width: 1 },
    };
    const axes = [
      axisStyle,
      {
        ...axisStyle,
        scale: "y",
        size: axisSizeOf,
        values: (chart, splits) =>
          splits.map((value) => valueTextOf(value, defaults.unit, defaults.decimals)),
        splits: logBase ? (chart, axis, low, high) => powersWithin(low, high, logBase) : undefined,
      },
    ];
    if (rightUnit !== undefined) {
      axes.push({
        ...axisStyle,
        scale: "right",
        side: 1,
        grid: { show: false },
        size: axisSizeOf,
        values: (chart, splits) => splits.map((value) => valueTextOf(value, rightUnit)),
      });
    }
    const legendShown =
      drawn.length <= 30 &&
      !(
        panel.spec.options &&
        panel.spec.options.legend &&
        panel.spec.options.legend.showLegend === false
      );
    const legend = document.createElement("div");
    legend.className = "legend";
    const options = {
      width: panel.body.clientWidth,
      height: panel.body.clientHeight,
      series: [{}, ...series],
      bands: stacked
        ? series.slice(1).map((drawnSeries, index) => ({ series: [index + 2, index + 1] }))
        : [],
      axes,
      legend: { show: legendShown, mount: (chart, table) => legend.append(table) },
      scales: {
        x: { time: true },
        y: logBase
          ? {
              distr: 3,
              log: logBase,
              range: (chart, low, high) => logAxisRangeOf(low, high, logBase),
            }
          : {
              range: (chart, low, high) =>
                axisRangeOf(low, high, defaults.min, defaults.max ?? custom.axisSoftMax),
            },
        right: { range: (chart, low, high) => axisRangeOf(low, high, 0, undefined) },
      },
      cursor: { points: { show: false } },
    };
    destroyChart(panel);
    const plot = document.createElement("div");
    panel.body.replaceChildren(plot, legend);
    panel.chart = new uPlot(options, [times, ...plotted], plot);
    panel.resize = () =>
      panel.chart.setSize({
        width: panel.body.clientWidth,
        height: Math.max(60, panel.body.clientHeight - legend.offsetHeight),
      });
    panel.resize();
  }

  function axisSizeOf(chart, values, axisIndex, cycle) {
    const axis = chart.axes[axisIndex];
    if (cycle > 1) {
      return axis._size;
    }
    const longestValue = (values || []).reduce(
      (longest, value) => (value.length > longest.length ? value : longest),
      "",
    );
    chart.ctx.font = axis.font[0];
    const valueWidth = chart.ctx.measureText(longestValue).width / devicePixelRatio;
    return Math.ceil(axis.ticks.size + axis.gap + valueWidth) + 4;
  }

  function timesOf(span) {
    const times = [];
    for (let at = span.start; at <= span.end; at += span.step) {
      times.push(at);
    }
    return times;
  }

  const colorSchemes = Object.fromEntries(
    Object.entries({
      Viridis: [
        "#440154",
        "#482777",
        "#3F4A8A",
        "#31678E",
        "#26838F",
        "#1F9D8A",
        "#6CCE5A",
        "#B6DE2B",
        "#FEE825",
      ],
      RdYlGn: [
        "#A50026",
        "#D73027",
        "#F46D43",
        "#FDAE61",
        "#FEE08B",
        "#FFFFBF",
        "#D9EF8B",
        "#A6D96A",
        "#66BD63",
        "#1A9850",
        "#006837",
      ],
    }).map(([name, stops]) => [
      name,
      stops.map((hex) => [1, 3, 5].map((at) => parseInt(hex.slice(at, at + 2), 16))),
    ]),
  );

  function drawHeatmap(panel, drawn, span) {
    destroyChart(panel);
    const options = panel.spec.options || {};
    const colorOptions = options.color || {};
    const isBuckets = (panel.spec.targets || []).some((target) => target.format === "heatmap");
    const rowUnit = (options.yAxis && options.yAxis.unit) || fieldDefaultsOf(panel.spec).unit;
    const valueUnit = isBuckets ? "short" : fieldDefaultsOf(panel.spec).unit;
    const rows = isBuckets ? bucketRowsOf(drawn, rowUnit) : seriesRowsOf(drawn);
    const times = timesOf(span);
    const cells = rows.map((row) => times.map((at) => row.valueAt(at)));
    let [lowest, highest] = [Infinity, -Infinity];
    for (const value of cells.flat()) {
      if (value !== null) {
        [lowest, highest] = [Math.min(lowest, value), Math.max(highest, value)];
      }
    }
    if (lowest > highest) {
      showNotice(panel, "No data");
      return;
    }
    const low = colorOptions.min ?? lowest;
    const high = colorOptions.max ?? highest;
    const stops = colorSchemes[colorOptions.scheme] || colorSchemes.Viridis;
    const scheme = colorOptions.reverse ? [...stops].reverse() : stops;
    const container = document.createElement("div");
    container.className = "heatmap";
    const canvas = document.createElement("canvas");
    const tip = document.createElement("div");
    tip.className = "tip";
    container.append(canvas, tip);
    panel.body.replaceChildren(container);
    const margin = { left: rowLabelWidthOf(canvas, rows) + 12, right: 8, top: 4, bottom: 22 };
    const cellOf = (event) => {
      const { cellWidth, cellHeight } = heatmapGeometryOf(
        panel.body,
        margin,
        rows.length,
        times.length,
      );
      const column = Math.floor((event.offsetX - margin.left) / cellWidth);
      const row = rows.length - 1 - Math.floor((event.offsetY - margin.top) / cellHeight);
      return { column, row };
    };
    canvas.addEventListener("mousemove", (event) => {
      const { column, row } = cellOf(event);
      if (column < 0 || column >= times.length || row < 0 || row >= rows.length) {
        tip.style.display = "none";
        return;
      }
      tip.textContent = `${new Date(times[column] * 1000).toLocaleString()} · ${rows[row].label}: ${valueTextOf(cells[row][column], valueUnit)}`;
      tip.style.display = "block";
      tip.style.left = `${Math.min(event.offsetX + 12, panel.body.clientWidth - tip.offsetWidth - 4)}px`;
      tip.style.top = `${Math.max(0, event.offsetY - 24)}px`;
    });
    canvas.addEventListener("mouseleave", () => {
      tip.style.display = "none";
    });
    const hiddenAtOrBelow = options.filterValues?.le ?? 1e-9;
    panel.resize = () =>
      paintHeatmap(canvas, panel.body, margin, {
        rows,
        times,
        cells,
        low,
        high,
        scheme,
        span,
        hiddenAtOrBelow,
        gap: options.cellGap ?? 1,
      });
    panel.resize();
  }

  function rowLabelWidthOf(canvas, rows) {
    const context = canvas.getContext("2d");
    context.font = heatmapLabelFont;
    const widths = rows.map((row) => context.measureText(row.label).width);
    return Math.ceil(Math.max(12, ...widths));
  }

  function bucketRowsOf(drawn, unit) {
    const buckets = drawn
      .map(({ metric, values }) => ({ bound: Number(metric.le), values }))
      .filter(({ bound }) => !Number.isNaN(bound))
      .sort((a, b) => a.bound - b.bound);
    return buckets.map((bucket, index) => ({
      label: Number.isFinite(bucket.bound) ? valueTextOf(bucket.bound, unit) : "+Inf",
      valueAt: (at) => {
        const count = bucket.values.get(at);
        if (count === undefined) {
          return null;
        }
        const below = index > 0 ? buckets[index - 1].values.get(at) : 0;
        return Math.max(0, count - (below || 0));
      },
    }));
  }

  function seriesRowsOf(drawn) {
    return [...drawn]
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }))
      .map(({ name, values }) => ({
        label: name,
        valueAt: (at) => values.get(at) ?? null,
      }));
  }

  function paintHeatmap(canvas, body, margin, heatmap) {
    const { rows, times, cells, low, high, scheme, span, gap, hiddenAtOrBelow } = heatmap;
    const { width, height, cellWidth, cellHeight } = heatmapGeometryOf(
      body,
      margin,
      rows.length,
      times.length,
    );
    const ratio = window.devicePixelRatio || 1;
    canvas.width = width * ratio;
    canvas.height = height * ratio;
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;
    const context = canvas.getContext("2d");
    context.scale(ratio, ratio);
    const cellGap = cellWidth > 3 && cellHeight > 3 ? gap : 0;
    for (let row = 0; row < rows.length; row++) {
      const y = margin.top + (rows.length - 1 - row) * cellHeight;
      for (let column = 0; column < times.length; column++) {
        const value = cells[row][column];
        if (value === null || value <= hiddenAtOrBelow) {
          continue;
        }
        context.fillStyle = schemeColorAt(scheme, high > low ? (value - low) / (high - low) : 0.5);
        context.fillRect(
          margin.left + column * cellWidth,
          y,
          Math.max(1, cellWidth - cellGap),
          Math.max(1, cellHeight - cellGap),
        );
      }
    }
    context.fillStyle = theme.muted;
    context.font = heatmapLabelFont;
    context.textAlign = "right";
    context.textBaseline = "middle";
    const rowStep = Math.max(1, Math.ceil(14 / cellHeight));
    for (let row = 0; row < rows.length; row += rowStep) {
      context.fillText(
        rows[row].label,
        margin.left - 6,
        margin.top + (rows.length - row - 0.5) * cellHeight,
      );
    }
    context.textAlign = "center";
    context.textBaseline = "top";
    const columnStep = Math.max(1, Math.ceil(110 / cellWidth));
    const withDate = span.end - span.start > 2 * 86400;
    for (let column = 0; column < times.length; column += columnStep) {
      const instant = new Date(times[column] * 1000);
      const clock = instant.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
      const label = withDate
        ? `${instant.toLocaleDateString([], { month: "2-digit", day: "2-digit" })} ${clock}`
        : clock;
      context.fillText(label, margin.left + (column + 0.5) * cellWidth, height - margin.bottom + 6);
    }
  }

  function heatmapGeometryOf(body, margin, rowCount, columnCount) {
    const width = body.clientWidth;
    const height = body.clientHeight;
    return {
      width,
      height,
      cellWidth: (width - margin.left - margin.right) / columnCount,
      cellHeight: (height - margin.top - margin.bottom) / rowCount,
    };
  }

  function schemeColorAt(scheme, share) {
    const position = Math.min(1, Math.max(0, share)) * (scheme.length - 1);
    const index = Math.min(scheme.length - 2, Math.floor(position));
    const mix = position - index;
    const [from, to] = [scheme[index], scheme[index + 1]];
    const channels = from.map((channel, at) => Math.round(channel + (to[at] - channel) * mix));
    return `rgb(${channels.join(",")})`;
  }

  function stackedColumnsOf(columns) {
    const sums = columns[0] ? columns[0].map(() => 0) : [];
    const filled = sums.map((sum, at) => columns.some((column) => column[at] !== null));
    return columns.map((column) =>
      column.map((value, at) => {
        if (!filled[at]) {
          return null;
        }
        sums[at] += value || 0;
        return sums[at];
      }),
    );
  }

  function alphaHexOf(percent) {
    return Math.round(Math.min(100, percent) * 2.55)
      .toString(16)
      .padStart(2, "0");
  }

  function powersWithin(low, high, base) {
    const powers = [];
    const first = Math.ceil(Math.log(low) / Math.log(base) - 1e-9);
    const last = Math.floor(Math.log(high) / Math.log(base) + 1e-9);
    for (let exponent = first; exponent <= last; exponent++) {
      powers.push(base ** exponent);
    }
    return powers;
  }

  function positiveOnly(column) {
    return column.map((value) => (value !== null && value > 0 ? value : null));
  }

  function logAxisRangeOf(low, high, base) {
    if (low === null || high === null) {
      return [1, base];
    }
    return uPlot.rangeLog(low, high, base, true);
  }

  function axisRangeOf(low, high, fixedLow, fixedHigh) {
    if (low === null || high === null) {
      return [fixedLow ?? 0, fixedHigh ?? 1];
    }
    const [paddedLow, paddedHigh] = uPlot.rangeNum(low, high, 0.1, true);
    const from = fixedLow ?? (low >= 0 ? Math.max(0, paddedLow) : paddedLow);
    const to =
      fixedHigh !== undefined && fixedHigh !== null
        ? Math.max(fixedHigh, from === high ? high + 1 : high)
        : paddedHigh;
    return [from, to === from ? from + 1 : to];
  }

  function resizeCharts() {
    for (const panel of panels) {
      if (panel.resize) {
        panel.resize();
      }
    }
  }

  function debounced(run, milliseconds) {
    let timer = 0;
    return () => {
      clearTimeout(timer);
      timer = setTimeout(run, milliseconds);
    };
  }

  function valueTextOf(value, unit, decimals) {
    if (value === null || value === undefined || Number.isNaN(value)) {
      return "–";
    }
    switch (unit) {
      case "percentunit":
        return `${fixedTextOf(value * 100, decimals)}%`;
      case "percent":
        return `${fixedTextOf(value, decimals)}%`;
      case "bytes":
        return scaledTextOf(value, 1024, ["B", "KiB", "MiB", "GiB", "TiB", "PiB"], decimals);
      case "decbytes":
        return scaledTextOf(value, 1000, ["B", "kB", "MB", "GB", "TB", "PB"], decimals);
      case "s":
        return durationLabelOf(value, decimals);
      case "ms":
        return durationLabelOf(value / 1000, decimals);
      case "ops":
        return `${shortTextOf(value, decimals)} ops/s`;
      case "reqps":
        return `${shortTextOf(value, decimals)} req/s`;
      case "cps":
        return `${shortTextOf(value, decimals)} c/s`;
      case "none":
        return fixedTextOf(value, decimals);
      default:
        return shortTextOf(value, decimals);
    }
  }

  function fixedTextOf(value, decimals) {
    if (decimals !== null && decimals !== undefined) {
      return value.toFixed(decimals);
    }
    const size = Math.abs(value);
    if (size >= 100 || size === 0) {
      return String(Math.round(value));
    }
    return String(Number(value.toPrecision(3)));
  }

  function scaledTextOf(value, base, units, decimals) {
    let index = 0;
    let size = value;
    while (Math.abs(size) >= base && index < units.length - 1) {
      size /= base;
      index++;
    }
    return `${fixedTextOf(size, decimals)} ${units[index]}`.trim();
  }

  function shortTextOf(value, decimals) {
    return scaledTextOf(value, 1000, ["", "K", "Mil", "Bil", "Tri"], decimals);
  }

  function durationLabelOf(seconds, decimals) {
    const size = Math.abs(seconds);
    if (size === 0) {
      return "0 s";
    }
    if (size < 1e-6) {
      return `${fixedTextOf(seconds * 1e9, decimals)} ns`;
    }
    if (size < 1e-3) {
      return `${fixedTextOf(seconds * 1e6, decimals)} µs`;
    }
    if (size < 1) {
      return `${fixedTextOf(seconds * 1e3, decimals)} ms`;
    }
    if (size < 60) {
      return `${fixedTextOf(seconds, decimals)} s`;
    }
    if (size < 3600) {
      return `${fixedTextOf(seconds / 60, decimals)} min`;
    }
    if (size < 86400) {
      return `${fixedTextOf(seconds / 3600, decimals)} hour`;
    }
    return `${fixedTextOf(seconds / 86400, decimals)} day`;
  }
})();
