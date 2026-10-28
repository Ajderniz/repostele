// htmx defaults for this app: send the CSRF header on writes, and swap error responses too.

function readCookie(name) {
  const match = document.cookie.match('(^|;)\\s*' + name + '\\s*=\\s*([^;]+)');
  return match ? match.pop() : '';
}

document.body.addEventListener('htmx:configRequest', function (e) {
  if (!['get', 'head', 'options'].includes(e.detail.verb)) {
    e.detail.headers['X-CSRF-Token'] = readCookie('csrf-token');
  }
});

document.body.addEventListener('htmx:beforeSwap', function (e) {
  if (e.detail.xhr.status >= 400) e.detail.shouldSwap = true;
});
