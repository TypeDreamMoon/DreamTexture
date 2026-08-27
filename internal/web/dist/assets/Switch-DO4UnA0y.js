import{$t as e,A as t,An as n,At as r,B as i,C as a,Dn as o,Dt as s,I as c,M as l,Mt as u,Nn as d,Nt as f,Ot as p,R as m,U as h,Xt as g,Yt as _,Zt as v,c as y,hn as b,j as x,jt as S,k as C,l as w,nn as T,o as E,on as D,q as O,u as k,v as A,x as j}from"./client-c5jocXoO.js";import{m as M,tt as N}from"./index-Diy25eYw.js";function P(e){let{primaryColor:t,opacityDisabled:n,borderRadius:r,textColor3:i}=e;return{...M,iconColor:i,textColor:`white`,loadingColor:t,opacityDisabled:n,railColor:`rgba(0, 0, 0, .14)`,railColorActive:t,buttonBoxShadow:`0 1px 4px 0 rgba(0, 0, 0, 0.3), inset 0 0 1px 0 rgba(0, 0, 0, 0.05)`,buttonColor:`#FFF`,railBorderRadiusSmall:r,railBorderRadiusMedium:r,railBorderRadiusLarge:r,buttonBorderRadiusSmall:r,buttonBorderRadiusMedium:r,buttonBorderRadiusLarge:r,boxShadowFocus:`0 0 0 2px ${l(t,{alpha:.2})}`}}var F={name:`Switch`,common:x,self:P},I=p(`switch`,`
 height: var(--n-height);
 min-width: var(--n-width);
 vertical-align: middle;
 user-select: none;
 -webkit-user-select: none;
 display: inline-flex;
 outline: none;
 justify-content: center;
 align-items: center;
`,[r(`children-placeholder`,`
 height: var(--n-rail-height);
 display: flex;
 flex-direction: column;
 overflow: hidden;
 pointer-events: none;
 visibility: hidden;
 `),r(`rail-placeholder`,`
 display: flex;
 flex-wrap: none;
 `),r(`button-placeholder`,`
 width: calc(1.75 * var(--n-rail-height));
 height: var(--n-rail-height);
 `),p(`base-loading`,`
 position: absolute;
 top: 50%;
 left: 50%;
 transform: translateX(-50%) translateY(-50%);
 font-size: calc(var(--n-button-width) - 4px);
 color: var(--n-loading-color);
 transition: color .3s var(--n-bezier);
 `,[y({left:`50%`,top:`50%`,originalTransform:`translateX(-50%) translateY(-50%)`})]),r(`checked, unchecked`,`
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 box-sizing: border-box;
 position: absolute;
 white-space: nowrap;
 top: 0;
 bottom: 0;
 display: flex;
 align-items: center;
 line-height: 1;
 `),r(`checked`,`
 right: 0;
 padding-right: calc(1.25 * var(--n-rail-height) - var(--n-offset));
 `),r(`unchecked`,`
 left: 0;
 justify-content: flex-end;
 padding-left: calc(1.25 * var(--n-rail-height) - var(--n-offset));
 `),s(`&:focus`,[r(`rail`,`
 box-shadow: var(--n-box-shadow-focus);
 `)]),S(`round`,[r(`rail`,`border-radius: calc(var(--n-rail-height) / 2);`,[r(`button`,`border-radius: calc(var(--n-button-height) / 2);`)])]),u(`disabled`,[u(`icon`,[S(`rubber-band`,[S(`pressed`,[r(`rail`,[r(`button`,`max-width: var(--n-button-width-pressed);`)])]),r(`rail`,[s(`&:active`,[r(`button`,`max-width: var(--n-button-width-pressed);`)])]),S(`active`,[S(`pressed`,[r(`rail`,[r(`button`,`left: calc(100% - var(--n-offset) - var(--n-button-width-pressed));`)])]),r(`rail`,[s(`&:active`,[r(`button`,`left: calc(100% - var(--n-offset) - var(--n-button-width-pressed));`)])])])])])]),S(`active`,[r(`rail`,[r(`button`,`left: calc(100% - var(--n-button-width) - var(--n-offset))`)])]),r(`rail`,`
 overflow: hidden;
 height: var(--n-rail-height);
 min-width: var(--n-rail-width);
 border-radius: var(--n-rail-border-radius);
 cursor: pointer;
 position: relative;
 transition:
 opacity .3s var(--n-bezier),
 background .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 background-color: var(--n-rail-color);
 `,[r(`button-icon`,`
 color: var(--n-icon-color);
 transition: color .3s var(--n-bezier);
 font-size: calc(var(--n-button-height) - 4px);
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 display: flex;
 justify-content: center;
 align-items: center;
 line-height: 1;
 `,[y()]),r(`button`,`
 align-items: center; 
 top: var(--n-offset);
 left: var(--n-offset);
 height: var(--n-button-height);
 width: var(--n-button-width-pressed);
 max-width: var(--n-button-width);
 border-radius: var(--n-button-border-radius);
 background-color: var(--n-button-color);
 box-shadow: var(--n-button-box-shadow);
 box-sizing: border-box;
 cursor: inherit;
 content: "";
 position: absolute;
 transition:
 background-color .3s var(--n-bezier),
 left .3s var(--n-bezier),
 opacity .3s var(--n-bezier),
 max-width .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 `)]),S(`active`,[r(`rail`,`background-color: var(--n-rail-color-active);`)]),S(`loading`,[r(`rail`,`
 cursor: wait;
 `)]),S(`disabled`,[r(`rail`,`
 cursor: not-allowed;
 opacity: .5;
 `)])]),L=[`aria-checked`,`tabindex`,`onClick`,`onFocus`,`onBlur`,`onKeyup`,`onKeydown`],R={...C.props,size:String,value:{type:[String,Number,Boolean],default:void 0},loading:Boolean,defaultValue:{type:[String,Number,Boolean],default:!1},disabled:{type:Boolean,default:void 0},round:{type:Boolean,default:!0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],checkedValue:{type:[String,Number,Boolean],default:!0},uncheckedValue:{type:[String,Number,Boolean],default:!1},railStyle:Function,rubberBand:{type:Boolean,default:!0},spinProps:Object,onChange:[Function,Array]},z,B=T({name:`Switch`,props:R,slots:Object,setup(e){z===void 0&&(z=typeof CSS<`u`?CSS.supports!==void 0&&CSS.supports(`width`,`max(1px)`):!0);let{mergedClsPrefixRef:r,inlineThemeDisabled:i,mergedComponentPropsRef:s}=O(e),l=C(`Switch`,`-switch`,I,F,e,r),u=k(e,{mergedSize(t){return e.size===void 0?t?t.mergedSize.value:s?.value?.Switch?.size||`medium`:e.size}}),{mergedSizeRef:d,mergedDisabledRef:p}=u,h=o(e.defaultValue),g=n(e,`value`),v=N(g,h),y=_(()=>v.value===e.checkedValue),b=o(!1),x=o(!1),S=_(()=>{let{railStyle:t}=e;if(t)return t({focused:x.value,checked:y.value})});function w(t){let{"onUpdate:value":n,onChange:r,onUpdateValue:i}=e,{nTriggerFormInput:o,nTriggerFormChange:s}=u;n&&a(n,t),i&&a(i,t),r&&a(r,t),h.value=t,o(),s()}function T(){let{nTriggerFormFocus:e}=u;e()}function E(){let{nTriggerFormBlur:e}=u;e()}function D(){e.loading||p.value||(v.value===e.checkedValue?w(e.uncheckedValue):w(e.checkedValue))}function A(){x.value=!0,T()}function j(){x.value=!1,E(),b.value=!1}function M(t){e.loading||p.value||t.key===` `&&(v.value===e.checkedValue?w(e.uncheckedValue):w(e.checkedValue),b.value=!1)}function P(t){e.loading||p.value||t.key===` `&&(t.preventDefault(),b.value=!0)}let L=_(()=>{let{value:e}=d,{self:{opacityDisabled:t,railColor:n,railColorActive:r,buttonBoxShadow:i,buttonColor:a,boxShadowFocus:o,loadingColor:s,textColor:u,iconColor:p,[f(`buttonHeight`,e)]:h,[f(`buttonWidth`,e)]:g,[f(`buttonWidthPressed`,e)]:_,[f(`railHeight`,e)]:v,[f(`railWidth`,e)]:y,[f(`railBorderRadius`,e)]:b,[f(`buttonBorderRadius`,e)]:x},common:{cubicBezierEaseInOut:S}}=l.value,C,w,T;return z?(C=`calc((${v} - ${h}) / 2)`,w=`max(${v}, ${h})`,T=`max(${y}, calc(${y} + ${h} - ${v}))`):(C=m((c(v)-c(h))/2),w=m(Math.max(c(v),c(h))),T=c(v)>c(h)?y:m(c(y)+c(h)-c(v))),{"--n-bezier":S,"--n-button-border-radius":x,"--n-button-box-shadow":i,"--n-button-color":a,"--n-button-width":g,"--n-button-width-pressed":_,"--n-button-height":h,"--n-height":w,"--n-offset":C,"--n-opacity-disabled":t,"--n-rail-border-radius":b,"--n-rail-color":n,"--n-rail-color-active":r,"--n-rail-height":v,"--n-rail-width":y,"--n-width":T,"--n-box-shadow-focus":o,"--n-loading-color":s,"--n-text-color":u,"--n-icon-color":p}}),R=i?t(`switch`,_(()=>d.value[0]),L,e):void 0;return{handleClick:D,handleBlur:j,handleFocus:A,handleKeyup:M,handleKeydown:P,mergedRailStyle:S,pressed:b,mergedClsPrefix:r,mergedValue:v,checked:y,mergedDisabled:p,cssVars:i?void 0:L,themeClass:R?.themeClass,onRender:R?.onRender}},render(){let{mergedClsPrefix:t,mergedDisabled:n,checked:r,mergedRailStyle:a,onRender:o,$slots:s}=this;o?.();let{checked:c,unchecked:l,icon:u,"checked-icon":f,"unchecked-icon":p}=s,m=!(A(u)&&A(f)&&A(p));return b(),e(`div`,{role:`switch`,"aria-checked":r,class:i([`${t}-switch`,this.themeClass,m&&`${t}-switch--icon`,r&&`${t}-switch--active`,n&&`${t}-switch--disabled`,this.round&&`${t}-switch--round`,this.loading&&`${t}-switch--loading`,this.pressed&&`${t}-switch--pressed`,this.rubberBand&&`${t}-switch--rubber-band`]),tabindex:this.mergedDisabled?void 0:0,style:d(this.cssVars),onClick:this.handleClick,onFocus:this.handleFocus,onBlur:this.handleBlur,onKeyup:this.handleKeyup,onKeydown:this.handleKeydown},[g(`div`,{class:i(`${t}-switch__rail`),"aria-hidden":`true`,style:d(a)},[h(()=>j(c,n=>j(l,r=>n||r?(b(),e(`div`,{key:4,"aria-hidden":!0,class:i(`${t}-switch__children-placeholder`)},[g(`div`,{class:i(`${t}-switch__rail-placeholder`)},[g(`div`,{class:i(`${t}-switch__button-placeholder`)},null,2),h(()=>n)],2),g(`div`,{class:i(`${t}-switch__rail-placeholder`)},[g(`div`,{class:i(`${t}-switch__button-placeholder`)},null,2),h(()=>r)],2)],2)):null))),g(`div`,{class:i(`${t}-switch__button`)},[h(()=>j(u,n=>j(f,r=>j(p,a=>(b(),v(w,null,{default:()=>this.loading?(b(),v(E,D({key:`loading`,clsPrefix:t,strokeWidth:20},this.spinProps),null,16,[`clsPrefix`])):this.checked&&(r||n)?(b(),e(`div`,{class:i(`${t}-switch__button-icon`),key:r?`checked-icon`:`icon`},[h(()=>r||n)],2)):!this.checked&&(a||n)?(b(),e(`div`,{class:i(`${t}-switch__button-icon`),key:a?`unchecked-icon`:`icon`},[h(()=>a||n)],2)):null},1024)))))),h(()=>j(c,n=>n&&(b(),e(`div`,{key:`checked`,class:i(`${t}-switch__checked`)},[h(()=>n)],2)))),h(()=>j(l,n=>n&&(b(),e(`div`,{key:`unchecked`,class:i(`${t}-switch__unchecked`)},[h(()=>n)],2))))],2)],6)],46,L)}});export{B as t};