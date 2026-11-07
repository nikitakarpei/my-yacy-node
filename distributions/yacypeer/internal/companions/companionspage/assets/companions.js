"use strict";

const page = document.querySelector("main");
const refreshPause = 3000;
const paused = document.getElementById("paused");
const pauses = page.dataset.pauses === "true";
const settings = document.querySelector("form.settings");

if (new URLSearchParams(window.location.search).has("paired")) {
  window.history.replaceState(null, "", page.dataset.path);
}

if (settings) {
  settings.addEventListener("change", () => settings.requestSubmit());
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

function listChanged(devices) {
  return (
    page.dataset.list === "true" &&
    (devices.length !== rows.size || devices.some((device) => !rows.has(device.device)))
  );
}

async function refresh() {
  try {
    const response = await fetch(page.dataset.connections, { cache: "no-store" });
    const devices = await response.json();
    if (listChanged(devices)) {
      window.location.reload();
      return;
    }
    devices.filter((device) => rows.has(device.device)).forEach(show);
    paused.hidden = !pauses || devices.some((device) => device.connected);
  } catch (error) {
    console.warn("device connections not read", error);
    return;
  }
}

if (rows.size > 0 || pauses) {
  window.setInterval(refresh, refreshPause);
}
