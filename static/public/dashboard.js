// Panel tabs: one tab stays "on". Content loads through each tab's own hx-get.

document.body.addEventListener('click', function (e) {
  const tab = e.target.closest('.dash-tab');
  if (!tab) return;
  tab.parentNode.querySelectorAll('.dash-tab').forEach(t => t.classList.toggle('on', t === tab));
});
