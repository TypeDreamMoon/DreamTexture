import{$t as e,A as t,An as n,At as r,B as i,C as a,Cn as o,D as s,Dn as c,Dt as l,E as u,Gt as d,H as f,Ht as p,Mn as m,Mt as h,Nn as g,Nt as _,Ot as v,Pn as y,Qt as b,Rt as x,S,Sn as C,U as w,Ut as T,V as E,X as D,Xt as O,Y as k,Yt as A,Zt as j,_n as M,an as N,b as P,bn as ee,en as F,g as I,gn as L,hn as R,in as z,jn as B,jt as V,k as H,n as te,nn as U,pn as W,q as G,r as K,t as q,tn as J,u as ne,x as re,z as Y}from"./client-Do8VSizq.js";import{i as ie,n as ae,t as X}from"./Select-Dkoeg12f.js";import{t as Z}from"./use-message-B4TpC56a.js";import{n as oe,t as se}from"./Slider-okY8ha9U.js";import{t as ce}from"./Progress-DuAk897b.js";import{t as le}from"./Spin-CQJ6XXoF.js";import{t as ue}from"./Switch-B5j0QnJO.js";import{$ as de,F as fe,O as Q,P as pe,S as me,_t as he,a as ge,c as _e,d as ve,f as $,gt as ye,i as be,l as xe,lt as Se,nt as Ce,o as we,r as Te,s as Ee,t as De,tt as Oe,w as ke,x as Ae}from"./index-DEvJnqPU.js";import{t as je}from"./PageHeader-Ds_XrywY.js";var Me=U({name:`ChevronLeft`,render(){return(()=>{let e=Y(`dfe229c2639b2082`);return e[0]||=O(`svg`,{viewBox:`0 0 16 16`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},[O(`path`,{d:`M10.3536 3.14645C10.5488 3.34171 10.5488 3.65829 10.3536 3.85355L6.20711 8L10.3536 12.1464C10.5488 12.3417 10.5488 12.6583 10.3536 12.8536C10.1583 13.0488 9.84171 13.0488 9.64645 12.8536L5.14645 8.35355C4.95118 8.15829 4.95118 7.84171 5.14645 7.64645L9.64645 3.14645C9.84171 2.95118 10.1583 2.95118 10.3536 3.14645Z`,fill:`currentColor`})],-1)})()}}),Ne=U({name:`ChevronRight`,render(){return(()=>{let e=Y(`6ab04425f4fcb756`);return e[0]||=O(`svg`,{viewBox:`0 0 16 16`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},[O(`path`,{d:`M5.64645 3.14645C5.45118 3.34171 5.45118 3.65829 5.64645 3.85355L9.79289 8L5.64645 12.1464C5.45118 12.3417 5.45118 12.6583 5.64645 12.8536C5.84171 13.0488 6.15829 13.0488 6.35355 12.8536L10.8536 8.35355C11.0488 8.15829 11.0488 7.84171 10.8536 7.64645L6.35355 3.14645C6.15829 2.95118 5.84171 2.95118 5.64645 3.14645Z`,fill:`currentColor`})],-1)})()}}),Pe=v(`collapse`,`width: 100%;`,[v(`collapse-item`,`
 font-size: var(--n-font-size);
 color: var(--n-text-color);
 transition:
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 margin: var(--n-item-margin);
 `,[V(`disabled`,[r(`header`,`cursor: not-allowed;`,[r(`header-main`,`
 color: var(--n-title-text-color-disabled);
 `),v(`collapse-item-arrow`,`
 color: var(--n-arrow-color-disabled);
 `)])]),v(`collapse-item`,`margin-left: 32px;`),l(`&:first-child`,`margin-top: 0;`),l(`&:first-child >`,[r(`header`,`padding-top: 0;`)]),V(`left-arrow-placement`,[r(`header`,[v(`collapse-item-arrow`,`margin-right: 4px;`)])]),V(`right-arrow-placement`,[r(`header`,[v(`collapse-item-arrow`,`margin-left: 4px;`)])]),r(`content-wrapper`,[r(`content-inner`,`padding-top: 16px;`),pe({duration:`0.15s`})]),V(`active`,[r(`header`,[V(`active`,[v(`collapse-item-arrow`,`transform: rotate(90deg);`)])])]),l(`&:not(:first-child)`,`border-top: 1px solid var(--n-divider-color);`),h(`disabled`,[V(`trigger-area-main`,[r(`header`,[r(`header-main`,`cursor: pointer;`),v(`collapse-item-arrow`,`cursor: default;`)])]),V(`trigger-area-arrow`,[r(`header`,[v(`collapse-item-arrow`,`cursor: pointer;`)])]),V(`trigger-area-extra`,[r(`header`,[r(`header-extra`,`cursor: pointer;`)])])]),r(`header`,`
 font-size: var(--n-title-font-size);
 display: flex;
 flex-wrap: nowrap;
 align-items: center;
 transition: color .3s var(--n-bezier);
 position: relative;
 padding: var(--n-title-padding);
 color: var(--n-title-text-color);
 `,[r(`header-main`,`
 display: flex;
 flex-wrap: nowrap;
 align-items: center;
 font-weight: var(--n-title-font-weight);
 transition: color .3s var(--n-bezier);
 flex: 1;
 color: var(--n-title-text-color);
 `),r(`header-extra`,`
 display: flex;
 align-items: center;
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),v(`collapse-item-arrow`,`
 display: flex;
 transition:
 transform .15s var(--n-bezier),
 color .3s var(--n-bezier);
 font-size: 18px;
 color: var(--n-arrow-color);
 `)])])]),Fe={...H.props,defaultExpandedNames:{type:[Array,String],default:null},expandedNames:[Array,String],arrowPlacement:{type:String,default:`left`},accordion:Boolean,displayDirective:{type:String,default:`if`},triggerAreas:{type:Array,default:()=>[`main`,`extra`,`arrow`]},onItemHeaderClick:[Function,Array],"onUpdate:expandedNames":[Function,Array],onUpdateExpandedNames:[Function,Array],onExpandedNamesChange:{type:[Function,Array],validator:()=>!0,default:void 0}},Ie=k(`n-collapse`),Le=U({name:`Collapse`,props:Fe,slots:Object,setup(e,{slots:n}){let{mergedClsPrefixRef:r,inlineThemeDisabled:i,mergedRtlRef:o}=G(e),s=c(e.defaultExpandedNames),l=A(()=>e.expandedNames),u=Oe(l,s),d=H(`Collapse`,`-collapse`,Pe,ke,e,r);function f(t){let{"onUpdate:expandedNames":n,onUpdateExpandedNames:r,onExpandedNamesChange:i}=e;r&&a(r,t),n&&a(n,t),i&&a(i,t),s.value=t}function p(t){let{onItemHeaderClick:n}=e;n&&a(n,t)}function m(t,n,r){let{accordion:i}=e,{value:a}=u;if(i)t?(f([n]),p({name:n,expanded:!0,event:r})):(f([]),p({name:n,expanded:!1,event:r}));else if(!Array.isArray(a))f([n]),p({name:n,expanded:!0,event:r});else{let e=a.slice(),t=e.findIndex(e=>n===e);~t?(e.splice(t,1),f(e),p({name:n,expanded:!1,event:r})):(e.push(n),f(e),p({name:n,expanded:!0,event:r}))}}L(Ie,{props:e,mergedClsPrefixRef:r,expandedNamesRef:u,slots:n,toggleItem:m});let h=I(`Collapse`,o,r),g=A(()=>{let{common:{cubicBezierEaseInOut:e},self:{titleFontWeight:t,dividerColor:n,titlePadding:r,titleTextColor:i,titleTextColorDisabled:a,textColor:o,arrowColor:s,fontSize:c,titleFontSize:l,arrowColorDisabled:u,itemMargin:f}}=d.value;return{"--n-font-size":c,"--n-bezier":e,"--n-text-color":o,"--n-divider-color":n,"--n-title-padding":r,"--n-title-font-size":l,"--n-title-text-color":i,"--n-title-text-color-disabled":a,"--n-title-font-weight":t,"--n-arrow-color":s,"--n-arrow-color-disabled":u,"--n-item-margin":f}}),_=i?t(`collapse`,void 0,g,e):void 0;return{rtlEnabled:h,mergedTheme:d,mergedClsPrefix:r,cssVars:i?void 0:g,themeClass:_?.themeClass,onRender:_?.onRender}},render(){return this.onRender?.(),R(),e(`div`,{class:i([`${this.mergedClsPrefix}-collapse`,this.rtlEnabled&&`${this.mergedClsPrefix}-collapse--rtl`,this.themeClass]),style:g(this.cssVars)},[w(()=>this.$slots.default?.())],6)}}),Re=U({name:`CollapseItemContent`,props:{displayDirective:{type:String,required:!0},show:Boolean,clsPrefix:{type:String,required:!0}},setup(e){return{onceTrue:Ce(n(e,`show`))}},render(){return R(),j(fe,null,{_:1,default:E(()=>{let{show:t,displayDirective:n,onceTrue:r,clsPrefix:a}=this,s=n===`show`&&r,c=(R(),e(`div`,{class:i(`${a}-collapse-item__content-wrapper`)},[O(`div`,{class:i(`${a}-collapse-item__content-inner`)},[w(()=>this.$slots.default?.())],2)],2));return s?o(c,[[p,t]]):t?c:null})})}}),ze=[`onClick`],Be=[`onClick`],Ve=U({name:`CollapseItem`,props:{title:String,name:[String,Number],disabled:Boolean,displayDirective:String},setup(e){let{mergedRtlRef:t}=G(e),r=Se(),i=u(()=>e.name??r),a=N(Ie);a||D(`collapse-item`,"`n-collapse-item` must be placed inside `n-collapse`.");let{expandedNamesRef:o,props:s,mergedClsPrefixRef:c,slots:l}=a,d=A(()=>{let{value:e}=o;if(Array.isArray(e)){let{value:t}=i;return!~e.findIndex(e=>e===t)}if(e){let{value:t}=i;return t!==e}return!0});return{rtlEnabled:I(`Collapse`,t,c),collapseSlots:l,randomName:r,mergedClsPrefix:c,collapsed:d,triggerAreas:n(s,`triggerAreas`),mergedDisplayDirective:A(()=>{let{displayDirective:t}=e;return t||s.displayDirective}),arrowPlacement:A(()=>s.arrowPlacement),handleClick(t){let n=`main`;ie(t,`arrow`)&&(n=`arrow`),ie(t,`extra`)&&(n=`extra`),s.triggerAreas.includes(n)&&a&&!e.disabled&&a.toggleItem(d.value,i.value,t)}}},render(){let{collapseSlots:t,$slots:n,arrowPlacement:r,collapsed:a,mergedDisplayDirective:o,mergedClsPrefix:c,disabled:l,triggerAreas:u}=this,d=P(n.header,{collapsed:a},()=>[this.title]),p=n[`header-extra`]||t[`header-extra`],m=n.arrow||t.arrow;return R(),e(`div`,{class:i([`${c}-collapse-item`,`${c}-collapse-item--${r}-arrow-placement`,l&&`${c}-collapse-item--disabled`,!a&&`${c}-collapse-item--active`,u.map(e=>`${c}-collapse-item--trigger-area-${e}`)])},[O(`div`,{class:i([`${c}-collapse-item__header`,!a&&`${c}-collapse-item__header--active`])},[O(`div`,{class:i(`${c}-collapse-item__header-main`),onClick:this.handleClick},[w(()=>r===`right`&&d),(R(),e(`div`,{class:i(`${c}-collapse-item-arrow`),key:+!this.rtlEnabled,"data-arrow":!0},[w(()=>P(m,{collapsed:a},()=>[(R(),j(s,{clsPrefix:c},{default:()=>this.rtlEnabled?(R(),j(Me,{key:1})):(R(),j(Ne,{key:2}))},1032,[`clsPrefix`]))]))],2)),w(()=>r===`left`&&d)],10,Be),w(()=>S(p,{collapsed:a},t=>(R(),e(`div`,{class:i(`${c}-collapse-item__header-extra`),onClick:this.handleClick,"data-extra":!0},[w(()=>t)],10,ze))))],2),(R(),j(Re,{clsPrefix:c,displayDirective:o,show:!a},f(n),1032,[`clsPrefix`,`displayDirective`,`show`]))],2)}}),He=v(`radio`,`
 line-height: var(--n-label-line-height);
 outline: none;
 position: relative;
 user-select: none;
 -webkit-user-select: none;
 display: inline-flex;
 align-items: flex-start;
 flex-wrap: nowrap;
 font-size: var(--n-font-size);
 word-break: break-word;
`,[V(`checked`,[r(`dot`,`
 background-color: var(--n-color-active);
 `)]),r(`dot-wrapper`,`
 position: relative;
 flex-shrink: 0;
 flex-grow: 0;
 width: var(--n-radio-size);
 `),v(`radio-input`,`
 position: absolute;
 border: 0;
 width: 0;
 height: 0;
 opacity: 0;
 margin: 0;
 `),r(`dot`,`
 position: absolute;
 top: 50%;
 left: 0;
 transform: translateY(-50%);
 height: var(--n-radio-size);
 width: var(--n-radio-size);
 background: var(--n-color);
 box-shadow: var(--n-box-shadow);
 border-radius: 50%;
 transition:
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 `,[l(`&::before`,`
 content: "";
 opacity: 0;
 position: absolute;
 left: 4px;
 top: 4px;
 height: calc(100% - 8px);
 width: calc(100% - 8px);
 border-radius: 50%;
 transform: scale(.8);
 background: var(--n-dot-color-active);
 transition: 
 opacity .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 transform .3s var(--n-bezier);
 `),V(`checked`,{boxShadow:`var(--n-box-shadow-active)`},[l(`&::before`,`
 opacity: 1;
 transform: scale(1);
 `)])]),r(`label`,`
 color: var(--n-text-color);
 padding: var(--n-label-padding);
 font-weight: var(--n-label-font-weight);
 display: inline-block;
 transition: color .3s var(--n-bezier);
 `),h(`disabled`,`
 cursor: pointer;
 `,[l(`&:hover`,[r(`dot`,{boxShadow:`var(--n-box-shadow-hover)`})]),V(`focus`,[l(`&:not(:active)`,[r(`dot`,{boxShadow:`var(--n-box-shadow-focus)`})])])]),V(`disabled`,`
 cursor: not-allowed;
 `,[r(`dot`,{boxShadow:`var(--n-box-shadow-disabled)`,backgroundColor:`var(--n-color-disabled)`},[l(`&::before`,{backgroundColor:`var(--n-dot-color-disabled)`}),V(`checked`,`
 opacity: 1;
 `)]),r(`label`,{color:`var(--n-text-color-disabled)`}),v(`radio-input`,`
 cursor: not-allowed;
 `)])]),Ue={name:String,value:{type:[String,Number,Boolean],default:`on`},checked:{type:Boolean,default:void 0},defaultChecked:Boolean,disabled:{type:Boolean,default:void 0},label:String,size:String,onUpdateChecked:[Function,Array],"onUpdate:checked":[Function,Array],checkedValue:{type:Boolean,default:void 0}},We=k(`n-radio-group`);function Ge(e){let t=N(We,null),{mergedClsPrefixRef:r,mergedComponentPropsRef:i}=G(e),o=ne(e,{mergedSize(n){let{size:r}=e;if(r!==void 0)return r;if(t){let{mergedSizeRef:{value:e}}=t;if(e!==void 0)return e}return n?n.mergedSize.value:i?.value?.Radio?.size||`medium`},mergedDisabled(n){return!!(e.disabled||t?.disabledRef.value||n?.disabled.value)}}),{mergedSizeRef:s,mergedDisabledRef:l}=o,d=c(null),f=c(null),p=c(e.defaultChecked),m=n(e,`checked`),h=Oe(m,p),g=u(()=>t?t.valueRef.value===e.value:h.value),_=u(()=>{let{name:n}=e;if(n!==void 0)return n;if(t)return t.nameRef.value}),v=c(!1);function y(){if(t){let{doUpdateValue:n}=t,{value:r}=e;a(n,r)}else{let{onUpdateChecked:t,"onUpdate:checked":n}=e,{nTriggerFormInput:r,nTriggerFormChange:i}=o;t&&a(t,!0),n&&a(n,!0),r(),i(),p.value=!0}}function b(){l.value||g.value||y()}function x(){b(),d.value&&(d.value.checked=g.value)}function S(){v.value=!1}function C(){v.value=!0}return{mergedClsPrefix:t?t.mergedClsPrefixRef:r,inputRef:d,labelRef:f,mergedName:_,mergedDisabled:l,renderSafeChecked:g,focus:v,mergedSize:s,handleRadioInputChange:x,handleRadioInputBlur:S,handleRadioInputFocus:C}}var Ke=[`value`,`name`,`checked`,`disabled`,`onChange`,`onFocus`,`onBlur`],qe={...H.props,...Ue},Je=U({name:`Radio`,props:qe,setup(e){let n=Ge(e),r=H(`Radio`,`-radio`,He,me,e,n.mergedClsPrefix),i=A(()=>{let{mergedSize:{value:e}}=n,{common:{cubicBezierEaseInOut:t},self:{boxShadow:i,boxShadowActive:a,boxShadowDisabled:o,boxShadowFocus:s,boxShadowHover:c,color:l,colorDisabled:u,colorActive:d,textColor:f,textColorDisabled:p,dotColorActive:m,dotColorDisabled:h,labelPadding:g,labelLineHeight:v,labelFontWeight:y,[_(`fontSize`,e)]:b,[_(`radioSize`,e)]:x}}=r.value;return{"--n-bezier":t,"--n-label-line-height":v,"--n-label-font-weight":y,"--n-box-shadow":i,"--n-box-shadow-active":a,"--n-box-shadow-disabled":o,"--n-box-shadow-focus":s,"--n-box-shadow-hover":c,"--n-color":l,"--n-color-active":d,"--n-color-disabled":u,"--n-dot-color-active":m,"--n-dot-color-disabled":h,"--n-font-size":b,"--n-radio-size":x,"--n-text-color":f,"--n-text-color-disabled":p,"--n-label-padding":g}}),{inlineThemeDisabled:a,mergedClsPrefixRef:o,mergedRtlRef:s}=G(e),c=I(`Radio`,s,o),l=a?t(`radio`,A(()=>n.mergedSize.value[0]),i,e):void 0;return Object.assign(n,{rtlEnabled:c,cssVars:a?void 0:i,themeClass:l?.themeClass,onRender:l?.onRender})},render(){let{$slots:t,mergedClsPrefix:n,onRender:r,label:a}=this;return r?.(),(()=>{let r=Y(`f8c6901d8cd45c02`);return R(),e(`label`,{class:i([`${n}-radio`,this.themeClass,this.rtlEnabled&&`${n}-radio--rtl`,this.mergedDisabled&&`${n}-radio--disabled`,this.renderSafeChecked&&`${n}-radio--checked`,this.focus&&`${n}-radio--focus`]),style:g(this.cssVars)},[O(`div`,{class:i(`${n}-radio__dot-wrapper`)},[r[0]||=w(`\xA0`,-1),O(`div`,{class:i([`${n}-radio__dot`,this.renderSafeChecked&&`${n}-radio__dot--checked`])},null,2),O(`input`,{ref:`inputRef`,type:`radio`,class:i(`${n}-radio-input`),value:this.value,name:this.mergedName,checked:this.renderSafeChecked,disabled:this.mergedDisabled,onChange:this.handleRadioInputChange,onFocus:this.handleRadioInputFocus,onBlur:this.handleRadioInputBlur},null,42,Ke)],2),w(()=>re(t.default,t=>!t&&!a?null:(R(),e(`div`,{ref:`labelRef`,class:i(`${n}-radio__label`)},[w(()=>t||a)],2))))],6)})()}}),Ye=[`value`,`name`,`checked`,`disabled`,`onChange`,`onFocus`,`onBlur`],Xe=U({name:`RadioButton`,props:Ue,setup:Ge,render(){let{mergedClsPrefix:t}=this;return R(),e(`label`,{class:i([`${t}-radio-button`,this.mergedDisabled&&`${t}-radio-button--disabled`,this.renderSafeChecked&&`${t}-radio-button--checked`,this.focus&&[`${t}-radio-button--focus`]])},[O(`input`,{ref:`inputRef`,type:`radio`,class:i(`${t}-radio-input`),value:this.value,name:this.mergedName,checked:this.renderSafeChecked,disabled:this.mergedDisabled,onChange:this.handleRadioInputChange,onFocus:this.handleRadioInputFocus,onBlur:this.handleRadioInputBlur},null,42,Ye),O(`div`,{class:i(`${t}-radio-button__state-border`)},null,2),w(()=>re(this.$slots.default,n=>!n&&!this.label?null:(R(),e(`div`,{ref:`labelRef`,class:i(`${t}-radio__label`)},[w(()=>n||this.label)],2))))],2)}});function Ze(e,t=`default`,n=[]){let r=e.$slots[t];return r===void 0?n:r()}var Qe=v(`radio-group`,`
 display: inline-block;
 font-size: var(--n-font-size);
`,[r(`splitor`,`
 display: inline-block;
 vertical-align: bottom;
 width: 1px;
 transition:
 background-color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 background: var(--n-button-border-color);
 `,[V(`checked`,{backgroundColor:`var(--n-button-border-color-active)`}),V(`disabled`,{opacity:`var(--n-opacity-disabled)`})]),V(`button-group`,`
 white-space: nowrap;
 height: var(--n-height);
 line-height: var(--n-height);
 `,[v(`radio-button`,{height:`var(--n-height)`,lineHeight:`var(--n-height)`}),r(`splitor`,{height:`var(--n-height)`})]),v(`radio-button`,`
 vertical-align: bottom;
 outline: none;
 position: relative;
 user-select: none;
 -webkit-user-select: none;
 display: inline-block;
 box-sizing: border-box;
 padding-left: 14px;
 padding-right: 14px;
 white-space: nowrap;
 transition:
 background-color .3s var(--n-bezier),
 opacity .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 background: var(--n-button-color);
 color: var(--n-button-text-color);
 border-top: 1px solid var(--n-button-border-color);
 border-bottom: 1px solid var(--n-button-border-color);
 `,[v(`radio-input`,`
 pointer-events: none;
 position: absolute;
 border: 0;
 border-radius: inherit;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 opacity: 0;
 z-index: 1;
 `),r(`state-border`,`
 z-index: 1;
 pointer-events: none;
 position: absolute;
 box-shadow: var(--n-button-box-shadow);
 transition: box-shadow .3s var(--n-bezier);
 left: -1px;
 bottom: -1px;
 right: -1px;
 top: -1px;
 `),l(`&:first-child`,`
 border-top-left-radius: var(--n-button-border-radius);
 border-bottom-left-radius: var(--n-button-border-radius);
 border-left: 1px solid var(--n-button-border-color);
 `,[r(`state-border`,`
 border-top-left-radius: var(--n-button-border-radius);
 border-bottom-left-radius: var(--n-button-border-radius);
 `)]),l(`&:last-child`,`
 border-top-right-radius: var(--n-button-border-radius);
 border-bottom-right-radius: var(--n-button-border-radius);
 border-right: 1px solid var(--n-button-border-color);
 `,[r(`state-border`,`
 border-top-right-radius: var(--n-button-border-radius);
 border-bottom-right-radius: var(--n-button-border-radius);
 `)]),h(`disabled`,`
 cursor: pointer;
 `,[l(`&:hover`,[r(`state-border`,`
 transition: box-shadow .3s var(--n-bezier);
 box-shadow: var(--n-button-box-shadow-hover);
 `),h(`checked`,{color:`var(--n-button-text-color-hover)`})]),V(`focus`,[l(`&:not(:active)`,[r(`state-border`,{boxShadow:`var(--n-button-box-shadow-focus)`})])])]),V(`checked`,`
 background: var(--n-button-color-active);
 color: var(--n-button-text-color-active);
 border-color: var(--n-button-border-color-active);
 `),V(`disabled`,`
 cursor: not-allowed;
 opacity: var(--n-opacity-disabled);
 `)])]),$e=[`onFocusin`,`onFocusout`];function et(t,n,r){let a=[],o=!1;for(let s=0;s<t.length;++s){let c=t[s],l=c.type?.name;l===`RadioButton`&&(o=!0);let u=c.props;if(l!==`RadioButton`){a.push(c);continue}if(s===0)a.push(c);else{let t=a[a.length-1].props,o=n===t.value,s=t.disabled,l=n===u.value,d=u.disabled,f=(o?2:0)+ +!s,p=(l?2:0)+ +!d,m={[`${r}-radio-group__splitor--disabled`]:s,[`${r}-radio-group__splitor--checked`]:o},h={[`${r}-radio-group__splitor--disabled`]:d,[`${r}-radio-group__splitor--checked`]:l},g=f<p?h:m;a.push((R(),e(`div`,{key:1,class:i([`${r}-radio-group__splitor`,g])},null,2)),c)}}return{children:a,isButtonGroup:o}}var tt={...H.props,name:String,options:Array,labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},value:[String,Number,Boolean],defaultValue:{type:[String,Number,Boolean],default:null},size:String,disabled:{type:Boolean,default:void 0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array]},nt=U({name:`RadioGroup`,props:tt,setup(e){let r=c(null),{mergedSizeRef:i,mergedDisabledRef:o,nTriggerFormChange:s,nTriggerFormInput:l,nTriggerFormBlur:u,nTriggerFormFocus:d}=ne(e),{mergedClsPrefixRef:f,inlineThemeDisabled:p,mergedRtlRef:m}=G(e),h=H(`Radio`,`-radio-group`,Qe,me,e,f),g=c(e.defaultValue),v=n(e,`value`),y=Oe(v,g);function b(t){let{onUpdateValue:n,"onUpdate:value":r}=e;n&&a(n,t),r&&a(r,t),g.value=t,s(),l()}function x(e){let{value:t}=r;t&&(t.contains(e.relatedTarget)||d())}function S(e){let{value:t}=r;t&&(t.contains(e.relatedTarget)||u())}L(We,{mergedClsPrefixRef:f,nameRef:n(e,`name`),valueRef:y,disabledRef:o,mergedSizeRef:i,doUpdateValue:b});let C=I(`Radio`,m,f),w=A(()=>{let{value:e}=i,{common:{cubicBezierEaseInOut:t},self:{buttonBorderColor:n,buttonBorderColorActive:r,buttonBorderRadius:a,buttonBoxShadow:o,buttonBoxShadowFocus:s,buttonBoxShadowHover:c,buttonColor:l,buttonColorActive:u,buttonTextColor:d,buttonTextColorActive:f,buttonTextColorHover:p,opacityDisabled:m,[_(`buttonHeight`,e)]:g,[_(`fontSize`,e)]:v}}=h.value;return{"--n-font-size":v,"--n-bezier":t,"--n-button-border-color":n,"--n-button-border-color-active":r,"--n-button-border-radius":a,"--n-button-box-shadow":o,"--n-button-box-shadow-focus":s,"--n-button-box-shadow-hover":c,"--n-button-color":l,"--n-button-color-active":u,"--n-button-text-color":d,"--n-button-text-color-hover":p,"--n-button-text-color-active":f,"--n-height":g,"--n-opacity-disabled":m}}),T=p?t(`radio-group`,A(()=>i.value[0]),w,e):void 0;return{selfElRef:r,rtlEnabled:C,mergedClsPrefix:f,mergedValue:y,handleFocusout:S,handleFocusin:x,cssVars:p?void 0:w,themeClass:T?.themeClass,onRender:T?.onRender}},render(){let{mergedValue:t,mergedClsPrefix:n,handleFocusin:r,handleFocusout:a}=this,{options:o,labelField:s,valueField:c}=this.$props,{children:l,isButtonGroup:u}=et(o?o.map(e=>{let t=e[c];return R(),j(Je,{key:typeof t==`boolean`?`__n_${t}`:t,value:t,disabled:e.disabled,label:e[s]},null,8,[`value`,`disabled`,`label`])}):de(Ze(this)),t,n);return this.onRender?.(),R(),e(`div`,{onFocusin:r,onFocusout:a,ref:`selfElRef`,class:i([`${n}-radio-group`,this.rtlEnabled&&`${n}-radio-group--rtl`,this.themeClass,u&&`${n}-radio-group--button-group`]),style:g(this.cssVars)},[w(()=>l)],46,$e)}}),rt={class:`picker`},it={key:0,class:`loading dt-faint`},at={key:1,class:`notice`},ot={key:2,class:`notice`},st={key:0,class:`tiny warn`},ct=K(U({__name:`ImagenModelPicker`,props:{provider:{},modelValue:{}},emits:[`update:modelValue`],setup(t,{emit:n}){let r=t,i=n;W(()=>Ee());let a=A(()=>xe(r.provider)),o=A(()=>_e(r.provider)),s=A(()=>ge.value===null),c=A(()=>o.value.map(e=>({label:e.label,value:e.id,note:e.note,known:e.known})));function l(e){let t=e.known===!1?`未收录，参数按同族推定`:e.note||``;return z(`div`,{class:`dt-opt`},[z(`span`,{class:`dt-opt-name`},String(e.label)),t?z(`span`,{class:e.known===!1?`dt-opt-tag dt-opt-tag-warn`:`dt-opt-note`},t):null])}let u=A({get:()=>String(r.modelValue??``),set:e=>i(`update:modelValue`,e)});ee(o,e=>{e.length&&(e.some(e=>e.id===u.value)||(u.value=e[0].id))},{immediate:!0});let f=A(()=>o.value.find(e=>e.id===u.value));return(n,r)=>(R(),e(`div`,rt,[s.value?(R(),e(`div`,it,[J(B(le),{size:12}),r[1]||=F(` 正在查可用模型… `,-1)])):a.value?.configured?a.value?.error||B(be)?(R(),e(`p`,ot,` 连不上 `+y(a.value?.label??t.provider)+`：`+y(a.value?.error||B(be)),1)):(R(),e(d,{key:3},[J(B(X),{value:u.value,"onUpdate:value":r[0]||=e=>u.value=e,options:c.value,"render-label":l,"consistent-menu-width":!1},null,8,[`value`,`options`]),f.value&&!f.value.known?(R(),e(`p`,st,` 这个模型不在收录表里，尺寸与画质选项按 gpt-image 系列推定，可能不完全适用。 `)):b(``,!0)],64)):(R(),e(`p`,at,[F(` 还没填 `+y(a.value?.label??t.provider)+` 的 API Key。 `,1),J(B(ye),{to:`/models`,class:`link`},{default:C(()=>[...r[2]||=[F(`去设置`,-1)]]),_:1}),r[3]||=F(` 填上之后才能用云端底图。 `,-1)]))]))}}),[[`__scopeId`,`data-v-e742735b`]]),lt={class:`field`},ut={class:`head`},dt={class:`name`},ft={class:`key dt-mono`},pt={key:4,class:`ranged`},mt=K(U({__name:`ParamField`,props:{param:{},modelValue:{},provider:{}},emits:[`update:modelValue`],setup(t,{emit:n}){let r=t,i=n,a=A({get:()=>r.modelValue,set:e=>i(`update:modelValue`,e)}),o=A(()=>(r.param.options??[]).map(e=>({label:String(e),value:e}))),s=A(()=>(r.param.type===`float`||r.param.type===`int`)&&r.param.min!==void 0&&r.param.max!==void 0),c=A(()=>r.param.type===`int`?1:.05);return(n,r)=>(R(),e(`div`,lt,[O(`div`,ut,[O(`span`,dt,y(t.param.label),1),t.param.note?(R(),j(B(Ae),{key:0,trigger:`hover`,style:{maxWidth:`320px`}},{trigger:C(()=>[...r[7]||=[O(`span`,{class:`hint`},`?`,-1)]]),default:C(()=>[F(` `+y(t.param.note),1)]),_:1})):b(``,!0),O(`span`,ft,y(t.param.key),1)]),t.param.widget===`imagen-model`?(R(),j(ct,{key:0,provider:t.provider??`openai`,modelValue:a.value,"onUpdate:modelValue":r[0]||=e=>a.value=e},null,8,[`provider`,`modelValue`])):t.param.type===`string`?(R(),j(B(ae),{key:1,value:a.value,"onUpdate:value":r[1]||=e=>a.value=e,type:t.param.multiline?`textarea`:`text`,autosize:t.param.multiline?{minRows:2,maxRows:6}:void 0,placeholder:String(t.param.default??``)},null,8,[`value`,`type`,`autosize`,`placeholder`])):t.param.type===`enum`?(R(),j(B(X),{key:2,value:a.value,"onUpdate:value":r[2]||=e=>a.value=e,options:o.value},null,8,[`value`,`options`])):t.param.type===`bool`?(R(),j(B(ue),{key:3,value:a.value,"onUpdate:value":r[3]||=e=>a.value=e},null,8,[`value`])):s.value?(R(),e(`div`,pt,[J(B(se),{value:a.value,"onUpdate:value":r[4]||=e=>a.value=e,min:t.param.min,max:t.param.max,step:c.value,tooltip:!1},null,8,[`value`,`min`,`max`,`step`]),J(B(oe),{value:a.value,"onUpdate:value":r[5]||=e=>a.value=e,min:t.param.min,max:t.param.max,step:c.value,size:`small`,class:`num`},null,8,[`value`,`min`,`max`,`step`])])):(R(),j(B(oe),{key:5,value:a.value,"onUpdate:value":r[6]||=e=>a.value=e,min:t.param.min,max:t.param.max,step:c.value,class:`full`},null,8,[`value`,`min`,`max`,`step`]))]))}}),[[`__scopeId`,`data-v-0382971b`]]),ht={class:`ref`},gt={class:`head`},_t={class:`d1`},vt={class:`d2 dt-faint`},yt={key:1,class:`picked`},bt={class:`thumb dt-swatch`},xt=[`src`],St={class:`info`},Ct=[`title`],wt={class:`hint dt-faint`},Tt={class:`slider`},Et={class:`val dt-mono`},Dt={key:1,class:`hint dt-faint`},Ot=K(U({__name:`ReferenceInput`,props:{modelValue:{},denoise:{},showDenoise:{type:Boolean,default:!0},note:{default:``}},emits:[`update:modelValue`,`update:denoise`],setup(t,{emit:n}){let r=t,i=n,a=c(``),o=c(``),s=c(!1),l=c(!1),u=c(null),f=Z();async function p(e){if(e){s.value=!0;try{let t=new FormData;t.append(`image`,e);let n=await fetch(`/api/uploads`,{method:`POST`,body:t}),s=await n.json();if(!n.ok)throw Error(s?.error??`HTTP ${n.status}`);i(`update:modelValue`,s.name),o.value=s.original??e.name,URL.revokeObjectURL(a.value),a.value=URL.createObjectURL(e),r.denoise>=.99&&i(`update:denoise`,.6)}catch(e){f.error(String(e.message))}finally{s.value=!1}}}function h(){URL.revokeObjectURL(a.value),a.value=``,o.value=``,i(`update:modelValue`,``),i(`update:denoise`,1)}function g(e){l.value=!1,p(e.dataTransfer?.files?.[0])}function _(e){return e<=.35?`几乎照抄原图，只做细节翻新`:e<=.55?`保留原图结构，换材质表现`:e<=.75?`借用大色调与构成，重新生成`:`基本无视原图`}return(n,r)=>(R(),e(`div`,ht,[O(`div`,gt,[r[6]||=O(`span`,{class:`name`},`参考图`,-1),r[7]||=O(`span`,{class:`opt dt-faint`},`可选`,-1),t.modelValue?(R(),e(`button`,{key:0,class:`clear`,onClick:h},`移除`)):b(``,!0)]),t.modelValue?(R(),e(`div`,yt,[O(`span`,bt,[O(`img`,{src:a.value,alt:``},null,8,xt)]),O(`div`,St,[O(`p`,{class:`fn dt-mono`,title:o.value},y(o.value),9,Ct),t.showDenoise?(R(),e(d,{key:0},[O(`p`,wt,y(_(t.denoise)),1),O(`div`,Tt,[J(B(se),{value:t.denoise,min:.1,max:1,step:.05,tooltip:!1,"onUpdate:value":r[3]||=e=>i(`update:denoise`,e)},null,8,[`value`]),O(`span`,Et,y(t.denoise.toFixed(2)),1)]),r[8]||=O(`p`,{class:`lab dt-label`},`重绘幅度`,-1)],64)):t.note?(R(),e(`p`,Dt,y(t.note),1)):b(``,!0)])])):(R(),e(`div`,{key:0,class:m([`drop`,{on:l.value,busy:s.value}]),onDragover:r[0]||=T(e=>l.value=!0,[`prevent`]),onDragleave:r[1]||=e=>l.value=!1,onDrop:T(g,[`prevent`]),onClick:r[2]||=e=>u.value?.click()},[O(`p`,_t,y(s.value?`上传中…`:`拖入图片，或点击选择`),1),O(`p`,vt,y(t.note||`留空则为纯文生图`),1)],34)),O(`input`,{ref_key:`input`,ref:u,type:`file`,accept:`image/png,image/jpeg,image/webp,image/bmp`,hidden:``,onChange:r[4]||=e=>p(e.target.files?.[0])},null,544),t.modelValue?(R(),j(B(Q),{key:2,size:`tiny`,tertiary:``,class:`swap`,onClick:r[5]||=e=>u.value?.click()},{default:C(()=>[...r[9]||=[F(` 换一张 `,-1)]]),_:1})):b(``,!0)]))}}),[[`__scopeId`,`data-v-905bc3cf`]]),kt={class:`rows`},At={class:`text`},jt={class:`t`},Mt={class:`lab`},Nt={class:`dt-muted`},Pt={key:0,class:`items dt-mono dt-faint`},Ft={class:`acts`},It=K(U({__name:`SelfCheck`,setup(t){let n=he(),r=Z(),i=c(null),a=c(!1),o=c(!1);async function s(){try{i.value=await q.checks()}catch{i.value=null}}W(s);let l=A(()=>i.value?.checks.filter(e=>e.status!==`ok`)??[]),u=A(()=>!o.value&&l.value.length>0),f=A(()=>l.value.some(e=>e.status===`fail`));async function p(e){if(e===`open-models`){n.push(`/models`);return}if(e===`restart-comfy`){a.value=!0;try{await q.comfyRestart(),r.success(`ComfyUI 已重启`),await s()}catch(e){r.error(String(e.message))}finally{a.value=!1}}}return(t,n)=>(R(),j(x,{name:`page`},{default:C(()=>[u.value?(R(),e(`section`,{key:0,class:m([`check dt-panel dt-enter`,{fatal:f.value}])},[O(`div`,kt,[(R(!0),e(d,null,M(l.value,t=>(R(),e(`div`,{key:t.key,class:`row`},[O(`span`,{class:m([`dot`,t.status])},null,2),O(`div`,At,[O(`p`,jt,[O(`span`,Mt,y(t.label),1),O(`span`,Nt,y(t.detail),1)]),t.items?.length?(R(),e(`p`,Pt,y(t.items.join(`　`)),1)):b(``,!0)]),t.fix?(R(),j(B(Q),{key:0,size:`tiny`,loading:a.value,onClick:e=>p(t.fix)},{default:C(()=>[F(y(t.fix===`open-models`?`去处理`:`重启 ComfyUI`),1)]),_:2},1032,[`loading`,`onClick`])):b(``,!0)]))),128))]),O(`div`,Ft,[O(`button`,{class:`link`,onClick:s},`重新检查`),O(`button`,{class:`link`,onClick:n[0]||=e=>o.value=!0},`本次忽略`)])],2)):b(``,!0)]),_:1}))}}),[[`__scopeId`,`data-v-b413ccfc`]]),Lt={class:`card dt-panel`},Rt={class:`thumb dt-swatch`},zt=[`src`],Bt={class:`meta`},Vt={class:`row`},Ht={class:`dt-mono seed`},Ut={key:0,class:`dt-mono pct`},Wt=[`title`],Gt={class:`actions`},Kt=K(U({__name:`JobCard`,props:{job:{}},emits:[`cancel`],setup(t){let n=t,r=A(()=>n.job.status===`succeeded`),i=A(()=>n.job.status===`failed`||n.job.status===`canceled`),a=A(()=>n.job.status===`queued`||n.job.status===`running`),o=A(()=>String(n.job.params?.seed??``)),s=A(()=>{switch(n.job.status){case`succeeded`:return{text:`完成`,color:`var(--dt-ok)`};case`failed`:return{text:`失败`,color:`var(--dt-danger)`};case`canceled`:return{text:`已取消`,color:`var(--dt-ink-faint)`};case`running`:return{text:n.job.stage||`生成中`,color:`var(--dt-accent)`};default:return{text:`排队中`,color:`var(--dt-ink-faint)`}}});return(n,c)=>(R(),e(`div`,Lt,[O(`div`,Rt,[r.value?(R(),j(B(ye),{key:0,to:`/material/${t.job.material_id}`,class:`thumblink`},{default:C(()=>[O(`img`,{src:B(te)(t.job.material_id,`basecolor.png`),alt:``,loading:`lazy`},null,8,zt)]),_:1},8,[`to`])):(R(),e(`div`,{key:1,class:m([`placeholder`,{"dt-sheen":a.value}])},[O(`span`,{style:g({color:s.value.color}),class:`state`},y(s.value.text),5),a.value?(R(),j(B(ce),{key:0,type:`line`,percentage:Math.round(t.job.progress*100),"show-indicator":!1,height:4,class:`bar`},null,8,[`percentage`])):b(``,!0)],2))]),O(`div`,Bt,[O(`div`,Vt,[O(`span`,Ht,`种子 `+y(o.value),1),a.value?(R(),e(`span`,Ut,y(Math.round(t.job.progress*100))+`%`,1)):b(``,!0)]),i.value&&t.job.error?(R(),e(`p`,{key:0,class:`err`,title:t.job.error},y(t.job.error),9,Wt)):b(``,!0),O(`div`,Gt,[r.value?(R(),j(B(ye),{key:0,to:`/material/${t.job.material_id}`},{default:C(()=>[J(B(Q),{size:`tiny`,tertiary:``},{default:C(()=>[...c[1]||=[F(`查看通道`,-1)]]),_:1})]),_:1},8,[`to`])):b(``,!0),a.value?(R(),j(B(Q),{key:1,size:`tiny`,tertiary:``,onClick:c[0]||=e=>n.$emit(`cancel`,t.job.id)},{default:C(()=>[...c[2]||=[F(`取消`,-1)]]),_:1})):b(``,!0)])])]))}}),[[`__scopeId`,`data-v-3b1885ea`]]),qt={class:`dt-page dt-page-wide page`},Jt={class:`panel dt-glass`},Yt={class:`pad`},Xt={key:0,class:`desc dt-muted`},Zt={key:1,class:`notice`},Qt={key:0,class:`pad fields`},$t={key:1,class:`expanded dt-mono`},en={key:3,class:`warn-line`},tn={class:`field`},nn={key:1,class:`advanced`},rn={class:`fields`},an={class:`pad submit`},on={key:0,class:`cost dt-mono`},sn={key:1,class:`tiny dt-faint`},cn={key:2,class:`tiny warn`},ln={class:`results`},un={key:0,class:`empty dt-faint`},dn={key:1,class:`grid`},fn=K(U({__name:`GenerateView`,setup(t){let n=Z(),r=c(``),i=c(4),a=c({}),o=c(!1),s=c(``),l=A(()=>$.value.find(e=>e.id===r.value)),u={realistic:`写实`,stylized:`手绘`},f=A(()=>$.value.map(e=>({label:e.name,value:e.id,style:u[e.style]??e.style,cloud:!!e.source})));function p(e){return z(`div`,{class:`dt-opt`},[z(`span`,{class:`dt-opt-name`},String(e.label)),z(`span`,{class:`dt-opt-tag`},String(e.style??``)),z(`span`,{class:e.cloud?`dt-opt-tag dt-opt-tag-accent`:`dt-opt-tag`},e.cloud?`云端底图`:`本地`)])}ee($,e=>{if(r.value||!e.length)return;let t=``;try{let e=sessionStorage.getItem(`dt.refill`);e&&(t=JSON.parse(e).workflow_id)}catch{t=``}r.value=e.find(e=>e.id===t)?.id??e[0].id},{immediate:!0}),ee(l,(e,t)=>{if(!e)return;let n={};for(let t of[...e.params,...e.advanced])n[t.key]=t.default??h(t);if(t){let e=new Map([...t.params,...t.advanced].map(e=>[e.key,e.default??h(e)]));for(let[t,r]of Object.entries(a.value))t in n&&(!e.has(t)||e.get(t)===r||typeof r==typeof n[t]&&(n[t]=r))}Object.assign(n,m(e.id)),a.value=n},{immediate:!0});function m(e){let t=sessionStorage.getItem(`dt.refill`);if(!t)return{};sessionStorage.removeItem(`dt.refill`);try{let n=JSON.parse(t);if(n.workflow_id!==e)return{};let{seed:r,...i}=n.params;return i}catch{return{}}}function h(e){switch(e.type){case`bool`:return!1;case`int`:case`float`:return e.min??0;case`enum`:return e.options?.[0]??``;default:return``}}let _=A(()=>l.value?.params.find(e=>e.key===`prompt`)),v=A(()=>l.value?.params.filter(e=>e.key!==`prompt`&&e.type!==`image`&&!e.hidden)??[]),x=A(()=>l.value?.params.find(e=>e.type===`image`&&!e.hidden)),S=A(()=>!!x.value),w=A(()=>l.value?.advanced.some(e=>e.key===`denoise`)??!1),T=A(()=>l.value?.advanced.filter(e=>!e.hidden&&!(S.value&&w.value&&e.key===`denoise`))??[]),E=A(()=>l.value?.source?.provider??``),D=A(()=>!!E.value),k=A({get:()=>String(a.value.reference??``),set:e=>a.value.reference=e}),N=A({get:()=>Number(a.value.denoise??1),set:e=>a.value.denoise=e}),P=A(()=>{let e=_.value;return e?`${e.prefix??``}${String(a.value[e.key]??``)}${e.suffix??``}`:``}),I=A(()=>!!l.value&&!o.value&&String(a.value.prompt??``).trim()!==``);async function L(){if(!(!l.value||!I.value)){o.value=!0;try{let e=await q.generate({workflow_id:l.value.id,params:{...a.value},variants:i.value,name:String(a.value.prompt??``).slice(0,60)});e.forEach(ve),s.value=e[0]?.batch_id??``,n.success(`已提交 ${e.length} 个变体`)}catch(e){n.error(String(e.message))}finally{o.value=!1}}}async function V(e){try{await q.cancelJob(e)}catch(e){n.error(String(e.message))}}let H=A(()=>{if(s.value)return De(s.value);let e=we.value[0];return e?.batch_id?De(e.batch_id):we.value.slice(0,4)}),te=A(()=>!Te.value?.ready),U=A(()=>{if(!D.value)return``;let e={low:`约 $0.01`,medium:`约 $0.05`,high:`约 $0.13~0.22`}[String(a.value.api_quality??`medium`)];return e?`${e}/张 × ${i.value} 张（1024² 参考值，实际以服务端回报为准）`:`将调用 ${i.value} 次云端接口，按 token 计费`}),W=A(()=>D.value&&String(a.value.api_quality??``)===`high`),G=A(()=>Number(a.value.tile_fix??0));return(t,n)=>(R(),e(`div`,qt,[J(je,{class:`span`,title:`生成台`,desc:`选一个风格预设、写句提示词，出一整套 PBR 通道。一次多出几张变体再挑，比反复重来快。`}),J(It,{class:`span`}),O(`section`,Jt,[O(`div`,Yt,[n[5]||=O(`p`,{class:`dt-label`},`风格预设`,-1),J(B(X),{value:r.value,"onUpdate:value":n[0]||=e=>r.value=e,options:f.value,"render-label":p,class:`styles`},null,8,[`value`,`options`]),l.value?(R(),e(`p`,Xt,y(l.value.description),1)):b(``,!0),l.value?.license_notice&&!l.value.license_notice.commercial?(R(),e(`p`,Zt,y(l.value.license_notice.component)+` 为研究用途许可，产出不可商用 `,1)):b(``,!0)]),l.value?(R(),e(`div`,Qt,[_.value?(R(),j(mt,{key:0,param:_.value,modelValue:a.value[_.value.key],"onUpdate:modelValue":n[1]||=e=>a.value[_.value.key]=e},null,8,[`param`,`modelValue`])):b(``,!0),_.value&&(_.value.prefix||_.value.suffix)?(R(),e(`p`,$t,y(P.value),1)):b(``,!0),S.value?(R(),j(Ot,{key:2,modelValue:k.value,"onUpdate:modelValue":n[2]||=e=>k.value=e,denoise:N.value,"onUpdate:denoise":n[3]||=e=>N.value=e,"show-denoise":w.value,note:D.value?`发给云端做图生图；本地不参与`:``},null,8,[`modelValue`,`denoise`,`show-denoise`,`note`])):b(``,!0),(R(!0),e(d,null,M(v.value,e=>(R(),j(mt,{key:e.key,param:e,provider:E.value,modelValue:a.value[e.key],"onUpdate:modelValue":t=>a.value[e.key]=t},null,8,[`param`,`provider`,`modelValue`,`onUpdate:modelValue`]))),128)),D.value&&!G.value?(R(),e(`p`,en,[...n[6]||=[F(` 无缝重整为 0 时直接分解云端底图，`,-1),O(`b`,null,`产物不保证可平铺`,-1),F(`—— 三维预览里会看到接缝。好处是不经过本地 SDXL，不用下底模。 `,-1)]])):b(``,!0),O(`div`,tn,[n[7]||=O(`div`,{class:`head`},[O(`span`,{class:`name`},`变体数`)],-1),J(B(nt),{value:i.value,"onUpdate:value":n[4]||=e=>i.value=e,size:`small`},{default:C(()=>[(R(),e(d,null,M([1,2,4,6],e=>J(B(Xe),{key:e,value:e},{default:C(()=>[F(y(e),1)]),_:2},1032,[`value`])),64))]),_:1},8,[`value`]),n[8]||=O(`p`,{class:`tiny dt-faint`},`一次多出几张再挑，比反复重来快`,-1)])])):b(``,!0),l.value&&l.value.advanced.length?(R(),e(`div`,nn,[J(B(Le),null,{default:C(()=>[J(B(Ve),{title:`高级参数`,name:`adv`},{default:C(()=>[O(`div`,rn,[(R(!0),e(d,null,M(T.value,e=>(R(),j(mt,{key:e.key,param:e,provider:E.value,modelValue:a.value[e.key],"onUpdate:modelValue":t=>a.value[e.key]=t},null,8,[`param`,`provider`,`modelValue`,`onUpdate:modelValue`]))),128))])]),_:1})]),_:1})])):b(``,!0),O(`div`,an,[U.value?(R(),e(`p`,on,y(U.value),1)):b(``,!0),J(B(Q),{type:`primary`,block:``,disabled:!I.value,loading:o.value,onClick:L},{default:C(()=>[...n[9]||=[F(` 生成 `,-1)]]),_:1},8,[`disabled`,`loading`]),W.value?(R(),e(`p`,sn,` high 档每张要两三分钟，比本地整条管线还慢，先用 medium 试提示词更划算。 `)):b(``,!0),te.value?(R(),e(`p`,cn,y(B(Te)?.reason||`ComfyUI 尚未就绪，任务会排队等待`),1)):b(``,!0)])]),O(`section`,ln,[H.value.length?(R(),e(`div`,dn,[(R(!0),e(d,null,M(H.value,(e,t)=>(R(),j(Kt,{key:e.id,job:e,class:`dt-enter`,style:g({animationDelay:`${Math.min(t,6)*45}ms`}),onCancel:V},null,8,[`job`,`style`]))),128))])):(R(),e(`div`,un,[...n[10]||=[O(`p`,null,`还没有生成记录。`,-1),O(`p`,{class:`tiny`},`左侧填个提示词，点生成。`,-1)]]))])]))}}),[[`__scopeId`,`data-v-d089a726`]]);export{fn as default};