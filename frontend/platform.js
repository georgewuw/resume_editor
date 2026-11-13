// Desktop conventions are derived from mygo, independent of browser user agents.
export function createDesktopPlatform(name) {
  const macOS = name === 'darwin';

  function shortcutLabel(key) {
    return (macOS ? '⌘' : 'Ctrl+') + key.toUpperCase();
  }

  function matchesShortcut(event, key) {
    return (
      (macOS ? event.metaKey : event.ctrlKey) &&
      !event.altKey &&
      event.key.toLowerCase() === key.toLowerCase()
    );
  }

  return { shortcutLabel, matchesShortcut };
}
