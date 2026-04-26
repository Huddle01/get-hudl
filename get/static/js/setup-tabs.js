// MCP install client switcher. Toggles active tab + shows the matching code slab.
const tabs = document.querySelectorAll('.tabs [data-tab]');
const slabs = document.querySelectorAll('[data-slab]');

function activate(id) {
  tabs.forEach((t) => t.classList.toggle('is-active', t.getAttribute('data-tab') === id));
  slabs.forEach((s) => {
    s.style.display = s.getAttribute('data-slab') === id ? 'block' : 'none';
  });
}

tabs.forEach((t) => {
  t.addEventListener('click', () => activate(t.getAttribute('data-tab')));
});

const initial = document.querySelector('.tabs [data-tab].is-active') || tabs[0];
if (initial) activate(initial.getAttribute('data-tab'));
