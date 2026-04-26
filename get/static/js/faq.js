// FAQ accordion. First item open by default; click toggles. Single-open behavior.
const items = document.querySelectorAll('.faq-item');

items.forEach((item, i) => {
  if (i === 0) item.classList.add('is-open');
  const toggle = item.querySelector('.faq-toggle');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    const wasOpen = item.classList.contains('is-open');
    items.forEach((it) => it.classList.remove('is-open'));
    if (!wasOpen) item.classList.add('is-open');
  });
});
