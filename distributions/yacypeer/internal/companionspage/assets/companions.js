"use strict";

const page = document.querySelector("main");
const refreshPause = 3000;

if (new URLSearchParams(window.location.search).has("paired")) {
  window.history.replaceState(null, "", page.dataset.path);
}

const rows = new Map(
  [...page.querySelectorAll("tr[data-device]")].map((row) => [row.dataset.device, row]),
);

function show(device) {
  const row = rows.get(device.device);
  row.querySelector(".app-version").textContent = device.appVersion;
  const cell = row.querySelector(".last-connected");
  if (device.connected) {
    const badge = document.createElement("span");
    badge.className = "connected";
    badge.textContent = "Connected";
    cell.replaceChildren(badge);
  } else {
    cell.textContent = device.lastConnected;
  }
}

async function refresh() {
  try {
    const response = await fetch(page.dataset.connections, { cache: "no-store" });
    const devices = await response.json();
    if (devices.length !== rows.size || devices.some((device) => !rows.has(device.device))) {
      window.location.reload();
      return;
    }
    devices.forEach(show);
  } catch {
    return;
  }
}

if (rows.size > 0) {
  window.setInterval(refresh, refreshPause);
}
