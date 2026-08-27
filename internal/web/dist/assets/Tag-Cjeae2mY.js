import{$t as e,A as t,An as n,At as r,B as i,C as a,Dn as o,Dt as s,L as c,M as l,Mt as u,Nn as d,Nt as f,Ot as p,U as m,Xt as h,Y as g,Yt as _,Zt as v,d as y,g as b,gn as x,hn as S,j as C,jt as w,k as T,m as E,nn as D,p as O,q as k,x as A}from"./client-Do8VSizq.js";function j(e){let{textColor2:t,primaryColorHover:n,primaryColorPressed:r,primaryColor:i,infoColor:a,successColor:o,warningColor:s,errorColor:c,baseColor:u,borderColor:d,opacityDisabled:f,tagColor:p,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,borderRadiusSmall:_,fontSizeMini:v,fontSizeTiny:y,fontSizeSmall:b,fontSizeMedium:x,heightMini:S,heightTiny:C,heightSmall:w,heightMedium:T,closeColorHover:D,closeColorPressed:O,buttonColor2Hover:k,buttonColor2Pressed:A,fontWeightStrong:j}=e;return{...E,closeBorderRadius:_,heightTiny:S,heightSmall:C,heightMedium:w,heightLarge:T,borderRadius:_,opacityDisabled:f,fontSizeTiny:v,fontSizeSmall:y,fontSizeMedium:b,fontSizeLarge:x,fontWeightStrong:j,textColorCheckable:t,textColorHoverCheckable:t,textColorPressedCheckable:t,textColorChecked:u,colorCheckable:`#0000`,colorHoverCheckable:k,colorPressedCheckable:A,colorChecked:i,colorCheckedHover:n,colorCheckedPressed:r,border:`1px solid ${d}`,textColor:t,color:p,colorBordered:`rgb(250, 250, 252)`,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,closeColorHover:D,closeColorPressed:O,borderPrimary:`1px solid ${l(i,{alpha:.3})}`,textColorPrimary:i,colorPrimary:l(i,{alpha:.12}),colorBorderedPrimary:l(i,{alpha:.1}),closeIconColorPrimary:i,closeIconColorHoverPrimary:i,closeIconColorPressedPrimary:i,closeColorHoverPrimary:l(i,{alpha:.12}),closeColorPressedPrimary:l(i,{alpha:.18}),borderInfo:`1px solid ${l(a,{alpha:.3})}`,textColorInfo:a,colorInfo:l(a,{alpha:.12}),colorBorderedInfo:l(a,{alpha:.1}),closeIconColorInfo:a,closeIconColorHoverInfo:a,closeIconColorPressedInfo:a,closeColorHoverInfo:l(a,{alpha:.12}),closeColorPressedInfo:l(a,{alpha:.18}),borderSuccess:`1px solid ${l(o,{alpha:.3})}`,textColorSuccess:o,colorSuccess:l(o,{alpha:.12}),colorBorderedSuccess:l(o,{alpha:.1}),closeIconColorSuccess:o,closeIconColorHoverSuccess:o,closeIconColorPressedSuccess:o,closeColorHoverSuccess:l(o,{alpha:.12}),closeColorPressedSuccess:l(o,{alpha:.18}),borderWarning:`1px solid ${l(s,{alpha:.35})}`,textColorWarning:s,colorWarning:l(s,{alpha:.15}),colorBorderedWarning:l(s,{alpha:.12}),closeIconColorWarning:s,closeIconColorHoverWarning:s,closeIconColorPressedWarning:s,closeColorHoverWarning:l(s,{alpha:.12}),closeColorPressedWarning:l(s,{alpha:.18}),borderError:`1px solid ${l(c,{alpha:.23})}`,textColorError:c,colorError:l(c,{alpha:.1}),colorBorderedError:l(c,{alpha:.08}),closeIconColorError:c,closeIconColorHoverError:c,closeIconColorPressedError:c,closeColorHoverError:l(c,{alpha:.12}),closeColorPressedError:l(c,{alpha:.18})}}var M={name:`Tag`,common:C,self:j},N={color:Object,type:{type:String,default:`default`},round:Boolean,size:String,closable:Boolean,disabled:{type:Boolean,default:void 0}},P=p(`tag`,`
 --n-close-margin: var(--n-close-margin-top) var(--n-close-margin-right) var(--n-close-margin-bottom) var(--n-close-margin-left);
 white-space: nowrap;
 position: relative;
 box-sizing: border-box;
 cursor: default;
 display: inline-flex;
 align-items: center;
 flex-wrap: nowrap;
 padding: var(--n-padding);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 line-height: 1;
 height: var(--n-height);
 font-size: var(--n-font-size);
`,[w(`strong`,`
 font-weight: var(--n-font-weight-strong);
 `),r(`border`,`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
 border: var(--n-border);
 transition: border-color .3s var(--n-bezier);
 `),r(`icon`,`
 display: flex;
 margin: 0 4px 0 0;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 font-size: var(--n-avatar-size-override);
 `),r(`avatar`,`
 display: flex;
 margin: 0 6px 0 0;
 `),r(`close`,`
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),w(`round`,`
 padding: 0 calc(var(--n-height) / 3);
 border-radius: calc(var(--n-height) / 2);
 `,[r(`icon`,`
 margin: 0 4px 0 calc((var(--n-height) - 8px) / -2);
 `),r(`avatar`,`
 margin: 0 6px 0 calc((var(--n-height) - 8px) / -2);
 `),w(`closable`,`
 padding: 0 calc(var(--n-height) / 4) 0 calc(var(--n-height) / 3);
 `)]),w(`icon, avatar`,[w(`round`,`
 padding: 0 calc(var(--n-height) / 3) 0 calc(var(--n-height) / 2);
 `)]),w(`disabled`,`
 cursor: not-allowed !important;
 opacity: var(--n-opacity-disabled);
 `),w(`checkable`,`
 cursor: pointer;
 box-shadow: none;
 color: var(--n-text-color-checkable);
 background-color: var(--n-color-checkable);
 `,[u(`disabled`,[s(`&:hover`,`background-color: var(--n-color-hover-checkable);`,[u(`checked`,`color: var(--n-text-color-hover-checkable);`)]),s(`&:active`,`background-color: var(--n-color-pressed-checkable);`,[u(`checked`,`color: var(--n-text-color-pressed-checkable);`)])]),w(`checked`,`
 color: var(--n-text-color-checked);
 background-color: var(--n-color-checked);
 `,[u(`disabled`,[s(`&:hover`,`background-color: var(--n-color-checked-hover);`),s(`&:active`,`background-color: var(--n-color-checked-pressed);`)])])])]),F=[`onClick`,`onMouseenter`,`onMouseleave`],I={...T.props,...N,bordered:{type:Boolean,default:void 0},checked:Boolean,checkable:Boolean,strong:Boolean,triggerClickOnClose:Boolean,onClose:[Array,Function],onMouseenter:Function,onMouseleave:Function,"onUpdate:checked":Function,onUpdateChecked:Function,internalCloseFocusable:{type:Boolean,default:!0},internalCloseIsButtonTag:{type:Boolean,default:!0},onCheckedChange:Function},L=g(`n-tag`),R=D({name:`Tag`,props:I,slots:Object,setup(e){let r=o(null),{mergedBorderedRef:i,mergedClsPrefixRef:s,inlineThemeDisabled:l,mergedRtlRef:u,mergedComponentPropsRef:d}=k(e),p=_(()=>e.size||d?.value?.Tag?.size||`medium`),m=T(`Tag`,`-tag`,P,M,e,s);x(L,{roundRef:n(e,`round`)});function h(){if(!e.disabled&&e.checkable){let{checked:t,onCheckedChange:n,onUpdateChecked:r,"onUpdate:checked":i}=e;r&&r(!t),i&&i(!t),n&&n(!t)}}function g(t){if(e.triggerClickOnClose||t.stopPropagation(),!e.disabled){let{onClose:n}=e;n&&a(n,t)}}let v={setTextContent(e){let{value:t}=r;t&&(t.textContent=e)}},y=b(`Tag`,u,s),S=_(()=>{let{type:t,color:{color:n,textColor:r}={}}=e,a=p.value,{common:{cubicBezierEaseInOut:o},self:{padding:s,closeMargin:l,borderRadius:u,opacityDisabled:d,textColorCheckable:h,textColorHoverCheckable:g,textColorPressedCheckable:_,textColorChecked:v,colorCheckable:y,colorHoverCheckable:b,colorPressedCheckable:x,colorChecked:S,colorCheckedHover:C,colorCheckedPressed:w,closeBorderRadius:T,fontWeightStrong:E,[f(`colorBordered`,t)]:D,[f(`closeSize`,a)]:O,[f(`closeIconSize`,a)]:k,[f(`fontSize`,a)]:A,[f(`height`,a)]:j,[f(`color`,t)]:M,[f(`textColor`,t)]:N,[f(`border`,t)]:P,[f(`closeIconColor`,t)]:F,[f(`closeIconColorHover`,t)]:I,[f(`closeIconColorPressed`,t)]:L,[f(`closeColorHover`,t)]:R,[f(`closeColorPressed`,t)]:z}}=m.value,B=c(l);return{"--n-font-weight-strong":E,"--n-avatar-size-override":`calc(${j} - 8px)`,"--n-bezier":o,"--n-border-radius":u,"--n-border":P,"--n-close-icon-size":k,"--n-close-color-pressed":z,"--n-close-color-hover":R,"--n-close-border-radius":T,"--n-close-icon-color":F,"--n-close-icon-color-hover":I,"--n-close-icon-color-pressed":L,"--n-close-icon-color-disabled":F,"--n-close-margin-top":B.top,"--n-close-margin-right":B.right,"--n-close-margin-bottom":B.bottom,"--n-close-margin-left":B.left,"--n-close-size":O,"--n-color":n||(i.value?D:M),"--n-color-checkable":y,"--n-color-checked":S,"--n-color-checked-hover":C,"--n-color-checked-pressed":w,"--n-color-hover-checkable":b,"--n-color-pressed-checkable":x,"--n-font-size":A,"--n-height":j,"--n-opacity-disabled":d,"--n-padding":s,"--n-text-color":r||N,"--n-text-color-checkable":h,"--n-text-color-checked":v,"--n-text-color-hover-checkable":g,"--n-text-color-pressed-checkable":_}}),C=l?t(`tag`,_(()=>{let t=``,{type:n,color:{color:r,textColor:a}={}}=e;return t+=n[0],t+=p.value[0],r&&(t+=`a${O(r)}`),a&&(t+=`b${O(a)}`),i.value&&(t+=`c`),t}),S,e):void 0;return{...v,rtlEnabled:y,mergedClsPrefix:s,contentRef:r,mergedBordered:i,handleClick:h,handleCloseClick:g,cssVars:l?void 0:S,themeClass:C?.themeClass,onRender:C?.onRender}},render(){let{mergedClsPrefix:t,rtlEnabled:n,closable:r,color:{borderColor:a}={},round:o,onRender:s,$slots:c}=this;s?.();let l=A(c.avatar,n=>n&&(S(),e(`div`,{class:i(`${t}-tag__avatar`)},[m(()=>n)],2))),u=A(c.icon,n=>n&&(S(),e(`div`,{class:i(`${t}-tag__icon`)},[m(()=>n)],2)));return S(),e(`div`,{class:i([`${t}-tag`,this.themeClass,{[`${t}-tag--rtl`]:n,[`${t}-tag--strong`]:this.strong,[`${t}-tag--disabled`]:this.disabled,[`${t}-tag--checkable`]:this.checkable,[`${t}-tag--checked`]:this.checkable&&this.checked,[`${t}-tag--round`]:o,[`${t}-tag--avatar`]:l,[`${t}-tag--icon`]:u,[`${t}-tag--closable`]:r}]),style:d(this.cssVars),onClick:this.handleClick,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},[m(()=>u||l),h(`span`,{class:i(`${t}-tag__content`),ref:`contentRef`},[m(()=>this.$slots.default?.())],2),!this.checkable&&r?(S(),v(y,{key:0,clsPrefix:t,class:i(`${t}-tag__close`),disabled:this.disabled,onClick:this.handleCloseClick,focusable:this.internalCloseFocusable,round:o,isButtonTag:this.internalCloseIsButtonTag,absolute:!0},null,8,[`clsPrefix`,`class`,`disabled`,`onClick`,`focusable`,`round`,`isButtonTag`])):m(()=>null),!this.checkable&&this.mergedBordered?(S(),e(`div`,{key:2,class:i(`${t}-tag__border`),style:d({borderColor:a})},null,6)):m(()=>null)],46,F)}});export{R as t};