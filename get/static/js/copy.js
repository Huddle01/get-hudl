// Copy-to-clipboard for any [data-copy="<target-selector>"] button.
// Falls back to execCommand for non-secure contexts.
function copyText(text) {
  if (navigator.clipboard && window.isSecureContext) {
    return navigator.clipboard.writeText(text);
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.focus();
  ta.select();
  try { document.execCommand('copy'); } catch (e) { /* noop */ }
  document.body.removeChild(ta);
  return Promise.resolve();
}

function flashCopied(btn) {
  const original = btn.textContent;
  btn.classList.add('is-copied');
  btn.textContent = 'Copied';
  clearTimeout(btn._flash);
  btn._flash = setTimeout(() => {
    btn.classList.remove('is-copied');
    btn.textContent = original;
  }, 1600);
}

document.addEventListener('click', (e) => {
  const btn = e.target.closest('[data-copy]');
  if (!btn) return;
  const targetSel = btn.getAttribute('data-copy');
  const targetEl = targetSel.startsWith('#') || targetSel.startsWith('.')
    ? document.querySelector(targetSel)
    : null;
  const text = targetEl ? targetEl.textContent.trim() : btn.getAttribute('data-text') || '';
  if (!text) return;
  copyText(text).then(() => flashCopied(btn));
});
