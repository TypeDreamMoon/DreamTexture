import{$t as e,A as t,At as n,B as r,C as i,Dt as a,L as o,M as s,Mt as c,Nt as l,On as u,Ot as d,Pn as f,U as p,Xt as m,Y as h,Yt as g,Zt as _,d as v,g as y,gn as b,hn as x,j as S,jn as C,jt as w,k as T,m as E,nn as D,p as O,q as k,x as A}from"./client-DaoK0eww.js";function j(e){let{textColor2:t,primaryColorHover:n,primaryColorPressed:r,primaryColor:i,infoColor:a,successColor:o,warningColor:c,errorColor:l,baseColor:u,borderColor:d,opacityDisabled:f,tagColor:p,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,borderRadiusSmall:_,fontSizeMini:v,fontSizeTiny:y,fontSizeSmall:b,fontSizeMedium:x,heightMini:S,heightTiny:C,heightSmall:w,heightMedium:T,closeColorHover:D,closeColorPressed:O,buttonColor2Hover:k,buttonColor2Pressed:A,fontWeightStrong:j}=e;return{...E,closeBorderRadius:_,heightTiny:S,heightSmall:C,heightMedium:w,heightLarge:T,borderRadius:_,opacityDisabled:f,fontSizeTiny:v,fontSizeSmall:y,fontSizeMedium:b,fontSizeLarge:x,fontWeightStrong:j,textColorCheckable:t,textColorHoverCheckable:t,textColorPressedCheckable:t,textColorChecked:u,colorCheckable:`#0000`,colorHoverCheckable:k,colorPressedCheckable:A,colorChecked:i,colorCheckedHover:n,colorCheckedPressed:r,border:`1px solid ${d}`,textColor:t,color:p,colorBordered:`rgb(250, 250, 252)`,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,closeColorHover:D,closeColorPressed:O,borderPrimary:`1px solid ${s(i,{alpha:.3})}`,textColorPrimary:i,colorPrimary:s(i,{alpha:.12}),colorBorderedPrimary:s(i,{alpha:.1}),closeIconColorPrimary:i,closeIconColorHoverPrimary:i,closeIconColorPressedPrimary:i,closeColorHoverPrimary:s(i,{alpha:.12}),closeColorPressedPrimary:s(i,{alpha:.18}),borderInfo:`1px solid ${s(a,{alpha:.3})}`,textColorInfo:a,colorInfo:s(a,{alpha:.12}),colorBorderedInfo:s(a,{alpha:.1}),closeIconColorInfo:a,closeIconColorHoverInfo:a,closeIconColorPressedInfo:a,closeColorHoverInfo:s(a,{alpha:.12}),closeColorPressedInfo:s(a,{alpha:.18}),borderSuccess:`1px solid ${s(o,{alpha:.3})}`,textColorSuccess:o,colorSuccess:s(o,{alpha:.12}),colorBorderedSuccess:s(o,{alpha:.1}),closeIconColorSuccess:o,closeIconColorHoverSuccess:o,closeIconColorPressedSuccess:o,closeColorHoverSuccess:s(o,{alpha:.12}),closeColorPressedSuccess:s(o,{alpha:.18}),borderWarning:`1px solid ${s(c,{alpha:.35})}`,textColorWarning:c,colorWarning:s(c,{alpha:.15}),colorBorderedWarning:s(c,{alpha:.12}),closeIconColorWarning:c,closeIconColorHoverWarning:c,closeIconColorPressedWarning:c,closeColorHoverWarning:s(c,{alpha:.12}),closeColorPressedWarning:s(c,{alpha:.18}),borderError:`1px solid ${s(l,{alpha:.23})}`,textColorError:l,colorError:s(l,{alpha:.1}),colorBorderedError:s(l,{alpha:.08}),closeIconColorError:l,closeIconColorHoverError:l,closeIconColorPressedError:l,closeColorHoverError:s(l,{alpha:.12}),closeColorPressedError:s(l,{alpha:.18})}}var M={name:`Tag`,common:S,self:j},N={color:Object,type:{type:String,default:`default`},round:Boolean,size:String,closable:Boolean,disabled:{type:Boolean,default:void 0}},P=d(`tag`,`
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
 `),n(`border`,`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
 border: var(--n-border);
 transition: border-color .3s var(--n-bezier);
 `),n(`icon`,`
 display: flex;
 margin: 0 4px 0 0;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 font-size: var(--n-avatar-size-override);
 `),n(`avatar`,`
 display: flex;
 margin: 0 6px 0 0;
 `),n(`close`,`
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),w(`round`,`
 padding: 0 calc(var(--n-height) / 3);
 border-radius: calc(var(--n-height) / 2);
 `,[n(`icon`,`
 margin: 0 4px 0 calc((var(--n-height) - 8px) / -2);
 `),n(`avatar`,`
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
 `,[c(`disabled`,[a(`&:hover`,`background-color: var(--n-color-hover-checkable);`,[c(`checked`,`color: var(--n-text-color-hover-checkable);`)]),a(`&:active`,`background-color: var(--n-color-pressed-checkable);`,[c(`checked`,`color: var(--n-text-color-pressed-checkable);`)])]),w(`checked`,`
 color: var(--n-text-color-checked);
 background-color: var(--n-color-checked);
 `,[c(`disabled`,[a(`&:hover`,`background-color: var(--n-color-checked-hover);`),a(`&:active`,`background-color: var(--n-color-checked-pressed);`)])])])]),F=[`onClick`,`onMouseenter`,`onMouseleave`],I={...T.props,...N,bordered:{type:Boolean,default:void 0},checked:Boolean,checkable:Boolean,strong:Boolean,triggerClickOnClose:Boolean,onClose:[Array,Function],onMouseenter:Function,onMouseleave:Function,"onUpdate:checked":Function,onUpdateChecked:Function,internalCloseFocusable:{type:Boolean,default:!0},internalCloseIsButtonTag:{type:Boolean,default:!0},onCheckedChange:Function},L=h(`n-tag`),R=D({name:`Tag`,props:I,slots:Object,setup(e){let n=u(null),{mergedBorderedRef:r,mergedClsPrefixRef:a,inlineThemeDisabled:s,mergedRtlRef:c,mergedComponentPropsRef:d}=k(e),f=g(()=>e.size||d?.value?.Tag?.size||`medium`),p=T(`Tag`,`-tag`,P,M,e,a);b(L,{roundRef:C(e,`round`)});function m(){if(!e.disabled&&e.checkable){let{checked:t,onCheckedChange:n,onUpdateChecked:r,"onUpdate:checked":i}=e;r&&r(!t),i&&i(!t),n&&n(!t)}}function h(t){if(e.triggerClickOnClose||t.stopPropagation(),!e.disabled){let{onClose:n}=e;n&&i(n,t)}}let _={setTextContent(e){let{value:t}=n;t&&(t.textContent=e)}},v=y(`Tag`,c,a),x=g(()=>{let{type:t,color:{color:n,textColor:i}={}}=e,a=f.value,{common:{cubicBezierEaseInOut:s},self:{padding:c,closeMargin:u,borderRadius:d,opacityDisabled:m,textColorCheckable:h,textColorHoverCheckable:g,textColorPressedCheckable:_,textColorChecked:v,colorCheckable:y,colorHoverCheckable:b,colorPressedCheckable:x,colorChecked:S,colorCheckedHover:C,colorCheckedPressed:w,closeBorderRadius:T,fontWeightStrong:E,[l(`colorBordered`,t)]:D,[l(`closeSize`,a)]:O,[l(`closeIconSize`,a)]:k,[l(`fontSize`,a)]:A,[l(`height`,a)]:j,[l(`color`,t)]:M,[l(`textColor`,t)]:N,[l(`border`,t)]:P,[l(`closeIconColor`,t)]:F,[l(`closeIconColorHover`,t)]:I,[l(`closeIconColorPressed`,t)]:L,[l(`closeColorHover`,t)]:R,[l(`closeColorPressed`,t)]:z}}=p.value,B=o(u);return{"--n-font-weight-strong":E,"--n-avatar-size-override":`calc(${j} - 8px)`,"--n-bezier":s,"--n-border-radius":d,"--n-border":P,"--n-close-icon-size":k,"--n-close-color-pressed":z,"--n-close-color-hover":R,"--n-close-border-radius":T,"--n-close-icon-color":F,"--n-close-icon-color-hover":I,"--n-close-icon-color-pressed":L,"--n-close-icon-color-disabled":F,"--n-close-margin-top":B.top,"--n-close-margin-right":B.right,"--n-close-margin-bottom":B.bottom,"--n-close-margin-left":B.left,"--n-close-size":O,"--n-color":n||(r.value?D:M),"--n-color-checkable":y,"--n-color-checked":S,"--n-color-checked-hover":C,"--n-color-checked-pressed":w,"--n-color-hover-checkable":b,"--n-color-pressed-checkable":x,"--n-font-size":A,"--n-height":j,"--n-opacity-disabled":m,"--n-padding":c,"--n-text-color":i||N,"--n-text-color-checkable":h,"--n-text-color-checked":v,"--n-text-color-hover-checkable":g,"--n-text-color-pressed-checkable":_}}),S=s?t(`tag`,g(()=>{let t=``,{type:n,color:{color:i,textColor:a}={}}=e;return t+=n[0],t+=f.value[0],i&&(t+=`a${O(i)}`),a&&(t+=`b${O(a)}`),r.value&&(t+=`c`),t}),x,e):void 0;return{..._,rtlEnabled:v,mergedClsPrefix:a,contentRef:n,mergedBordered:r,handleClick:m,handleCloseClick:h,cssVars:s?void 0:x,themeClass:S?.themeClass,onRender:S?.onRender}},render(){let{mergedClsPrefix:t,rtlEnabled:n,closable:i,color:{borderColor:a}={},round:o,onRender:s,$slots:c}=this;s?.();let l=A(c.avatar,n=>n&&(x(),e(`div`,{class:r(`${t}-tag__avatar`)},[p(()=>n)],2))),u=A(c.icon,n=>n&&(x(),e(`div`,{class:r(`${t}-tag__icon`)},[p(()=>n)],2)));return x(),e(`div`,{class:r([`${t}-tag`,this.themeClass,{[`${t}-tag--rtl`]:n,[`${t}-tag--strong`]:this.strong,[`${t}-tag--disabled`]:this.disabled,[`${t}-tag--checkable`]:this.checkable,[`${t}-tag--checked`]:this.checkable&&this.checked,[`${t}-tag--round`]:o,[`${t}-tag--avatar`]:l,[`${t}-tag--icon`]:u,[`${t}-tag--closable`]:i}]),style:f(this.cssVars),onClick:this.handleClick,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},[p(()=>u||l),m(`span`,{class:r(`${t}-tag__content`),ref:`contentRef`},[p(()=>this.$slots.default?.())],2),!this.checkable&&i?(x(),_(v,{key:0,clsPrefix:t,class:r(`${t}-tag__close`),disabled:this.disabled,onClick:this.handleCloseClick,focusable:this.internalCloseFocusable,round:o,isButtonTag:this.internalCloseIsButtonTag,absolute:!0},null,8,[`clsPrefix`,`class`,`disabled`,`onClick`,`focusable`,`round`,`isButtonTag`])):p(()=>null),!this.checkable&&this.mergedBordered?(x(),e(`div`,{key:2,class:r(`${t}-tag__border`),style:f({borderColor:a})},null,6)):p(()=>null)],46,F)}});export{R as t};