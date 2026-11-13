/**
 * Snapshot the document contract without depending on newer WebView clone APIs.
 * @param {import('./contracts.js').ResumeContent} content
 * @returns {import('./contracts.js').ResumeContent}
 */
export function copyContent(content) {
  return {
    fields: Object.fromEntries(
      Object.entries(content.fields).map(([key, lines]) => [key, lines.slice()]),
    ),
  };
}
