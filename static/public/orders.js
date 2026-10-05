// Order lists (staff queue, customer orders): highlight the picked order, open its
// detail in the side aside (inline when the aside is hidden), reload on order-changed.

const ORDER_LISTS = ['queue-list', 'orders-list', 'dash-panel'];
const ORDER_LIST_PATH = /^\/(order|dashboard\/orders(\/history)?)(\?|$)/;

document.addEventListener('htmx:afterRequest', function (e) {
  const el = e.detail.elt;
  const verb = (e.detail.requestConfig || {}).verb;
  if (el && el.classList && el.classList.contains('order-entry') && verb === 'get') {
    document.querySelectorAll('.order-entry-selected').forEach(x => x.classList.remove('order-entry-selected'));
    el.classList.add('order-entry-selected');
  }

  const target = e.detail.target;
  if (target && ORDER_LISTS.includes(target.id)) {
    const path = (e.detail.pathInfo && e.detail.pathInfo.finalRequestPath) || '';
    target.dataset.orderSrc = ORDER_LIST_PATH.test(path) ? path : '';
  }
});

document.addEventListener('order-changed', function () {
  ORDER_LISTS.forEach(function (id) {
    const list = document.getElementById(id);
    if (!list || !list.dataset.orderSrc || !list.children.length) return;
    // Reload replaces the rows (and any inline card); reopen the picked one after settle.
    const picked = list.querySelector('.order-entry-selected');
    list.dataset.pickedId = picked ? picked.id : '';
    htmx.ajax('GET', list.dataset.orderSrc, { target: list, swap: 'innerHTML' });
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
  if (!el || !el.dataset.asideTarget) return;
  // Use the aside only where CSS shows it (container query), else fall back to inline detail.
  const aside = document.getElementById(el.dataset.asideTarget);
  if (aside && getComputedStyle(aside).display === 'none') {
    e.detail.target = document.getElementById('order-detail-' + el.dataset.id);
    e.detail.shouldSwap = true;
  }
});
