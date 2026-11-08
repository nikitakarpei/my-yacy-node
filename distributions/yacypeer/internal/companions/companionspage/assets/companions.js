"use strict";

const page = document.querySelector("main");
const refreshPause = 3000;
const pausedBanner = document.getElementById("paused-banner");
const settingsForm = document.querySelector("form.settings");

if (new URLSearchParams(window.location.search).has(page.dataset.pairedParameter)) {
  window.history.replaceState(null, "", page.dataset.companionsPath);
}

if (settingsForm) {
  settingsForm.addEventListener("change", () => settingsForm.requestSubmit());
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
    const response = await fetch(page.dataset.states, { cache: "no-store" });
    const states = await response.json();
    if (listChanged(states.devices)) {
      window.location.reload();
      return;
    }
    states.devices.filter((device) => rows.has(device.id)).forEach(show);
    pausedBanner.hidden = !states.fetchingPaused;
  } catch (error) {
    console.warn("companions page states not read", error);
    return;
  }
}

window.setInterval(refresh, refreshPause);
