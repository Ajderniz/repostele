// Cart kept in localStorage. Drives the menu steppers, the nav badge, the cart page
// and checkout (SINPE receipt scan, then order POST).

const CART_KEY = 'cart';
const CART_MAX_QTY = 4;

function readCart() {
  try {
    return JSON.parse(localStorage.getItem(CART_KEY) || '{}');
  } catch (e) {
    return {};
  }
}

function writeCart(cart) {
  localStorage.setItem(CART_KEY, JSON.stringify(cart));
}

function cartCount(cart) {
  return Object.values(cart).reduce((n, it) => n + it.quant, 0);
}

function cartTotal(cart) {
  return Object.values(cart).reduce((t, it) => t + it.price * it.quant, 0);
}

function setText(id, text) {
  const el = document.getElementById(id);
  if (el) el.textContent = text;
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

// --- state changes ---

function setQuant(id, quant) {
  const cart = readCart();
  if (quant <= 0) delete cart[id];
  else if (cart[id]) cart[id].quant = Math.min(quant, CART_MAX_QTY);
  writeCart(cart);
  refreshCart();
}

function addToCart(id, name, price) {
  const cart = readCart();
  if (cart[id]) cart[id].quant = Math.min(cart[id].quant + 1, CART_MAX_QTY);
  else cart[id] = { id: Number(id), name: name, price: Number(price), quant: 1 };
  writeCart(cart);
  refreshCart();
}

function refreshCart() {
  const cart = readCart();
  updateNavBadge(cart);
  syncMenu(cart);
  renderCart(cart);
}

// --- views ---

function updateNavBadge(cart) {
  const badge = document.getElementById('cart-count');
  if (!badge) return;
  const n = cartCount(cart);
  badge.textContent = n > 0 ? n : '';
  badge.hidden = n === 0;
}

function syncMenu(cart) {
  const n = cartCount(cart);
  const total = cartTotal(cart);

  document.querySelectorAll('[data-menu-q]').forEach(q => {
    const it = cart[q.dataset.menuQ];
    q.textContent = it ? it.quant : 0;
  });
  document.querySelectorAll('[data-menu-step="1"]').forEach(btn => {
    const it = cart[btn.dataset.id];
    btn.disabled = !!it && it.quant >= CART_MAX_QTY;
  });

  const label = n + (n === 1 ? ' ítem' : ' ítems');
  setText('menu-cart-count', label);
  setText('menu-cart-total', '₡' + total);
  const bar = document.getElementById('menu-cart-bar');
  if (bar) {
    bar.hidden = n === 0;
    setText('menu-cart-bar-text', label + ' · ₡' + total);
  }
}

function renderCart(cart) {
  const list = document.getElementById('cart-list');
  if (!list) return;
  const ids = Object.keys(cart);

  list.replaceChildren();
  if (ids.length === 0) {
    list.append(el('p', 'empty-msg', 'El carrito está vacío'));
  } else {
    ids.forEach(id => list.append(cartRow(id, cart[id])));
  }
  setText('cart-total', '₡' + cartTotal(cart));
  updateContinueState(cart);
}

function cartRow(id, it) {
  const row = el('div', 'cart-row');

  const qty = el('span', 'cart-row-qty');
  const minus = el('button', 'btn btn-compact', '−');
  minus.type = 'button';
  minus.onclick = () => setQuant(id, it.quant - 1);
  const plus = el('button', 'btn btn-compact', '+');
  plus.type = 'button';
  plus.disabled = it.quant >= CART_MAX_QTY;
  plus.onclick = () => setQuant(id, it.quant + 1);
  qty.append(minus, el('span', 'cart-row-qty-num', String(it.quant)), plus);

  const remove = el('button', 'btn btn-compact', 'Quitar');
  remove.type = 'button';
  remove.onclick = () => setQuant(id, 0);

  row.append(
    el('span', 'cart-row-name', it.name),
    el('span', 'cart-row-price', '₡' + it.price),
    qty,
    el('span', 'cart-row-subtotal', '₡' + it.price * it.quant),
    remove,
  );
  return row;
}

// --- checkout ---

function refNumInput() {
  return document.getElementById('ref-num-input');
}

function refNumValid() {
  const input = refNumInput();
  return !!input && /^[0-9]{25}$/.test(input.value);
}

function updateContinueState(cart = readCart()) {
  const btn = document.getElementById('cart-continue');
  if (!btn) return;
  const n = cartCount(cart);
  const ok = refNumValid();

  btn.disabled = n === 0 || !ok;
  const step = n === 0 ? 1 : ok ? 3 : 2;
  [1, 2, 3].forEach(i => {
    const s = document.getElementById('cart-step-' + i);
    if (s) s.className = i === step ? 'on' : i < step ? 'done' : '';
  });

  const missing = document.getElementById('cart-missing');
  if (missing) {
    let text = '';
    if (n === 0) text = 'Agregue ítems desde el menú para continuar.';
    else if (!ok) text = 'Falta el comprobante SINPE: suba la captura o ingrese el código de 25 dígitos.';
    missing.textContent = text;
    missing.hidden = !text;
  }
}

function showRefPreview(file) {
  const img = document.getElementById('ref-preview');
  if (!img) return;
  if (img.src && img.src.startsWith('blob:')) URL.revokeObjectURL(img.src);
  img.src = URL.createObjectURL(file);
  img.hidden = false;
}

// OCR is ~60 KB, so it loads only when a receipt is picked.
let tesseractLoading = null;
function loadTesseract() {
  if (window.Tesseract) return Promise.resolve();
  if (!tesseractLoading) {
    tesseractLoading = new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = '/tesseract.min.js';
      script.onload = resolve;
      script.onerror = () => {
        tesseractLoading = null; // allow a retry on the next pick
        reject(new Error('tesseract load failed'));
      };
      document.head.append(script);
    });
  }
  return tesseractLoading;
}

function extractRefNum(text) {
  const runs = text.match(/[0-9 ]{20,}/g) || [];
  for (const run of runs) {
    const digits = run.replace(/\s/g, '');
    if (digits.length === 25) return digits;
  }
  const all = text.replace(/\D/g, '');
  return all.length === 25 ? all : '';
}

async function scanRefImg(file) {
  showRefPreview(file);
  const statusEl = document.getElementById('ref-status');
  const confirmBox = document.getElementById('ref-confirm');
  const input = refNumInput();
  if (!statusEl || !confirmBox || !input) return;

  statusEl.hidden = false;
  statusEl.textContent = 'Escaneando comprobante...';
  confirmBox.hidden = true;
  updateContinueState();

  try {
    await loadTesseract();
    const { data: { text } } = await Tesseract.recognize(file, 'eng');
    const code = extractRefNum(text);
    input.value = code;
    confirmBox.hidden = false;
    statusEl.textContent = code
      ? 'Código detectado. Verifique antes de enviar.'
      : 'No se detectó código. Ingréselo manualmente.';
  } catch (e) {
    confirmBox.hidden = false;
    statusEl.textContent = 'No se pudo leer la imagen. Ingrese el código manualmente.';
  }
  updateContinueState();
}

async function submitOrder() {
  if (!refNumValid()) return;
  const cart = readCart();
  if (cartCount(cart) === 0) return;

  const items = {};
  Object.values(cart).forEach(it => { items[it.id] = it.quant; });
  const btn = document.getElementById('cart-continue');
  const result = document.getElementById('order-result');
  btn.disabled = true;

  try {
    const res = await fetch('/order', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': readCookie('csrf-token'),
        'HX-Request': 'true',
      },
      body: JSON.stringify({ ref_num: refNumInput().value, items: items }),
    });
    const body = await res.text();
    if (result) result.innerHTML = body;
    if (res.ok) {
      localStorage.removeItem(CART_KEY);
      refreshCart();
      notifyWrite(); // raw fetch skips htmx:afterRequest, so sync notifs here
    } else {
      updateContinueState();
    }
  } catch (e) {
    if (result) result.innerHTML = '<div class="hx-flash"><p class="hx-msg">No se pudo enviar la orden.</p></div>';
    updateContinueState();
  }
}

// --- events ---

document.body.addEventListener('click', function (e) {
  const menuStep = e.target.closest('[data-menu-step]');
  if (menuStep) {
    const id = menuStep.dataset.id;
    if (Number(menuStep.dataset.menuStep) > 0) {
      addToCart(id, menuStep.dataset.itemName, menuStep.dataset.itemPrice);
    } else {
      const it = readCart()[id];
      setQuant(id, (it ? it.quant : 0) - 1);
    }
    return;
  }

  const cont = e.target.closest('#cart-continue');
  if (cont && !cont.disabled) submitOrder();
});

document.body.addEventListener('change', function (e) {
  if (e.target.id !== 'ref-img-input') return;
  const file = e.target.files && e.target.files[0];
  if (file) scanRefImg(file);
});

document.body.addEventListener('input', function (e) {
  if (e.target.id === 'ref-num-input') updateContinueState();
});

document.body.addEventListener('htmx:load', refreshCart);
