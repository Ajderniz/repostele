// Polling for #notif-poll. Keeps the server token so unchanged polls return 204.
// After own writes, a silent sync refreshes the notif state without a toast.

function notifParams() {
  const poll = document.getElementById('notif-poll');
  if (!poll) return { token: '', silent: '' };
  return { token: poll.dataset.token || '', silent: poll.dataset.silent || '' };
}

// Call after any write. Silent poll sync (no toast for own action), then the same
// order-changed / counts-changed events a poll change would fire.
function notifyWrite() {
  const poll = document.getElementById('notif-poll');
  if (poll) {
    poll.dataset.silent = '1';
    htmx.trigger(poll, 'notif-sync');
  }
  htmx.trigger(document.body, 'order-changed');
  htmx.trigger(document.body, 'counts-changed');
}

document.addEventListener('htmx:afterRequest', function (e) {
  const el = e.detail.elt;
  if (el && el.id === 'notif-poll') {
    const token = e.detail.xhr && e.detail.xhr.getResponseHeader('X-Notif-Token');
    if (token !== null) el.dataset.token = token;
    delete el.dataset.silent;
    return;
  }

  const verb = (e.detail.requestConfig || {}).verb;
  if (verb && verb !== 'get' && e.detail.successful) notifyWrite();
});
