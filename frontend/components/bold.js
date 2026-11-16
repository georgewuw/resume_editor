// Apply the same single-line markers that the resume renderer understands.
export function toggleBold(value, start, end) {
  if (start === end) {
    return {
      value: value.slice(0, start) + '****' + value.slice(end),
      start: start + 2,
      end: start + 2,
    };
  }
  const selected = value.slice(start, end);
  if (
    !selected.includes('\n') &&
    value.slice(start - 2, start) === '**' &&
    value.slice(end, end + 2) === '**'
  ) {
    return {
      value: value.slice(0, start - 2) + selected + value.slice(end + 2),
      start: start - 2,
      end: end - 2,
    };
  }
  const lines = selected.split('\n');
  const remove = lines
    .filter((line) => line.trim())
    .every((line) => /^\*\*[^*\n]+\*\*$/.test(line));
  const replacement = lines
    .map((line) => {
      if (!line.trim()) return line;
      const marked = /^\*\*[^*\n]+\*\*$/.test(line);
      return remove ? line.slice(2, -2) : marked ? line : '**' + line + '**';
    })
    .join('\n');
  return {
    value: value.slice(0, start) + replacement + value.slice(end),
    start,
    end: start + replacement.length,
  };
}
