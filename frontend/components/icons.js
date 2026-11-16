const paths = {
  plus: 'M10 4v12M4 10h12',
  trash: 'M3 5h14M7 5V3h6v2M5 5l1 12h8l1-12M8 8v6M12 8v6',
  file: 'M5 2.5h7l3 3v12H5zM12 2.5v4h3M8 10h4M8 13h4',
  personal: 'M13 6.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0ZM4 17v-1a6 6 0 0 1 12 0v1',
  advantage: 'M4 5h12M4 10h12M4 15h8',
  work: 'M7 6V4h6v2M3 6h14v11H3zM3 10a16 16 0 0 0 14 0M10 10v3',
  project: 'M3 3h5v5H3zM12 3h5v5h-5zM3 12h5v5H3zM12 12h5v5h-5z',
  education: 'M2 7l8-4 8 4-8 4zM5 9v5c3 2 7 2 10 0V9M18 7v6',
  certificate: 'M5 3h10v10H5zM8 6h4M8 9h4M7 13v5l3-2 3 2v-5',
};

export function createIcon(name) {
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('viewBox', '0 0 20 20');
  svg.setAttribute('aria-hidden', 'true');
  const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  path.setAttribute('d', paths[name] || paths.file);
  svg.append(path);
  return svg;
}
