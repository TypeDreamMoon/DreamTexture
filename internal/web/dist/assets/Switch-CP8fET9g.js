import{$t as e,A as t,At as n,B as r,C as i,Dt as a,I as o,M as s,Mt as c,Nt as l,On as u,Ot as d,Pn as f,R as p,U as m,Xt as h,Yt as g,Zt as _,c as v,hn as y,j as b,jn as x,jt as S,k as C,l as w,nn as T,o as E,on as D,q as O,u as k,v as A,x as j}from"./client-HeelZ21K.js";import{g as M,rt as N}from"./index-DWif4xih.js";function P(e){let{primaryColor:t,opacityDisabled:n,borderRadius:r,textColor3:i}=e;return{...M,iconColor:i,textColor:`white`,loadingColor:t,opacityDisabled:n,railColor:`rgba(0, 0, 0, .14)`,railColorActive:t,buttonBoxShadow:`0 1px 4px 0 rgba(0, 0, 0, 0.3), inset 0 0 1px 0 rgba(0, 0, 0, 0.05)`,buttonColor:`#FFF`,railBorderRadiusSmall:r,railBorderRadiusMedium:r,railBorderRadiusLarge:r,buttonBorderRadiusSmall:r,buttonBorderRadiusMedium:r,buttonBorderRadiusLarge:r,boxShadowFocus:`0 0 0 2px ${s(t,{alpha:.2})}`}}var F={name:`Switch`,common:b,self:P},I=d(`switch`,`
 height: var(--n-height);
 min-width: var(--n-width);
 vertical-align: middle;
 user-select: none;
 -webkit-user-select: none;
 display: inline-flex;
 outline: none;
 justify-content: center;
 align-items: center;
`,[n(`children-placeholder`,`
 height: var(--n-rail-height);
 display: flex;
 flex-direction: column;
 overflow: hidden;
 pointer-events: none;
 visibility: hidden;
 `),n(`rail-placeholder`,`
 display: flex;
 flex-wrap: none;
 `),n(`button-placeholder`,`
 width: calc(1.75 * var(--n-rail-height));
 height: var(--n-rail-height);
 `),d(`base-loading`,`
 position: absolute;
 top: 50%;
 left: 50%;
 transform: translateX(-50%) translateY(-50%);
 font-size: calc(var(--n-button-width) - 4px);
 color: var(--n-loading-color);
 transition: color .3s var(--n-bezier);
 `,[v({left:`50%`,top:`50%`,originalTransform:`translateX(-50%) translateY(-50%)`})]),n(`checked, unchecked`,`
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
 `),n(`checked`,`
 right: 0;
 padding-right: calc(1.25 * var(--n-rail-height) - var(--n-offset));
 `),n(`unchecked`,`
 left: 0;
 justify-content: flex-end;
 padding-left: calc(1.25 * var(--n-rail-height) - var(--n-offset));
 `),a(`&:focus`,[n(`rail`,`
 box-shadow: var(--n-box-shadow-focus);
 `)]),S(`round`,[n(`rail`,`border-radius: calc(var(--n-rail-height) / 2);`,[n(`button`,`border-radius: calc(var(--n-button-height) / 2);`)])]),c(`disabled`,[c(`icon`,[S(`rubber-band`,[S(`pressed`,[n(`rail`,[n(`button`,`max-width: var(--n-button-width-pressed);`)])]),n(`rail`,[a(`&:active`,[n(`button`,`max-width: var(--n-button-width-pressed);`)])]),S(`active`,[S(`pressed`,[n(`rail`,[n(`button`,`left: calc(100% - var(--n-offset) - var(--n-button-width-pressed));`)])]),n(`rail`,[a(`&:active`,[n(`button`,`left: calc(100% - var(--n-offset) - var(--n-button-width-pressed));`)])])])])])]),S(`active`,[n(`rail`,[n(`button`,`left: calc(100% - var(--n-button-width) - var(--n-offset))`)])]),n(`rail`,`
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
 `,[n(`button-icon`,`
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
 `,[v()]),n(`button`,`
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
 `)]),S(`active`,[n(`rail`,`background-color: var(--n-rail-color-active);`)]),S(`loading`,[n(`rail`,`
 cursor: wait;
 `)]),S(`disabled`,[n(`rail`,`
 cursor: not-allowed;
 opacity: .5;
 `)])]),L=[`aria-checked`,`tabindex`,`onClick`,`onFocus`,`onBlur`,`onKeyup`,`onKeydown`],R={...C.props,size:String,value:{type:[String,Number,Boolean],default:void 0},loading:Boolean,defaultValue:{type:[String,Number,Boolean],default:!1},disabled:{type:Boolean,default:void 0},round:{type:Boolean,default:!0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],checkedValue:{type:[String,Number,Boolean],default:!0},uncheckedValue:{type:[String,Number,Boolean],default:!1},railStyle:Function,rubberBand:{type:Boolean,default:!0},spinProps:Object,onChange:[Function,Array]},z,B=T({name:`Switch`,props:R,slots:Object,setup(e){z===void 0&&(z=typeof CSS<`u`?CSS.supports!==void 0&&CSS.supports(`width`,`max(1px)`):!0);let{mergedClsPrefixRef:n,inlineThemeDisabled:r,mergedComponentPropsRef:a}=O(e),s=C(`Switch`,`-switch`,I,F,e,n),c=k(e,{mergedSize(t){return e.size===void 0?t?t.mergedSize.value:a?.value?.Switch?.size||`medium`:e.size}}),{mergedSizeRef:d,mergedDisabledRef:f}=c,m=u(e.defaultValue),h=x(e,`value`),_=N(h,m),v=g(()=>_.value===e.checkedValue),y=u(!1),b=u(!1),S=g(()=>{let{railStyle:t}=e;if(t)return t({focused:b.value,checked:v.value})});function w(t){let{"onUpdate:value":n,onChange:r,onUpdateValue:a}=e,{nTriggerFormInput:o,nTriggerFormChange:s}=c;n&&i(n,t),a&&i(a,t),r&&i(r,t),m.value=t,o(),s()}function T(){let{nTriggerFormFocus:e}=c;e()}function E(){let{nTriggerFormBlur:e}=c;e()}function D(){e.loading||f.value||(_.value===e.checkedValue?w(e.uncheckedValue):w(e.checkedValue))}function A(){b.value=!0,T()}function j(){b.value=!1,E(),y.value=!1}function M(t){e.loading||f.value||t.key===` `&&(_.value===e.checkedValue?w(e.uncheckedValue):w(e.checkedValue),y.value=!1)}function P(t){e.loading||f.value||t.key===` `&&(t.preventDefault(),y.value=!0)}let L=g(()=>{let{value:e}=d,{self:{opacityDisabled:t,railColor:n,railColorActive:r,buttonBoxShadow:i,buttonColor:a,boxShadowFocus:c,loadingColor:u,textColor:f,iconColor:m,[l(`buttonHeight`,e)]:h,[l(`buttonWidth`,e)]:g,[l(`buttonWidthPressed`,e)]:_,[l(`railHeight`,e)]:v,[l(`railWidth`,e)]:y,[l(`railBorderRadius`,e)]:b,[l(`buttonBorderRadius`,e)]:x},common:{cubicBezierEaseInOut:S}}=s.value,C,w,T;return z?(C=`calc((${v} - ${h}) / 2)`,w=`max(${v}, ${h})`,T=`max(${y}, calc(${y} + ${h} - ${v}))`):(C=p((o(v)-o(h))/2),w=p(Math.max(o(v),o(h))),T=o(v)>o(h)?y:p(o(y)+o(h)-o(v))),{"--n-bezier":S,"--n-button-border-radius":x,"--n-button-box-shadow":i,"--n-button-color":a,"--n-button-width":g,"--n-button-width-pressed":_,"--n-button-height":h,"--n-height":w,"--n-offset":C,"--n-opacity-disabled":t,"--n-rail-border-radius":b,"--n-rail-color":n,"--n-rail-color-active":r,"--n-rail-height":v,"--n-rail-width":y,"--n-width":T,"--n-box-shadow-focus":c,"--n-loading-color":u,"--n-text-color":f,"--n-icon-color":m}}),R=r?t(`switch`,g(()=>d.value[0]),L,e):void 0;return{handleClick:D,handleBlur:j,handleFocus:A,handleKeyup:M,handleKeydown:P,mergedRailStyle:S,pressed:y,mergedClsPrefix:n,mergedValue:_,checked:v,mergedDisabled:f,cssVars:r?void 0:L,themeClass:R?.themeClass,onRender:R?.onRender}},render(){let{mergedClsPrefix:t,mergedDisabled:n,checked:i,mergedRailStyle:a,onRender:o,$slots:s}=this;o?.();let{checked:c,unchecked:l,icon:u,"checked-icon":d,"unchecked-icon":p}=s,g=!(A(u)&&A(d)&&A(p));return y(),e(`div`,{role:`switch`,"aria-checked":i,class:r([`${t}-switch`,this.themeClass,g&&`${t}-switch--icon`,i&&`${t}-switch--active`,n&&`${t}-switch--disabled`,this.round&&`${t}-switch--round`,this.loading&&`${t}-switch--loading`,this.pressed&&`${t}-switch--pressed`,this.rubberBand&&`${t}-switch--rubber-band`]),tabindex:this.mergedDisabled?void 0:0,style:f(this.cssVars),onClick:this.handleClick,onFocus:this.handleFocus,onBlur:this.handleBlur,onKeyup:this.handleKeyup,onKeydown:this.handleKeydown},[h(`div`,{class:r(`${t}-switch__rail`),"aria-hidden":`true`,style:f(a)},[m(()=>j(c,n=>j(l,i=>n||i?(y(),e(`div`,{key:4,"aria-hidden":!0,class:r(`${t}-switch__children-placeholder`)},[h(`div`,{class:r(`${t}-switch__rail-placeholder`)},[h(`div`,{class:r(`${t}-switch__button-placeholder`)},null,2),m(()=>n)],2),h(`div`,{class:r(`${t}-switch__rail-placeholder`)},[h(`div`,{class:r(`${t}-switch__button-placeholder`)},null,2),m(()=>i)],2)],2)):null))),h(`div`,{class:r(`${t}-switch__button`)},[m(()=>j(u,n=>j(d,i=>j(p,a=>(y(),_(w,null,{default:()=>this.loading?(y(),_(E,D({key:`loading`,clsPrefix:t,strokeWidth:20},this.spinProps),null,16,[`clsPrefix`])):this.checked&&(i||n)?(y(),e(`div`,{class:r(`${t}-switch__button-icon`),key:i?`checked-icon`:`icon`},[m(()=>i||n)],2)):!this.checked&&(a||n)?(y(),e(`div`,{class:r(`${t}-switch__button-icon`),key:a?`unchecked-icon`:`icon`},[m(()=>a||n)],2)):null},1024)))))),m(()=>j(c,n=>n&&(y(),e(`div`,{key:`checked`,class:r(`${t}-switch__checked`)},[m(()=>n)],2)))),m(()=>j(l,n=>n&&(y(),e(`div`,{key:`unchecked`,class:r(`${t}-switch__unchecked`)},[m(()=>n)],2))))],2)],6)],46,L)}});export{B as t};