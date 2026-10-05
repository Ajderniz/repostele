// Order lists (staff queue, customer orders, dashboard history). The picked order's
// detail goes to the side aside where CSS shows it; where it is hidden, the detail opens
// inline under the row and only one inline detail stays open per list.

// src: element whose dataset holds the list's request path; dest: element that gets the rows.
const ORDER_LISTS = [
  { src: 'queue-list', dest: 'queue-list' },
  { src: 'orders-list', dest: 'orders-list' },
  { src: 'dash-panel', dest: 'dash-list' },
];
const ORDER_LIST_PATH = /^\/(order|dashboard\/orders(\/history)?)(\?|$)/;

// The aside a row targets, or null when CSS hides it (narrow screens).
function shownAside(row) {
  const aside = row.dataset.asideTarget && document.getElementById(row.dataset.asideTarget);
  return aside && getComputedStyle(aside).display !== 'none' ? aside : null;
}

document.addEventListener('htmx:beforeRequest', function (e) {
  const el = e.detail.elt;
  if (!el || !el.classList || !el.classList.contains('order-entry') || !el.dataset.id) return;
  // Inline mode: clicking the open row closes it, no request.
  const inline = document.getElementById('order-detail-' + el.dataset.id);
  if (shownAside(el) || !inline || !inline.children.length) return;
  e.preventDefault();
  inline.replaceChildren();
  el.classList.remove('order-entry-selected');
});

document.addEventListener('htmx:afterRequest', function (e) {
  const el = e.detail.elt;
  const verb = (e.detail.requestConfig || {}).verb;
  if (el && el.classList && el.classList.contains('order-entry') && verb === 'get') {
    document.querySelectorAll('.order-entry-selected').forEach(x => x.classList.remove('order-entry-selected'));
    el.classList.add('order-entry-selected');
  }

  const target = e.detail.target;
  if (target && ORDER_LISTS.some(l => l.src === target.id)) {
    const path = (e.detail.pathInfo && e.detail.pathInfo.finalRequestPath) || '';
    target.dataset.orderSrc = ORDER_LIST_PATH.test(path) ? path : '';
  }
});

document.addEventListener('order-changed', function () {
  ORDER_LISTS.forEach(function ({ src, dest }) {
    const source = document.getElementById(src);
    const list = document.getElementById(dest);
    if (!source || !list || !source.dataset.orderSrc || !list.children.length) return;
    // Reload replaces the rows (and any inline card); reopen the picked one after settle.
    const picked = list.querySelector('.order-entry-selected');
    list.dataset.pickedId = picked ? picked.id : '';
    htmx.ajax('GET', source.dataset.orderSrc, { target: list, swap: 'innerHTML' });
  });
});

document.addEventListener('htmx:afterSettle', function (e) {
  const list = e.detail.target;
  if (!list || !list.dataset || !list.dataset.pickedId) return;
  const row = document.getElementById(list.dataset.pickedId);
  delete list.dataset.pickedId;
  if (row) row.click();
});

document.addEventListener('htmx:beforeSwap', function (e) {
  const el = (e.detail.requestConfig || {}).elt;
  if (!el || !el.dataset.asideTarget || shownAside(el)) return;
  // Narrow: detail goes inline. Close any other inline detail in the same list first.
  const target = document.getElementById('order-detail-' + el.dataset.id);
  if (!target) return;
  const list = el.parentElement;
  list.querySelectorAll('.order-entry-detail').forEach(d => { if (d !== target) d.replaceChildren(); });
  list.querySelectorAll('.order-entry-selected').forEach(r => { if (r !== el) r.classList.remove('order-entry-selected'); });
  e.detail.target = target;
  e.detail.shouldSwap = true;
});
