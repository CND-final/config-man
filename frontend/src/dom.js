export function $(selector) {
  return document.querySelector(selector);
}

export function $all(selector) {
  return [...document.querySelectorAll(selector)];
}

export function showToast(message) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.add("show");
  window.setTimeout(() => toast.classList.remove("show"), 2200);
}

export function setAuthenticated(authenticated) {
  $("#loginScreen").classList.toggle("hidden", authenticated);
  $("#appShell").classList.toggle("hidden", !authenticated);
}
