function gC(name){
constm=document.cookie.match('(^|;)\\s*'+name+'\\s*=\\s*([^;]+)');
returnm?m.pop():'';
}
document.body.addEventListener('htmx:configRequest',(e)=>{
if(!['get','head','options'].includes(e.detail.verb)){
e.detail.headers['X-CSRF-Token']=gC('csrf-token');
}
});
