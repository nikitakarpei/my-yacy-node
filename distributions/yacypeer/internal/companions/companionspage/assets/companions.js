"use strict";

const page = document.querySelector("main");
const refreshPause = 3000;
const pausedBanner = document.getElementById("paused-banner");
const onlyCompanionsFetch = page.dataset.onlyCompanionsFetch === "true";
const settings = document.querySelector("form.settings");

if (new URLSearchParams(window.location.search).has(page.dataset.pairedParameter)) {
  window.history.replaceState(null, "", page.dataset.companionsPath);
}

if (settings) {
  settings.addEventListener("change", () => settings.requestSubmit());
}

const rows = new Map(
  [...page.querySelectorAll("tr[data-device-id]")].map((row) => [row.dataset.deviceId, row]),
);

function show(device) {
  const row = rows.get(device.id);
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
    page.dataset.listsDevices === "true" &&
    (devices.length !== rows.size || devices.some((device) => !rows.has(device.id)))
  );
}

async function refresh() {
  try {
    const response = await fetch(page.dataset.deviceStates, { cache: "no-store" });
    const devices = await response.json();
    if (listChanged(devices)) {
      window.location.reload();
      return;
    }
    devices.filter((device) => rows.has(device.id)).forEach(show);
    pausedBanner.hidden = !onlyCompanionsFetch || devices.some((device) => device.connected);
  } catch (error) {
    console.warn("device states not read", error);
    return;
  }
}

if (rows.size > 0 || onlyCompanionsFetch) {
  window.setInterval(refresh, refreshPause);
}
