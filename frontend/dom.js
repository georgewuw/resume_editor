export function element(id) {
  return document.getElementById(id);
}

export function cssNumber(name) {
  return Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue(name));
}
