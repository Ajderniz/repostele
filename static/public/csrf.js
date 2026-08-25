function gC(name){
const m=document.cookie.match('(^|;)\\s*'+name+'\\s*=\\s*([^;]+)');
return m?m.pop():'';
}
document.body.addEventListener('htmx:configRequest',(e)=>{
if(!['get','head','options'].includes(e.detail.verb)){
e.detail.headers['X-CSRF-Token']=gC('csrf-token');
}
});
