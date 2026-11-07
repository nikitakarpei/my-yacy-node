"use strict";

const page = document.querySelector("main");
const codeImage = page.querySelector(".code-image");
const codeState = page.querySelector(".code-state");
const codeStateText = page.querySelector(".code-state-text");
const refusal = page.querySelector(".alert");

const events = new EventSource(page.dataset.events);

events.addEventListener("code", (event) => {
  codeImage.innerHTML = event.data;
  codeImage.classList.remove("waiting");
  codeState.hidden = true;
});

events.addEventListener("started", (event) => {
  const { deviceName } = JSON.parse(event.data);
  codeStateText.textContent = `Pairing with ${deviceName}…`;
  codeImage.classList.add("waiting");
  codeState.hidden = false;
  refusal.hidden = true;
});

events.addEventListener("refused", (event) => {
  const { deviceName, cause } = JSON.parse(event.data);
  refusal.querySelector(".alert-title").textContent = `Couldn’t pair ${deviceName}`;
  refusal.querySelector(".alert-cause").textContent = cause;
  refusal.hidden = false;
});

events.addEventListener("paired", (event) => {
  events.close();
  window.location.assign(JSON.parse(event.data).url);
});
