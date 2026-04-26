// Animated agent chat panels — three scripted conversations that loop.
// Mounts into <div data-chat-panel="0|1|2"></div>.
// Each script: user msg → tool calls → assistant reply → idle cursor, repeat.

const SCRIPTS = [
  {
    app: 'Claude', model: 'sonnet-4-5',
    user: 'set up a VM for my python script',
    tools: [
      { at: 1.4, name: 'huddle01:vm_create', args: 'class: "cpu-perf", region: "auto"', result: 'vm_b7p2 · 8 vCPU · 32 GB · ubuntu-22.04 · ready in 41s' },
      { at: 3.2, name: 'huddle01:vm_deploy', args: 'vm: "vm_b7p2", source: "./script.py"', result: 'live at b7p2.huddle01.run' },
    ],
    reply: { at: 4.6, text: 'Done. Your script is running on **vm_b7p2** (8 vCPU, 32 GB) and reachable at **b7p2.huddle01.run**.' },
  },
  {
    app: 'Cursor', model: 'sonnet-4-5',
    user: 'deploy this Next.js repo to a fresh box',
    tools: [
      { at: 1.4, name: 'huddle01:vm_create', args: 'class: "compute+", region: "ap-south-1"', result: 'vm_q44k · 16 vCPU · 64 GB · NVMe' },
      { at: 3.2, name: 'huddle01:repo_deploy', args: 'vm: "vm_q44k", repo: "."', result: 'cloned · pnpm build · TLS issued · pm2 saved' },
    ],
    reply: { at: 4.6, text: 'Live at **q44k.huddle01.run**. CI hook configured for `main` — pushes auto-redeploy.' },
  },
  {
    app: 'Claude', model: 'sonnet-4-5',
    user: 'how much have my VMs cost this week?',
    tools: [
      { at: 1.2, name: 'huddle01:usage_report', args: 'range: "7d"', result: '4 VMs · 18h 32m total · $4.27' },
    ],
    reply: { at: 3.0, text: 'You spent **$4.27** across 4 VMs this week. Biggest: **vm_x9p1** (compute+) at $3.66 — running 3h 40m.' },
  },
];

const LOOP_LENGTH = 7.5;

function escapeHtml(s) {
  return s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}

function renderInline(text) {
  // Match either a **bold** span or a `code` span, then escape the captured payload.
  return text.replace(/\*\*([^*]+)\*\*|`([^`]+)`/g, (_m, bold, code) => {
    if (bold !== undefined) return `<strong>${escapeHtml(bold)}</strong>`;
    return `<code>${escapeHtml(code)}</code>`;
  });
}

function hudlMark(size) {
  return `<svg width="${size}" height="${size}" viewBox="0 0 32 32" fill="currentColor" aria-hidden="true">
    <rect x="4" y="4" width="10" height="10" rx="1.5"/>
    <rect x="18" y="4" width="10" height="10" rx="1.5" opacity="0.55"/>
    <rect x="4" y="18" width="10" height="10" rx="1.5" opacity="0.7"/>
    <rect x="18" y="18" width="10" height="10" rx="5"/>
  </svg>`;
}

function buildPanel(script) {
  const tools = script.tools.map((tool, i) => `
    <div class="chat-msg chat-msg--tool" data-show-at="${tool.at}" data-tool-index="${i}">
      <div class="avatar avatar--bot">${hudlMark(12)}</div>
      <div class="chat-tool">
        <div class="chat-tool__head">
          <svg width="10" height="10" viewBox="0 0 10 10"><path d="M3 2 L6 5 L3 8" stroke="currentColor" stroke-width="1.2" fill="none" stroke-linecap="round" /></svg>
          <span class="label">Used tool</span>
          <span class="name">${escapeHtml(tool.name)}</span>
        </div>
        <div class="chat-tool__body">
          <div class="chat-tool__args">→ ${escapeHtml(tool.args)}</div>
          <div class="chat-tool__result">← ${escapeHtml(tool.result)}</div>
        </div>
      </div>
    </div>
  `).join('');

  return `
    <div class="chat-panel">
      <div class="chat-panel__chrome">
        <div class="chat-panel__lights"><span></span><span></span><span></span></div>
        <div class="chat-panel__title">${escapeHtml(script.app)}</div>
        <div class="chat-panel__model">${escapeHtml(script.model)}</div>
      </div>
      <div class="chat-panel__body">
        <div class="chat-msg chat-msg--user" data-show-at="0.2">
          <div class="bubble">${escapeHtml(script.user)}</div>
          <div class="avatar">U</div>
        </div>
        ${tools}
        <div class="chat-msg chat-msg--reply" data-show-at="${script.reply.at}">
          <div class="avatar avatar--bot">${hudlMark(12)}</div>
          <div class="chat-reply">${renderInline(script.reply.text)}</div>
        </div>
        <span class="chat-cursor" data-show-at="${script.reply.at + 0.8}"></span>
      </div>
    </div>
  `;
}

function startLoop(panel, offsetSeconds) {
  const messages = panel.querySelectorAll('[data-show-at]');
  const start = performance.now() + offsetSeconds * 1000;
  let raf;
  const tick = (now) => {
    const elapsed = ((now - start) / 1000) % LOOP_LENGTH;
    messages.forEach((el) => {
      const at = parseFloat(el.getAttribute('data-show-at'));
      el.classList.toggle('is-shown', elapsed >= at);
    });
    raf = requestAnimationFrame(tick);
  };
  raf = requestAnimationFrame(tick);
  return () => cancelAnimationFrame(raf);
}

document.querySelectorAll('[data-chat-panel]').forEach((slot) => {
  const i = parseInt(slot.getAttribute('data-chat-panel'), 10) || 0;
  const script = SCRIPTS[i] || SCRIPTS[0];
  slot.innerHTML = buildPanel(script);
  startLoop(slot, i * 0.8);
});
