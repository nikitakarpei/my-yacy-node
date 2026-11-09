"use strict";

const page = document.querySelector("main");
const codeImage = page.querySelector(".code-image");
const codeState = page.querySelector(".code-state");
const codeStateText = page.querySelector(".code-state-text");
const refusal = page.querySelector(".refusal");

const events = new EventSource(page.dataset.pairingEvents);

events.addEventListener("code", (event) => {
  codeImage.innerHTML = event.data;
  codeImage.classList.remove("waiting");
  codeState.hidden = true;
});

events.addEventListener("begun", (event) => {
  const { deviceName } = JSON.parse(event.data);
  codeStateText.textContent = `Pairing with ${deviceName}…`;
  codeImage.classList.add("waiting");
  codeState.hidden = false;
  refusal.hidden = true;
});

events.addEventListener("refused", (event) => {
  const { deviceName, reason } = JSON.parse(event.data);
  refusal.querySelector(".refusal-title").textContent = `Couldn’t pair ${deviceName}`;
  refusal.querySelector(".refusal-reason").textContent = reason;
  refusal.hidden = false;
});

events.addEventListener("paired", (event) => {
  events.close();
  window.location.assign(JSON.parse(event.data).companionsURL);
});
