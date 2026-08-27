import{$t as e,A as t,An as n,At as r,B as i,C as a,Dn as o,Dt as s,E as c,Ft as l,Gt as u,Nn as d,Nt as f,Ot as p,Pn as m,Pt as h,Qt as g,Sn as _,U as v,Xt as y,Y as b,Yt as x,Zt as S,_n as C,an as w,c as T,en as E,g as D,gn as O,hn as k,jn as A,jt as j,k as M,l as N,nn as P,q as F,r as I,t as L,tn as R,u as z,x as ee,z as B}from"./client-Do8VSizq.js";import{n as V,t as H}from"./Select-Dkoeg12f.js";import{t as U}from"./Tag-Cjeae2mY.js";import{t as te}from"./Alert-uOU0ATz6.js";import{t as ne}from"./use-message-B4TpC56a.js";import{D as re,O as W,f as G,it as K,lt as q,r as J,tt as Y,y as X}from"./index-DEvJnqPU.js";import{t as ie}from"./PageHeader-Ds_XrywY.js";var ae=()=>(()=>{let e=B(`75be776d8875fa17`);return e[0]||=y(`svg`,{viewBox:`0 0 64 64`,class:`check-icon`},[y(`path`,{d:`M50.42,16.76L22.34,39.45l-8.1-11.46c-1.12-1.58-3.3-1.96-4.88-0.84c-1.58,1.12-1.95,3.3-0.84,4.88l10.26,14.51  c0.56,0.79,1.42,1.31,2.38,1.45c0.16,0.02,0.32,0.03,0.48,0.03c0.8,0,1.57-0.27,2.2-0.78l30.99-25.03c1.5-1.21,1.74-3.42,0.52-4.92  C54.13,15.78,51.93,15.55,50.42,16.76z`})],-1)})(),oe=()=>(()=>{let e=B(`c6eed899356c8404`);return e[0]||=y(`svg`,{viewBox:`0 0 100 100`,class:`line-icon`},[y(`path`,{d:`M80.2,55.5H21.4c-2.8,0-5.1-2.5-5.1-5.5l0,0c0-3,2.3-5.5,5.1-5.5h58.7c2.8,0,5.1,2.5,5.1,5.5l0,0C85.2,53.1,82.9,55.5,80.2,55.5z`})],-1)})(),se=s([p(`checkbox`,`
 font-size: var(--n-font-size);
 outline: none;
 cursor: pointer;
 display: inline-flex;
 flex-wrap: nowrap;
 align-items: flex-start;
 word-break: break-word;
 line-height: var(--n-size);
 --n-merged-color-table: var(--n-color-table);
 `,[j(`show-label`,`line-height: var(--n-label-line-height);`),s(`&:hover`,[p(`checkbox-box`,[r(`border`,`border: var(--n-border-checked);`)])]),s(`&:focus:not(:active)`,[p(`checkbox-box`,[r(`border`,`
 border: var(--n-border-focus);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),j(`inside-table`,[p(`checkbox-box`,`
 background-color: var(--n-merged-color-table);
 `)]),j(`checked`,[p(`checkbox-box`,`
 background-color: var(--n-color-checked);
 `,[p(`checkbox-icon`,[s(`.check-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),j(`indeterminate`,[p(`checkbox-box`,[p(`checkbox-icon`,[s(`.check-icon`,`
 opacity: 0;
 transform: scale(.5);
 `),s(`.line-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),j(`checked, indeterminate`,[s(`&:focus:not(:active)`,[p(`checkbox-box`,[r(`border`,`
 border: var(--n-border-checked);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),p(`checkbox-box`,`
 background-color: var(--n-color-checked);
 border-left: 0;
 border-top: 0;
 `,[r(`border`,{border:`var(--n-border-checked)`})])]),j(`disabled`,{cursor:`not-allowed`},[j(`checked`,[p(`checkbox-box`,`
 background-color: var(--n-color-disabled-checked);
 `,[r(`border`,{border:`var(--n-border-disabled-checked)`}),p(`checkbox-icon`,[s(`.check-icon, .line-icon`,{fill:`var(--n-check-mark-color-disabled-checked)`})])])]),p(`checkbox-box`,`
 background-color: var(--n-color-disabled);
 `,[r(`border`,`
 border: var(--n-border-disabled);
 `),p(`checkbox-icon`,[s(`.check-icon, .line-icon`,`
 fill: var(--n-check-mark-color-disabled);
 `)])]),r(`label`,`
 color: var(--n-text-color-disabled);
 `)]),p(`checkbox-box-wrapper`,`
 position: relative;
 width: var(--n-size);
 flex-shrink: 0;
 flex-grow: 0;
 user-select: none;
 -webkit-user-select: none;
 `),p(`checkbox-box`,`
 position: absolute;
 left: 0;
 top: 50%;
 transform: translateY(-50%);
 height: var(--n-size);
 width: var(--n-size);
 display: inline-block;
 box-sizing: border-box;
 border-radius: var(--n-border-radius);
 background-color: var(--n-color);
 transition: background-color 0.3s var(--n-bezier);
 `,[r(`border`,`
 transition:
 border-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 border-radius: inherit;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border: var(--n-border);
 `),p(`checkbox-icon`,`
 display: flex;
 align-items: center;
 justify-content: center;
 position: absolute;
 left: 1px;
 right: 1px;
 top: 1px;
 bottom: 1px;
 `,[s(`.check-icon, .line-icon`,`
 width: 100%;
 fill: var(--n-check-mark-color);
 opacity: 0;
 transform: scale(0.5);
 transform-origin: center;
 transition:
 fill 0.3s var(--n-bezier),
 transform 0.3s var(--n-bezier),
 opacity 0.3s var(--n-bezier),
 border-color 0.3s var(--n-bezier);
 `),T({left:`1px`,top:`1px`})])]),r(`label`,`
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 user-select: none;
 -webkit-user-select: none;
 padding: var(--n-label-padding);
 font-weight: var(--n-label-font-weight);
 `,[s(`&:empty`,{display:`none`})])]),h(p(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-modal);
 `)),l(p(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-popover);
 `))]),ce=[`id`],le=[`tabindex`,`aria-checked`,`aria-labelledby`,`onKeyup`,`onKeydown`,`onClick`],ue={...M.props,size:String,checked:{type:[Boolean,String,Number],default:void 0},defaultChecked:{type:[Boolean,String,Number],default:!1},value:[String,Number],disabled:{type:Boolean,default:void 0},indeterminate:Boolean,label:String,focusable:{type:Boolean,default:!0},checkedValue:{type:[Boolean,String,Number],default:!0},uncheckedValue:{type:[Boolean,String,Number],default:!1},"onUpdate:checked":[Function,Array],onUpdateChecked:[Function,Array],privateInsideTable:Boolean,onChange:[Function,Array]},Z=P({name:`Checkbox`,props:ue,setup(e){let r=w(Q,null),i=o(null),{mergedClsPrefixRef:s,inlineThemeDisabled:l,mergedRtlRef:u,mergedComponentPropsRef:d}=F(e),p=o(e.defaultChecked),m=n(e,`checked`),h=Y(m,p),g=c(()=>{if(r){let t=r.valueSetRef.value;return t&&e.value!==void 0?t.has(e.value):!1}return h.value===e.checkedValue}),_=z(e,{mergedSize(t){let{size:n}=e;if(n!==void 0)return n;if(r){let{value:e}=r.mergedSizeRef;if(e!==void 0)return e}if(t){let{mergedSize:e}=t;if(e!==void 0)return e.value}return d?.value?.Checkbox?.size||`medium`},mergedDisabled(t){let{disabled:n}=e;if(n!==void 0)return n;if(r){if(r.disabledRef.value)return!0;let{maxRef:{value:e},checkedCountRef:t}=r;if(e!==void 0&&t.value>=e&&!g.value)return!0;let{minRef:{value:n}}=r;if(n!==void 0&&t.value<=n&&g.value)return!0}return t?t.disabled.value:!1}}),{mergedDisabledRef:v,mergedSizeRef:y}=_,b=M(`Checkbox`,`-checkbox`,se,re,e,s);function S(t){if(r&&e.value!==void 0)r.toggleCheckbox(!g.value,e.value);else{let{onChange:n,"onUpdate:checked":r,onUpdateChecked:i}=e,{nTriggerFormInput:o,nTriggerFormChange:s}=_,c=g.value?e.uncheckedValue:e.checkedValue;r&&a(r,c,t),i&&a(i,c,t),n&&a(n,c,t),o(),s(),p.value=c}}function C(e){v.value||S(e)}function T(e){if(!v.value)switch(e.key){case` `:case`Enter`:S(e)}}function E(e){e.key===` `&&e.preventDefault()}let O={focus:()=>{i.value?.focus()},blur:()=>{i.value?.blur()}},k=D(`Checkbox`,u,s),A=x(()=>{let{value:e}=y,{common:{cubicBezierEaseInOut:t},self:{borderRadius:n,color:r,colorChecked:i,colorDisabled:a,colorTableHeader:o,colorTableHeaderModal:s,colorTableHeaderPopover:c,checkMarkColor:l,checkMarkColorDisabled:u,border:d,borderFocus:p,borderDisabled:m,borderChecked:h,boxShadowFocus:g,textColor:_,textColorDisabled:v,checkMarkColorDisabledChecked:x,colorDisabledChecked:S,borderDisabledChecked:C,labelPadding:w,labelLineHeight:T,labelFontWeight:E,[f(`fontSize`,e)]:D,[f(`size`,e)]:O}}=b.value;return{"--n-label-line-height":T,"--n-label-font-weight":E,"--n-size":O,"--n-bezier":t,"--n-border-radius":n,"--n-border":d,"--n-border-checked":h,"--n-border-focus":p,"--n-border-disabled":m,"--n-border-disabled-checked":C,"--n-box-shadow-focus":g,"--n-color":r,"--n-color-checked":i,"--n-color-table":o,"--n-color-table-modal":s,"--n-color-table-popover":c,"--n-color-disabled":a,"--n-color-disabled-checked":S,"--n-text-color":_,"--n-text-color-disabled":v,"--n-check-mark-color":l,"--n-check-mark-color-disabled":u,"--n-check-mark-color-disabled-checked":x,"--n-font-size":D,"--n-label-padding":w}}),j=l?t(`checkbox`,x(()=>y.value[0]),A,e):void 0;return Object.assign(_,O,{rtlEnabled:k,selfRef:i,mergedClsPrefix:s,mergedDisabled:v,renderedChecked:g,mergedTheme:b,labelId:q(),handleClick:C,handleKeyUp:T,handleKeyDown:E,cssVars:l?void 0:A,themeClass:j?.themeClass,onRender:j?.onRender})},render(){let{$slots:t,renderedChecked:n,mergedDisabled:r,indeterminate:a,privateInsideTable:o,cssVars:s,labelId:c,label:l,mergedClsPrefix:u,focusable:f,handleKeyUp:p,handleKeyDown:m,handleClick:h}=this;this.onRender?.();let g=ee(t.default,t=>l||t?(k(),e(`span`,{key:1,class:i(`${u}-checkbox__label`),id:c},[v(()=>l||t)],10,ce)):null);return(()=>{let t=B(`70be6e74cd27cb50`);return k(),e(`div`,{ref:`selfRef`,class:i([`${u}-checkbox`,this.themeClass,this.rtlEnabled&&`${u}-checkbox--rtl`,n&&`${u}-checkbox--checked`,r&&`${u}-checkbox--disabled`,a&&`${u}-checkbox--indeterminate`,o&&`${u}-checkbox--inside-table`,g&&`${u}-checkbox--show-label`]),tabindex:r||!f?void 0:0,role:`checkbox`,"aria-checked":a?`mixed`:n,"aria-labelledby":c,style:d(s),onKeyup:p,onKeydown:m,onClick:h,onMousedown:t[0]||=()=>{K(`selectstart`,window,e=>{e.preventDefault()},{once:!0})}},[y(`div`,{class:i(`${u}-checkbox-box-wrapper`)},[t[1]||=v(`\xA0`,-1),y(`div`,{class:i(`${u}-checkbox-box`)},[R(N,null,{default:()=>this.indeterminate?(k(),e(`div`,{key:`indeterminate`,class:i(`${u}-checkbox-icon`)},[v(()=>oe())],2)):(k(),e(`div`,{key:`check`,class:i(`${u}-checkbox-icon`)},[v(()=>ae())],2))},1024),y(`div`,{class:i(`${u}-checkbox-box__border`)},null,2)],2)],2),v(()=>g)],46,le)})()}}),Q=b(`n-checkbox-group`);P({name:`CheckboxGroup`,props:{min:Number,max:Number,size:String,options:Array,labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},value:Array,defaultValue:{type:Array,default:null},disabled:{type:Boolean,default:void 0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],onChange:[Function,Array]},setup(e){let{mergedClsPrefixRef:t}=F(e),r=z(e),{mergedSizeRef:i,mergedDisabledRef:s}=r,c=o(e.defaultValue),l=x(()=>e.value),u=Y(l,c),d=x(()=>u.value?.length||0),f=x(()=>Array.isArray(u.value)?new Set(u.value):new Set);function p(t,n){let{nTriggerFormInput:i,nTriggerFormChange:o}=r,{onChange:s,"onUpdate:value":l,onUpdateValue:d}=e;if(Array.isArray(u.value)){let e=Array.from(u.value),r=e.findIndex(e=>e===n);t?~r||(e.push(n),d&&a(d,e,{actionType:`check`,value:n}),l&&a(l,e,{actionType:`check`,value:n}),i(),o(),c.value=e,s&&a(s,e)):~r&&(e.splice(r,1),d&&a(d,e,{actionType:`uncheck`,value:n}),l&&a(l,e,{actionType:`uncheck`,value:n}),s&&a(s,e),c.value=e,i(),o())}else t?(d&&a(d,[n],{actionType:`check`,value:n}),l&&a(l,[n],{actionType:`check`,value:n}),s&&a(s,[n]),c.value=[n],i(),o()):(d&&a(d,[],{actionType:`uncheck`,value:n}),l&&a(l,[],{actionType:`uncheck`,value:n}),s&&a(s,[]),c.value=[],i(),o())}return O(Q,{checkedCountRef:d,maxRef:n(e,`max`),minRef:n(e,`min`),valueSetRef:f,disabledRef:s,mergedSizeRef:i,toggleCheckbox:p}),{mergedClsPrefix:t}},render(){let{options:t,labelField:n,valueField:r}=this.$props;return k(),e(`div`,{class:i(`${this.mergedClsPrefix}-checkbox-group`),role:`group`},[t?(k(),e(u,{key:0},[v(()=>t.map(e=>{let t=e[r];return k(),S(Z,{key:t,value:t,disabled:e.disabled,label:e[n]},null,8,[`value`,`disabled`,`label`])}))],64)):(k(),e(u,{key:1},[v(()=>this.$slots.default?.())],64))],2)}});var de={class:`dt-page`},fe=[`href`],pe={class:`dt-mono`},me={class:`list`},he={class:`top`},ge={class:`title`},_e={class:`name`},ve={class:`id dt-mono dt-faint`},ye={class:`desc dt-muted`},be={class:`facts`},$={class:`dt-mono`},xe={class:`dt-mono`},Se={class:`row`},Ce=[`href`,`download`],we={class:`form`},Te={class:`filerow`},Ee={class:`footer`},De=I(P({__name:`WorkflowsView`,setup(t){let n=ne(),r=o(``),i=o(!1),a=o(null),s=x(()=>J.value?.base_url??`http://127.0.0.1:8188`);async function c(){i.value=!0;try{await L.reloadWorkflows(),G.value=await L.workflows(),n.success(`已重新加载`)}catch(e){n.error(String(e.message))}finally{i.value=!1}}async function l(e){r.value=e;try{let t=await L.openInComfy(e);a.value={id:e,file:t.file},window.open(s.value,`_blank`,`noopener`)}catch(e){n.error(String(e.message))}finally{r.value=``}}let f=o(!1),p=o({id:``,name:``,style:`custom`,override:!1}),h=o(``),v=o(!1),b=o(null),w=[{label:`写实`,value:`realistic`},{label:`风格化`,value:`stylized`},{label:`自定义`,value:`custom`}];function T(e){let t=e.target.files?.[0];if(!t)return;let n=new FileReader;n.onload=()=>{h.value=String(n.result??``),p.value.id||(p.value.id=t.name.replace(/\.json$/i,``).toLowerCase())},n.readAsText(t)}async function D(){let e;try{e=JSON.parse(h.value)}catch{n.error(`内容不是合法 JSON`);return}v.value=!0;try{await L.importWorkflow({...p.value,graph:e}),G.value=await L.workflows(),n.success(`已导入 ${p.value.id}`),f.value=!1,h.value=``,p.value={id:``,name:``,style:`custom`,override:!1}}catch(e){n.error(String(e.message))}finally{v.value=!1}}return(t,n)=>(k(),e(`div`,de,[R(ie,{title:`工作流`,desc:`生成台上的每个风格预设就是这里的一套工作流。可以在 ComfyUI 里改完再导回来。`},{actions:_(()=>[y(`a`,{href:s.value,target:`_blank`,rel:`noreferrer`},[R(A(W),{size:`small`},{default:_(()=>[...n[10]||=[E(`打开 ComfyUI`,-1)]]),_:1})],8,fe),R(A(W),{size:`small`,loading:i.value,onClick:c},{default:_(()=>[...n[11]||=[E(`重新加载`,-1)]]),_:1},8,[`loading`]),R(A(W),{size:`small`,type:`primary`,onClick:n[0]||=e=>f.value=!0},{default:_(()=>[...n[12]||=[E(`导入`,-1)]]),_:1})]),_:1}),a.value?(k(),S(A(te),{key:0,type:`success`,bordered:!1,closable:``,class:`tip`,onClose:n[1]||=e=>a.value=null},{default:_(()=>[y(`b`,null,m(a.value.id),1),n[13]||=E(` 已放进 ComfyUI 的工作流列表。在 ComfyUI 左侧「工作流」面板里打开 `,-1),y(`span`,pe,m(a.value.file),1),n[14]||=E(`，改完用「工作流 → 导出(API)」导出， 再回这里点「导入」覆盖同名 id 即可生效。 `,-1)]),_:1})):g(``,!0),y(`div`,me,[(k(!0),e(u,null,C(A(G),(t,i)=>(k(),e(`article`,{key:t.id,class:`wf dt-panel dt-enter`,style:d({animationDelay:`${i*40}ms`})},[y(`div`,he,[y(`div`,ge,[y(`span`,_e,m(t.name),1),R(A(U),{size:`tiny`,bordered:!1},{default:_(()=>[E(m(t.style),1)]),_:2},1024),R(A(U),{size:`tiny`,bordered:!1},{default:_(()=>[E(m(t.resolution)+`²`,1)]),_:2},1024),t.tileable?(k(),S(A(U),{key:0,size:`tiny`,bordered:!1},{default:_(()=>[...n[15]||=[E(`无缝`,-1)]]),_:1})):g(``,!0),t.license_notice&&!t.license_notice.commercial?(k(),S(A(U),{key:1,size:`tiny`,type:`warning`,bordered:!1},{default:_(()=>[...n[16]||=[E(`不可商用`,-1)]]),_:1})):g(``,!0)]),y(`span`,ve,m(t.id)+` v`+m(t.version),1)]),y(`p`,ye,m(t.description),1),y(`dl`,be,[n[18]||=y(`dt`,null,`可调参数`,-1),y(`dd`,null,m(t.params.length)+` 基础 · `+m(t.advanced.length)+` 高级`,1),n[19]||=y(`dt`,null,`输出通道`,-1),y(`dd`,$,m(Object.keys(t.outputs).sort().join(` `)),1),t.node_packs?.length?(k(),e(u,{key:0},[n[17]||=y(`dt`,null,`节点包`,-1),y(`dd`,xe,m(t.node_packs.join(` · `)),1)],64)):g(``,!0)]),y(`div`,Se,[R(A(W),{size:`tiny`,loading:r.value===t.id,onClick:e=>l(t.id)},{default:_(()=>[...n[20]||=[E(` 在 ComfyUI 中编辑 `,-1)]]),_:1},8,[`loading`,`onClick`]),y(`a`,{href:`/api/workflows/${t.id}/template`,download:`${t.id}.json`},[R(A(W),{size:`tiny`,tertiary:``},{default:_(()=>[...n[21]||=[E(`下载模板`,-1)]]),_:1})],8,Ce)])],4))),128))]),R(A(X),{show:f.value,"onUpdate:show":n[9]||=e=>f.value=e,preset:`card`,title:`导入工作流`,style:{"max-width":`620px`}},{footer:_(()=>[y(`div`,Ee,[R(A(W),{size:`small`,onClick:n[8]||=e=>f.value=!1},{default:_(()=>[...n[27]||=[E(`取消`,-1)]]),_:1}),R(A(W),{size:`small`,type:`primary`,loading:v.value,disabled:!p.value.id||!h.value,onClick:D},{default:_(()=>[...n[28]||=[E(` 导入 `,-1)]]),_:1},8,[`loading`,`disabled`])])]),default:_(()=>[n[29]||=y(`p`,{class:`hint dt-muted`},[E(` 在 ComfyUI 里用`),y(`b`,null,`「工作流 → 导出(API)」`),E(`导出的 JSON。普通的保存/导出是 UI 格式， 缺少 DreamTexture 需要的结构，导入会被拒绝。 `)],-1),n[30]||=y(`p`,{class:`hint dt-faint`},[E(` 输出节点需命名为 `),y(`span`,{class:`dt-mono`},`dt.out.basecolor`),E(` 这样的标题， DreamTexture 靠它把产物对上通道；提示词节点命名为 `),y(`span`,{class:`dt-mono`},`dt.positive`),E(` 会自动生成对应的可调参数。 `)],-1),y(`div`,we,[y(`label`,null,[n[22]||=y(`span`,{class:`dt-label`},`id`,-1),R(A(V),{value:p.value.id,"onUpdate:value":n[2]||=e=>p.value.id=e,placeholder:`小写字母、数字、. _ -`,size:`small`},null,8,[`value`])]),y(`label`,null,[n[23]||=y(`span`,{class:`dt-label`},`名称`,-1),R(A(V),{value:p.value.name,"onUpdate:value":n[3]||=e=>p.value.name=e,placeholder:`留空则用 id`,size:`small`},null,8,[`value`])]),y(`label`,null,[n[24]||=y(`span`,{class:`dt-label`},`风格`,-1),R(A(H),{value:p.value.style,"onUpdate:value":n[4]||=e=>p.value.style=e,options:w,size:`small`},null,8,[`value`])])]),y(`div`,Te,[R(A(W),{size:`small`,onClick:n[5]||=e=>b.value?.click()},{default:_(()=>[...n[25]||=[E(`选择 JSON 文件`,-1)]]),_:1}),R(A(Z),{checked:p.value.override,"onUpdate:checked":n[6]||=e=>p.value.override=e,size:`small`},{default:_(()=>[...n[26]||=[E(`覆盖同名工作流`,-1)]]),_:1},8,[`checked`])]),y(`input`,{ref_key:`fileInput`,ref:b,type:`file`,accept:`application/json,.json`,hidden:``,onChange:T},null,544),R(A(V),{value:h.value,"onUpdate:value":n[7]||=e=>h.value=e,type:`textarea`,autosize:{minRows:6,maxRows:12},placeholder:`也可以直接把 API 格式 JSON 粘贴到这里`,class:`ta`},null,8,[`value`])]),_:1},8,[`show`])]))}}),[[`__scopeId`,`data-v-d72bfcf2`]]);export{De as default};