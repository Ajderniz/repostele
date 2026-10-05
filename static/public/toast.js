// Shows #toast whenever its text changes (htmx OOB swaps), and once for a ?msg= query param.

const TOAST_MS = 4000;
let toastTimer = null;

function showToast() {
  const el = document.getElementById('toast');
  if (!el) return;
  const text = el.textContent.trim();
  el.hidden = !text;
  if (!text) return;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, TOAST_MS);
}

function toastFromQuery() {
  const params = new URLSearchParams(location.search);
  const msg = params.get('msg');
  if (!msg) return;
  const el = document.getElementById('toast');
  if (el) {
    el.textContent = msg;
    showToast();
  }
  params.delete('msg');
  const qs = params.toString();
  history.replaceState(null, '', location.pathname + (qs ? '?' + qs : ''));
}

document.addEventListener('DOMContentLoaded', function () {
  const wrap = document.getElementById('toast-wrap');
  if (wrap) {
    new MutationObserver(showToast).observe(wrap, { childList: true, subtree: true, characterData: true });
  }
  toastFromQuery();
});
