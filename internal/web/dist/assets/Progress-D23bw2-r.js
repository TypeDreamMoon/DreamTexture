import{$t as e,A as t,B as n,D as r,Dt as i,Gt as a,H as o,Lt as s,Nt as c,Ot as l,Pn as u,U as d,Xt as f,Yt as p,Zt as m,hn as h,jt as g,k as _,nn as v,q as y}from"./client-UpnskrDa.js";import{B as b,R as x,V as S,et as C,y as w,z as T}from"./index-1kS9fx0V.js";var E=[`id`],D=[`stop-color`],O=[`stop-color`],k=[`viewBox`],A=[`d`,`stroke-width`],j=[`d`,`stroke-width`],M={success:(h(),m(T)),error:(h(),m(S)),warning:(h(),m(x)),info:(h(),m(b))},N=v({name:`ProgressCircle`,props:{clsPrefix:{type:String,required:!0},status:{type:String,required:!0},strokeWidth:{type:Number,required:!0},fillColor:[String,Object],railColor:String,railStyle:[String,Object],percentage:{type:Number,default:0},offsetDegree:{type:Number,default:0},showIndicator:{type:Boolean,required:!0},indicatorTextColor:String,unit:String,viewBoxWidth:{type:Number,required:!0},gapDegree:{type:Number,required:!0},gapOffsetDegree:{type:Number,default:0}},setup(t,{slots:i}){let o=p(()=>{let e=`gradient`,{fillColor:n}=t;return typeof n==`object`?`${e}-${s(JSON.stringify(n))}`:e});function c(e,n,r,i){let{gapDegree:a,viewBoxWidth:s,strokeWidth:c}=t,l=50+c/2,u=`M ${l},${l} m 0,50
      a 50,50 0 1 1 0,-100
      a 50,50 0 1 1 0,100`,d=Math.PI*2*50;return{pathString:u,pathStyle:{stroke:i===`rail`?r:typeof t.fillColor==`object`?`url(#${o.value})`:r,strokeDasharray:`${Math.min(e,100)/100*(d-a)}px ${s*8}px`,strokeDashoffset:`-${a/2}px`,transformOrigin:n?`center`:void 0,transform:n?`rotate(${n}deg)`:void 0}}}let l=()=>{let n=typeof t.fillColor==`object`,r=n?t.fillColor.stops[0]:``,i=n?t.fillColor.stops[1]:``;return n&&(h(),e(`defs`,null,[f(`linearGradient`,{id:o.value,x1:`0%`,y1:`100%`,x2:`100%`,y2:`0%`},[f(`stop`,{offset:`0%`,"stop-color":r},null,8,D),f(`stop`,{offset:`100%`,"stop-color":i},null,8,O)],8,E)]))};return()=>{let{fillColor:o,railColor:s,strokeWidth:p,offsetDegree:g,status:_,percentage:v,showIndicator:y,indicatorTextColor:b,unit:x,gapOffsetDegree:S,clsPrefix:C}=t,{pathString:w,pathStyle:T}=c(100,0,s,`rail`),{pathString:E,pathStyle:D}=c(v,g,o,`fill`),O=100+p;return h(),e(`div`,{class:n(`${C}-progress-content`),role:`none`},[f(`div`,{class:n(`${C}-progress-graph`),"aria-hidden":!0},[f(`div`,{class:n(`${C}-progress-graph-circle`),style:u({transform:S?`rotate(${S}deg)`:void 0})},[(h(),e(`svg`,{viewBox:`0 0 ${O} ${O}`},[d(()=>l()),f(`g`,null,[f(`path`,{class:n(`${C}-progress-graph-circle-rail`),d:w,"stroke-width":p,"stroke-linecap":`round`,fill:`none`,style:u(T)},null,14,A)]),f(`g`,null,[f(`path`,{class:n([`${C}-progress-graph-circle-fill`,v===0&&`${C}-progress-graph-circle-fill--empty`]),d:E,"stroke-width":p,"stroke-linecap":`round`,fill:`none`,style:u(D)},null,14,j)])],8,k))],6)],2),y?(h(),e(`div`,{key:0},[i.default?(h(),e(`div`,{key:0,class:n(`${C}-progress-custom-content`),role:`none`},[d(()=>i.default())],2)):(h(),e(a,{key:1},[_==="default"?(h(),e(`div`,{key:1,class:n(`${C}-progress-text`),style:u({color:b}),role:`none`},[f(`span`,{class:n(`${C}-progress-text__percentage`)},[d(()=>v)],2),f(`span`,{class:n(`${C}-progress-text__unit`)},[d(()=>x)],2)],6)):(h(),e(`div`,{key:0,class:n(`${C}-progress-icon`),"aria-hidden":!0},[(h(),m(r,{clsPrefix:C},{default:()=>M[_]},1032,[`clsPrefix`]))],2))],64))])):d(()=>null)],2)}}}),P={success:(h(),m(T)),error:(h(),m(S)),warning:(h(),m(x)),info:(h(),m(b))},F=v({name:`ProgressLine`,props:{clsPrefix:{type:String,required:!0},percentage:{type:Number,default:0},railColor:String,railStyle:[String,Object],fillColor:[String,Object],status:{type:String,required:!0},indicatorPlacement:{type:String,required:!0},indicatorTextColor:String,unit:{type:String,default:`%`},processing:{type:Boolean,required:!0},showIndicator:{type:Boolean,required:!0},height:[String,Number],railBorderRadius:[String,Number],fillBorderRadius:[String,Number]},setup(t,{slots:i}){let o=p(()=>C(t.height)),s=p(()=>typeof t.fillColor==`object`?`linear-gradient(to right, ${t.fillColor?.stops[0]} , ${t.fillColor?.stops[1]})`:t.fillColor),c=p(()=>t.railBorderRadius===void 0?t.height===void 0?``:C(t.height,{c:.5}):C(t.railBorderRadius)),l=p(()=>t.fillBorderRadius===void 0?t.railBorderRadius===void 0?t.height===void 0?``:C(t.height,{c:.5}):C(t.railBorderRadius):C(t.fillBorderRadius));return()=>{let{indicatorPlacement:p,railColor:g,railStyle:_,percentage:v,unit:y,indicatorTextColor:b,status:x,showIndicator:S,processing:C,clsPrefix:w}=t;return h(),e(`div`,{class:n(`${w}-progress-content`),role:`none`},[f(`div`,{class:n(`${w}-progress-graph`),"aria-hidden":!0},[f(`div`,{class:n([`${w}-progress-graph-line`,{[`${w}-progress-graph-line--indicator-${p}`]:!0}])},[f(`div`,{class:n(`${w}-progress-graph-line-rail`),style:u([{backgroundColor:g,height:o.value,borderRadius:c.value},_])},[f(`div`,{class:n([`${w}-progress-graph-line-fill`,C&&`${w}-progress-graph-line-fill--processing`]),style:u({maxWidth:`${t.percentage}%`,background:s.value,height:o.value,lineHeight:o.value,borderRadius:l.value})},[p===`inside`?(h(),e(`div`,{key:0,class:n(`${w}-progress-graph-line-indicator`),style:u({color:b})},[i.default?(h(),e(a,{key:0},[d(()=>i.default())],64)):(h(),e(a,{key:1},[d(()=>`${v}${y}`)],64))],6)):d(()=>null)],6)],6)],2)],2),S&&p===`outside`?(h(),e(`div`,{key:0},[i.default?(h(),e(`div`,{key:0,class:n(`${w}-progress-custom-content`),style:u({color:b}),role:`none`},[d(()=>i.default())],6)):(h(),e(a,{key:1},[x==="default"?(h(),e(`div`,{key:0,role:`none`,class:n(`${w}-progress-icon ${w}-progress-icon--as-text`),style:u({color:b})},[d(()=>v),d(()=>y)],6)):(h(),e(`div`,{key:1,class:n(`${w}-progress-icon`),"aria-hidden":!0},[(h(),m(r,{clsPrefix:w},{default:()=>P[x]},1032,[`clsPrefix`]))],2))],64))])):d(()=>null)],2)}}}),I=[`id`],L=[`stop-color`],R=[`stop-color`],z=[`d`,`stroke-width`],B=[`d`,`stroke-width`],V=[`viewBox`];function H(e,t,n=100){return`m ${n/2} ${n/2-e} a ${e} ${e} 0 1 1 0 ${2*e} a ${e} ${e} 0 1 1 0 -${2*e}`}var U=v({name:`ProgressMultipleCircle`,props:{clsPrefix:{type:String,required:!0},viewBoxWidth:{type:Number,required:!0},percentage:{type:Array,default:[0]},strokeWidth:{type:Number,required:!0},circleGap:{type:Number,required:!0},showIndicator:{type:Boolean,required:!0},fillColor:{type:Array,default:()=>[]},railColor:{type:Array,default:()=>[]},railStyle:{type:Array,default:()=>[]}},setup(t,{slots:r}){let i=p(()=>t.percentage.map((e,n)=>`${Math.PI*e/100*(t.viewBoxWidth/2-t.strokeWidth/2*(1+2*n)-t.circleGap*n)*2}, ${t.viewBoxWidth*8}`)),a=(n,r)=>{let i=t.fillColor[r],a=typeof i==`object`?i.stops[0]:``,o=typeof i==`object`?i.stops[1]:``;return typeof t.fillColor[r]==`object`&&(h(),e(`linearGradient`,{id:`gradient-${r}`,x1:`100%`,y1:`0%`,x2:`0%`,y2:`100%`},[f(`stop`,{offset:`0%`,"stop-color":a},null,8,L),f(`stop`,{offset:`100%`,"stop-color":o},null,8,R)],8,I))};return()=>{let{viewBoxWidth:o,strokeWidth:s,circleGap:c,showIndicator:l,fillColor:p,railColor:m,railStyle:g,percentage:_,clsPrefix:v}=t;return h(),e(`div`,{class:n(`${v}-progress-content`),role:`none`},[f(`div`,{class:n(`${v}-progress-graph`),"aria-hidden":!0},[f(`div`,{class:n(`${v}-progress-graph-circle`)},[(h(),e(`svg`,{viewBox:`0 0 ${o} ${o}`},[f(`defs`,null,[d(()=>_.map((e,t)=>a(e,t)))]),d(()=>_.map((t,r)=>(h(),e(`g`,{key:r},[f(`path`,{class:n(`${v}-progress-graph-circle-rail`),d:H(o/2-s/2*(1+2*r)-c*r,s,o),"stroke-width":s,"stroke-linecap":`round`,fill:`none`,style:u([{strokeDashoffset:0,stroke:m[r]},g[r]])},null,14,z),f(`path`,{class:n([`${v}-progress-graph-circle-fill`,t===0&&`${v}-progress-graph-circle-fill--empty`]),d:H(o/2-s/2*(1+2*r)-c*r,s,o),"stroke-width":s,"stroke-linecap":`round`,fill:`none`,style:u({strokeDasharray:i.value[r],strokeDashoffset:0,stroke:typeof p[r]==`object`?`url(#gradient-${r})`:p[r]})},null,14,B)]))))],8,V))],2)],2),l&&r.default?(h(),e(`div`,{key:0},[f(`div`,{class:n(`${v}-progress-text`)},[d(()=>r.default())],2)])):d(()=>null)],2)}}}),W=i([l(`progress`,{display:`inline-block`},[l(`progress-icon`,`
 color: var(--n-icon-color);
 transition: color .3s var(--n-bezier);
 `),g(`line`,`
 width: 100%;
 display: block;
 `,[l(`progress-content`,`
 display: flex;
 align-items: center;
 `,[l(`progress-graph`,{flex:1})]),l(`progress-custom-content`,{marginLeft:`14px`}),l(`progress-icon`,`
 width: 30px;
 padding-left: 14px;
 height: var(--n-icon-size-line);
 line-height: var(--n-icon-size-line);
 font-size: var(--n-icon-size-line);
 `,[g(`as-text`,`
 color: var(--n-text-color-line-outer);
 text-align: center;
 width: 40px;
 font-size: var(--n-font-size);
 padding-left: 4px;
 transition: color .3s var(--n-bezier);
 `)])]),g(`circle, dashboard`,{width:`120px`},[l(`progress-custom-content`,`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 `),l(`progress-text`,`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 color: inherit;
 font-size: var(--n-font-size-circle);
 color: var(--n-text-color-circle);
 font-weight: var(--n-font-weight-circle);
 transition: color .3s var(--n-bezier);
 white-space: nowrap;
 `),l(`progress-icon`,`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 color: var(--n-icon-color);
 font-size: var(--n-icon-size-circle);
 `)]),g(`multiple-circle`,`
 width: 200px;
 color: inherit;
 `,[l(`progress-text`,`
 font-weight: var(--n-font-weight-circle);
 color: var(--n-text-color-circle);
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 transition: color .3s var(--n-bezier);
 `)]),l(`progress-content`,{position:`relative`}),l(`progress-graph`,{position:`relative`},[l(`progress-graph-circle`,[i(`svg`,{verticalAlign:`bottom`}),l(`progress-graph-circle-fill`,`
 stroke: var(--n-fill-color);
 transition:
 opacity .3s var(--n-bezier),
 stroke .3s var(--n-bezier),
 stroke-dasharray .3s var(--n-bezier);
 `,[g(`empty`,{opacity:0})]),l(`progress-graph-circle-rail`,`
 transition: stroke .3s var(--n-bezier);
 overflow: hidden;
 stroke: var(--n-rail-color);
 `)]),l(`progress-graph-line`,[g(`indicator-inside`,[l(`progress-graph-line-rail`,`
 height: 16px;
 line-height: 16px;
 border-radius: 10px;
 `,[l(`progress-graph-line-fill`,`
 height: inherit;
 border-radius: 10px;
 `),l(`progress-graph-line-indicator`,`
 background: #0000;
 white-space: nowrap;
 text-align: right;
 margin-left: 14px;
 margin-right: 14px;
 height: inherit;
 font-size: 12px;
 color: var(--n-text-color-line-inner);
 transition: color .3s var(--n-bezier);
 `)])]),g(`indicator-inside-label`,`
 height: 16px;
 display: flex;
 align-items: center;
 `,[l(`progress-graph-line-rail`,`
 flex: 1;
 transition: background-color .3s var(--n-bezier);
 `),l(`progress-graph-line-indicator`,`
 background: var(--n-fill-color);
 font-size: 12px;
 transform: translateZ(0);
 display: flex;
 vertical-align: middle;
 height: 16px;
 line-height: 16px;
 padding: 0 10px;
 border-radius: 10px;
 position: absolute;
 white-space: nowrap;
 color: var(--n-text-color-line-inner);
 transition:
 right .2s var(--n-bezier),
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 `)]),l(`progress-graph-line-rail`,`
 position: relative;
 overflow: hidden;
 height: var(--n-rail-height);
 border-radius: 5px;
 background-color: var(--n-rail-color);
 transition: background-color .3s var(--n-bezier);
 `,[l(`progress-graph-line-fill`,`
 background: var(--n-fill-color);
 position: relative;
 border-radius: 5px;
 height: inherit;
 width: 100%;
 max-width: 0%;
 transition:
 background-color .3s var(--n-bezier),
 max-width .2s var(--n-bezier);
 `,[g(`processing`,[i(`&::after`,`
 content: "";
 background-image: var(--n-line-bg-processing);
 animation: progress-processing-animation 2s var(--n-bezier) infinite;
 `)])])])])])]),i(`@keyframes progress-processing-animation`,`
 0% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 100%;
 opacity: 1;
 }
 66% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 0;
 opacity: 0;
 }
 100% {
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 0;
 opacity: 0;
 }
 `)]),G=[`aria-valuenow`,`role`],K={..._.props,processing:Boolean,type:{type:String,default:`line`},gapDegree:Number,gapOffsetDegree:Number,status:{type:String,default:`default`},railColor:[String,Array],railStyle:[String,Array],color:[String,Array,Object],viewBoxWidth:{type:Number,default:100},strokeWidth:{type:Number,default:7},percentage:[Number,Array],unit:{type:String,default:`%`},showIndicator:{type:Boolean,default:!0},indicatorPosition:{type:String,default:`outside`},indicatorPlacement:{type:String,default:`outside`},indicatorTextColor:String,circleGap:{type:Number,default:1},height:Number,borderRadius:[String,Number],fillBorderRadius:[String,Number],offsetDegree:Number},q=v({name:`Progress`,props:K,setup(e){let n=p(()=>e.indicatorPlacement||e.indicatorPosition),r=p(()=>{if(e.gapDegree||e.gapDegree===0)return e.gapDegree;if(e.type===`dashboard`)return 75}),{mergedClsPrefixRef:i,inlineThemeDisabled:a}=y(e),o=_(`Progress`,`-progress`,W,w,e,i),s=p(()=>{let{status:t}=e,{common:{cubicBezierEaseInOut:n},self:{fontSize:r,fontSizeCircle:i,railColor:a,railHeight:s,iconSizeCircle:l,iconSizeLine:u,textColorCircle:d,textColorLineInner:f,textColorLineOuter:p,lineBgProcessing:m,fontWeightCircle:h,[c(`iconColor`,t)]:g,[c(`fillColor`,t)]:_}}=o.value;return{"--n-bezier":n,"--n-fill-color":_,"--n-font-size":r,"--n-font-size-circle":i,"--n-font-weight-circle":h,"--n-icon-color":g,"--n-icon-size-circle":l,"--n-icon-size-line":u,"--n-line-bg-processing":m,"--n-rail-color":a,"--n-rail-height":s,"--n-text-color-circle":d,"--n-text-color-line-inner":f,"--n-text-color-line-outer":p}}),l=a?t(`progress`,p(()=>e.status[0]),s,e):void 0;return{mergedClsPrefix:i,mergedIndicatorPlacement:n,gapDeg:r,cssVars:a?void 0:s,themeClass:l?.themeClass,onRender:l?.onRender}},render(){let{type:t,cssVars:r,indicatorTextColor:i,showIndicator:s,status:c,railColor:l,railStyle:f,color:p,percentage:g,viewBoxWidth:_,strokeWidth:v,mergedIndicatorPlacement:y,unit:b,borderRadius:x,fillBorderRadius:S,height:C,processing:w,circleGap:T,mergedClsPrefix:E,gapDeg:D,gapOffsetDegree:O,themeClass:k,$slots:A,onRender:j}=this;return j?.(),h(),e(`div`,{class:n([k,`${E}-progress`,`${E}-progress--${t}`,`${E}-progress--${c}`]),style:u(r),"aria-valuemax":100,"aria-valuemin":0,"aria-valuenow":g,role:t===`circle`||t===`line`||t===`dashboard`?`progressbar`:`none`},[t===`circle`||t===`dashboard`?(h(),m(N,{key:0,clsPrefix:E,status:c,showIndicator:s,indicatorTextColor:i,railColor:l,fillColor:p,railStyle:f,offsetDegree:this.offsetDegree,percentage:g,viewBoxWidth:_,strokeWidth:v,gapDegree:D===void 0?t===`dashboard`?75:0:D,gapOffsetDegree:O,unit:b},o(A),1032,[`clsPrefix`,`status`,`showIndicator`,`indicatorTextColor`,`railColor`,`fillColor`,`railStyle`,`offsetDegree`,`percentage`,`viewBoxWidth`,`strokeWidth`,`gapDegree`,`gapOffsetDegree`,`unit`])):(h(),e(a,{key:1},[t===`line`?(h(),m(F,{key:0,clsPrefix:E,status:c,showIndicator:s,indicatorTextColor:i,railColor:l,fillColor:p,railStyle:f,percentage:g,processing:w,indicatorPlacement:y,unit:b,fillBorderRadius:S,railBorderRadius:x,height:C},o(A),1032,[`clsPrefix`,`status`,`showIndicator`,`indicatorTextColor`,`railColor`,`fillColor`,`railStyle`,`percentage`,`processing`,`indicatorPlacement`,`unit`,`fillBorderRadius`,`railBorderRadius`,`height`])):(h(),e(a,{key:1},[t===`multiple-circle`?(h(),m(U,{key:0,clsPrefix:E,strokeWidth:v,railColor:l,fillColor:p,railStyle:f,viewBoxWidth:_,percentage:g,showIndicator:s,circleGap:T},o(A),1032,[`clsPrefix`,`strokeWidth`,`railColor`,`fillColor`,`railStyle`,`viewBoxWidth`,`percentage`,`showIndicator`,`circleGap`])):d(()=>null)],64))],64))],14,G)}});export{q as t};