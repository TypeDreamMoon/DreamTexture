import{$t as e,A as t,At as n,B as r,D as i,Dt as a,L as o,M as s,N as c,Nt as l,On as u,Ot as d,U as f,Xt as p,Yt as m,Zt as h,d as g,g as _,hn as v,j as y,jt as b,k as x,nn as S,on as C,q as w,x as T,y as E}from"./client-HeelZ21K.js";import{B as D,H as O,I as k,L as A,R as j,V as M,z as N}from"./index-DWif4xih.js";function P(e){let{lineHeight:t,borderRadius:n,fontWeightStrong:r,baseColor:i,dividerColor:a,actionColor:o,textColor1:l,textColor2:u,closeColorHover:d,closeColorPressed:f,closeIconColor:p,closeIconColorHover:m,closeIconColorPressed:h,infoColor:g,successColor:_,warningColor:v,errorColor:y,fontSize:b}=e;return{...O,fontSize:b,lineHeight:t,titleFontWeight:r,borderRadius:n,border:`1px solid ${a}`,color:o,titleTextColor:l,iconColor:u,contentTextColor:u,closeBorderRadius:n,closeColorHover:d,closeColorPressed:f,closeIconColor:p,closeIconColorHover:m,closeIconColorPressed:h,borderInfo:`1px solid ${c(i,s(g,{alpha:.25}))}`,colorInfo:c(i,s(g,{alpha:.08})),titleTextColorInfo:l,iconColorInfo:g,contentTextColorInfo:u,closeColorHoverInfo:d,closeColorPressedInfo:f,closeIconColorInfo:p,closeIconColorHoverInfo:m,closeIconColorPressedInfo:h,borderSuccess:`1px solid ${c(i,s(_,{alpha:.25}))}`,colorSuccess:c(i,s(_,{alpha:.08})),titleTextColorSuccess:l,iconColorSuccess:_,contentTextColorSuccess:u,closeColorHoverSuccess:d,closeColorPressedSuccess:f,closeIconColorSuccess:p,closeIconColorHoverSuccess:m,closeIconColorPressedSuccess:h,borderWarning:`1px solid ${c(i,s(v,{alpha:.33}))}`,colorWarning:c(i,s(v,{alpha:.08})),titleTextColorWarning:l,iconColorWarning:v,contentTextColorWarning:u,closeColorHoverWarning:d,closeColorPressedWarning:f,closeIconColorWarning:p,closeIconColorHoverWarning:m,closeIconColorPressedWarning:h,borderError:`1px solid ${c(i,s(y,{alpha:.25}))}`,colorError:c(i,s(y,{alpha:.08})),titleTextColorError:l,iconColorError:y,contentTextColorError:u,closeColorHoverError:d,closeColorPressedError:f,closeIconColorError:p,closeIconColorHoverError:m,closeIconColorPressedError:h}}var F={name:`Alert`,common:y,self:P},I=d(`alert`,`
 line-height: var(--n-line-height);
 border-radius: var(--n-border-radius);
 position: relative;
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-color);
 text-align: start;
 word-break: break-word;
`,[n(`border`,`
 border-radius: inherit;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 transition: border-color .3s var(--n-bezier);
 border: var(--n-border);
 pointer-events: none;
 `),b(`closable`,[d(`alert-body`,[n(`title`,`
 padding-right: 24px;
 `)])]),n(`icon`,{color:`var(--n-icon-color)`}),d(`alert-body`,{padding:`var(--n-padding)`},[n(`title`,{color:`var(--n-title-text-color)`}),n(`content`,{color:`var(--n-content-text-color)`})]),k({originalTransition:`transform .3s var(--n-bezier)`,enterToProps:{transform:`scale(1)`},leaveToProps:{transform:`scale(0.9)`}}),n(`icon`,`
 position: absolute;
 left: 0;
 top: 0;
 align-items: center;
 justify-content: center;
 display: flex;
 width: var(--n-icon-size);
 height: var(--n-icon-size);
 font-size: var(--n-icon-size);
 margin: var(--n-icon-margin);
 `),n(`close`,`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 position: absolute;
 right: 0;
 top: 0;
 margin: var(--n-close-margin);
 `),b(`show-icon`,[d(`alert-body`,{paddingLeft:`calc(var(--n-icon-margin-left) + var(--n-icon-size) + var(--n-icon-margin-right))`})]),b(`right-adjust`,[d(`alert-body`,{paddingRight:`calc(var(--n-close-size) + var(--n-padding) + 2px)`})]),d(`alert-body`,`
 border-radius: var(--n-border-radius);
 transition: border-color .3s var(--n-bezier);
 `,[n(`title`,`
 transition: color .3s var(--n-bezier);
 font-size: 16px;
 line-height: 19px;
 font-weight: var(--n-title-font-weight);
 `,[a(`& +`,[n(`content`,{marginTop:`9px`})])]),n(`content`,{transition:`color .3s var(--n-bezier)`,fontSize:`var(--n-font-size)`})]),n(`icon`,{transition:`color .3s var(--n-bezier)`})]),L={...x.props,title:String,showIcon:{type:Boolean,default:!0},type:{type:String,default:`default`},bordered:{type:Boolean,default:!0},closable:Boolean,onClose:Function,onAfterLeave:Function,onAfterHide:Function},R=S({name:`Alert`,inheritAttrs:!1,props:L,slots:Object,setup(e){let{mergedClsPrefixRef:n,mergedBorderedRef:r,inlineThemeDisabled:i,mergedRtlRef:a}=w(e),s=x(`Alert`,`-alert`,I,F,e,n),c=_(`Alert`,a,n),d=m(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=s.value,{fontSize:r,borderRadius:i,titleFontWeight:a,lineHeight:c,iconSize:u,iconMargin:d,iconMarginRtl:f,closeIconSize:p,closeBorderRadius:m,closeSize:h,closeMargin:g,closeMarginRtl:_,padding:v}=n,{type:y}=e,{left:b,right:x}=o(d);return{"--n-bezier":t,"--n-color":n[l(`color`,y)],"--n-close-icon-size":p,"--n-close-border-radius":m,"--n-close-color-hover":n[l(`closeColorHover`,y)],"--n-close-color-pressed":n[l(`closeColorPressed`,y)],"--n-close-icon-color":n[l(`closeIconColor`,y)],"--n-close-icon-color-hover":n[l(`closeIconColorHover`,y)],"--n-close-icon-color-pressed":n[l(`closeIconColorPressed`,y)],"--n-icon-color":n[l(`iconColor`,y)],"--n-border":n[l(`border`,y)],"--n-title-text-color":n[l(`titleTextColor`,y)],"--n-content-text-color":n[l(`contentTextColor`,y)],"--n-line-height":c,"--n-border-radius":i,"--n-font-size":r,"--n-title-font-weight":a,"--n-icon-size":u,"--n-icon-margin":d,"--n-icon-margin-rtl":f,"--n-close-size":h,"--n-close-margin":g,"--n-close-margin-rtl":_,"--n-padding":v,"--n-icon-margin-left":b,"--n-icon-margin-right":x}}),f=i?t(`alert`,m(()=>e.type[0]),d,e):void 0,p=u(!0),h=()=>{let{onAfterLeave:t,onAfterHide:n}=e;t&&t(),n&&n()};return{rtlEnabled:c,mergedClsPrefix:n,mergedBordered:r,visible:p,handleCloseClick:()=>{Promise.resolve(e.onClose?.()).then(e=>{e!==!1&&(p.value=!1)})},handleAfterLeave:()=>{h()},mergedTheme:s,cssVars:i?void 0:d,themeClass:f?.themeClass,onRender:f?.onRender}},render(){return this.onRender?.(),v(),h(A,{onAfterLeave:this.handleAfterLeave},{default:()=>{let{mergedClsPrefix:t,$slots:n}=this,a={class:[`${t}-alert`,this.themeClass,this.closable&&`${t}-alert--closable`,this.showIcon&&`${t}-alert--show-icon`,!this.title&&this.closable&&`${t}-alert--right-adjust`,this.rtlEnabled&&`${t}-alert--rtl`],style:this.cssVars,role:`alert`};return this.visible?(v(),e(`div`,C({key:1},C(this.$attrs,a)),[f(()=>this.closable&&(v(),h(g,{clsPrefix:t,class:r(`${t}-alert__close`),onClick:this.handleCloseClick},null,8,[`clsPrefix`,`class`,`onClick`]))),f(()=>this.bordered&&(v(),e(`div`,{class:r(`${t}-alert__border`)},null,2))),f(()=>this.showIcon&&(v(),e(`div`,{class:r(`${t}-alert__icon`),"aria-hidden":`true`},[f(()=>E(n.icon,()=>[(v(),h(i,{clsPrefix:t},{default:()=>{switch(this.type){case`success`:return v(),h(N,{key:3});case`info`:return v(),h(D,{key:4});case`warning`:return v(),h(j,{key:5});case`error`:return v(),h(M,{key:6});default:return null}}},1032,[`clsPrefix`]))]))],2))),p(`div`,{class:r([`${t}-alert-body`,this.mergedBordered&&`${t}-alert-body--bordered`])},[f(()=>T(n.header,n=>{let i=n||this.title;return i?(v(),e(`div`,{key:2,class:r(`${t}-alert-body__title`)},[f(()=>i)],2)):null})),f(()=>n.default&&(v(),e(`div`,{class:r(`${t}-alert-body__content`)},[f(()=>n.default())],2)))],2)],16)):null}},1032,[`onAfterLeave`])}});export{R as t};