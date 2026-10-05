function notifParams(){const p=document.getElementById('notif-poll');return{token:p.dataset.token||'',silent:p.dataset.silent||''}}
const _ORDER_LIST_PATH=/^\/(order|dashboard\/orders)(\?|$)/;
const _ORDER_LISTS=['queue-list','orders-list'];
document.addEventListener('htmx:afterRequest',function(e){
  const el=e.detail.elt;
  if(el&&el.id==='notif-poll'){
    const token=e.detail.xhr&&e.detail.xhr.getResponseHeader('X-Notif-Token');
    if(token!==null)el.dataset.token=token;
    delete el.dataset.silent;
    return;
  }
  const cfg=e.detail.requestConfig||{};
  if(el&&el.classList&&el.classList.contains('order-entry')&&cfg.verb==='get'){
    document.querySelectorAll('.order-entry-selected').forEach(x=>x.classList.remove('order-entry-selected'));
    el.classList.add('order-entry-selected');
  }
  if(cfg.verb&&cfg.verb!=='get'&&e.detail.successful){
    const poll=document.getElementById('notif-poll');
    if(poll){poll.dataset.silent='1';htmx.trigger(poll,'notif-sync')}
    htmx.trigger(document.body,'counts-changed');
  }
  const tgt=e.detail.target;
  if(tgt&&_ORDER_LISTS.includes(tgt.id)){
    const path=(e.detail.pathInfo&&e.detail.pathInfo.finalRequestPath)||'';
    tgt.dataset.orderSrc=_ORDER_LIST_PATH.test(path)?path:'';
  }
});
document.addEventListener('order-changed',function(){
  _ORDER_LISTS.forEach(function(id){
    const el=document.getElementById(id);
    if(el&&el.dataset.orderSrc&&el.children.length){
      htmx.ajax('GET',el.dataset.orderSrc,{target:el,swap:'innerHTML'});
    }
  });
});
document.addEventListener('htmx:beforeSwap',function(e){
  const el=e.detail.requestConfig&&e.detail.requestConfig.elt;
  if(!el||!el.dataset.asideTarget)return;
  // Use the aside only where CSS shows it (container query), else fall back to inline detail.
  const aside=document.getElementById(el.dataset.asideTarget);
  if(aside&&getComputedStyle(aside).display==='none'){
    e.detail.target=document.getElementById('order-detail-'+el.dataset.id);
    e.detail.shouldSwap=true;
  }
});
