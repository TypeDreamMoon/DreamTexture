import{$t as e,A as t,B as n,Dt as r,Nt as i,On as a,Ot as o,Pn as s,R as c,Rt as l,U as u,Xt as d,Yt as f,Zt as p,h as m,hn as h,jt as g,k as _,nn as v,o as y,q as b,s as x,tn as S,w as C,xn as w}from"./client-Ruw1_y6D.js";import{_ as T}from"./index-DmsKMie1.js";var E=r([r(`@keyframes spin-rotate`,`
 from {
 transform: rotate(0);
 }
 to {
 transform: rotate(360deg);
 }
 `),o(`spin-container`,`
 position: relative;
 `,[o(`spin-body`,`
 position: absolute;
 top: 50%;
 left: 50%;
 transform: translateX(-50%) translateY(-50%);
 `,[m()])]),o(`spin-body`,`
 display: inline-flex;
 align-items: center;
 justify-content: center;
 flex-direction: column;
 `),o(`spin`,`
 display: inline-flex;
 height: var(--n-size);
 width: var(--n-size);
 font-size: var(--n-size);
 color: var(--n-color);
 `,[g(`rotate`,`
 animation: spin-rotate 2s linear infinite;
 `)]),o(`spin-description`,`
 display: inline-block;
 font-size: var(--n-font-size);
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 margin-top: 8px;
 `),o(`spin-content`,`
 opacity: 1;
 transition: opacity .3s var(--n-bezier);
 pointer-events: all;
 `,[g(`spinning`,`
 user-select: none;
 -webkit-user-select: none;
 pointer-events: none;
 opacity: var(--n-opacity-spinning);
 `)])]),D={small:20,medium:18,large:16},O={..._.props,contentClass:String,contentStyle:[Object,String],description:String,size:{type:[String,Number],default:`medium`},show:{type:Boolean,default:!0},rotate:{type:Boolean,default:!0},spinning:{type:Boolean,validator:()=>!0,default:void 0},delay:Number,...x,strokeWidth:Number},k=v({name:`Spin`,props:O,slots:Object,setup(e){let{mergedClsPrefixRef:n,inlineThemeDisabled:r}=b(e),o=_(`Spin`,`-spin`,E,T,e,n),s=f(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:r}=o.value,{opacitySpinning:a,color:s,textColor:l}=r;return{"--n-bezier":n,"--n-opacity-spinning":a,"--n-size":typeof t==`number`?c(t):r[i(`size`,t)],"--n-color":s,"--n-text-color":l}}),l=r?t(`spin`,f(()=>{let{size:t}=e;return typeof t==`number`?String(t):t[0]}),s,e):void 0,u=C(e,[`spinning`,`show`]),d=a(!1);return w(t=>{let n;if(u.value){let{delay:r}=e;if(r){n=window.setTimeout(()=>{d.value=!0},r),t(()=>{clearTimeout(n)});return}}d.value=u.value}),{mergedClsPrefix:n,active:d,mergedStrokeWidth:f(()=>{let{strokeWidth:t}=e;if(t!==void 0)return t;let{size:n}=e;return D[typeof n==`number`?`medium`:n]}),cssVars:r?void 0:s,themeClass:l?.themeClass,onRender:l?.onRender}},render(){let{$slots:t,mergedClsPrefix:r,description:i}=this,a=t.icon&&this.rotate,o=(i||t.description)&&(h(),e(`div`,{class:n(`${r}-spin-description`)},[u(()=>i||t.description?.())],2)),c=t.icon?(h(),e(`div`,{key:1,class:n([`${r}-spin-body`,this.themeClass])},[d(`div`,{class:n([`${r}-spin`,a&&`${r}-spin--rotate`]),style:s(t.default?``:this.cssVars)},[u(()=>t.icon())],6),u(()=>o)],2)):(h(),e(`div`,{key:2,class:n([`${r}-spin-body`,this.themeClass])},[(h(),p(y,{clsPrefix:r,style:s(t.default?``:this.cssVars),stroke:this.stroke,"stroke-width":this.mergedStrokeWidth,radius:this.radius,scale:this.scale,class:n(`${r}-spin`)},null,8,[`clsPrefix`,`style`,`stroke`,`stroke-width`,`radius`,`scale`,`class`])),u(()=>o)],2));return this.onRender?.(),t.default?(h(),e(`div`,{key:3,class:n([`${r}-spin-container`,this.themeClass]),style:s(this.cssVars)},[d(`div`,{class:n([`${r}-spin-content`,this.active&&`${r}-spin-content--spinning`,this.contentClass]),style:s(this.contentStyle)},[u(()=>t.default?.())],6),S(l,{name:`fade-in-transition`},{default:()=>this.active?c:null},1024)],6)):c}});export{k as t};