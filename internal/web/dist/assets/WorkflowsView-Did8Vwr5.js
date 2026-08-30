import{$t as e,A as t,At as n,B as r,C as i,Dt as a,E as o,Fn as s,Ft as c,Gt as l,Mn as u,Nt as d,On as f,Ot as p,Pn as m,Pt as h,Qt as g,Sn as _,U as v,Xt as y,Y as b,Yt as x,Zt as S,_n as C,an as w,c as T,en as E,g as D,gn as O,hn as k,jn as A,jt as j,k as M,l as N,nn as P,q as F,r as I,t as L,tn as R,u as z,x as B,z as V}from"./client-DNfQtFy2.js";import{n as H,t as U}from"./Select-8ZcNmrp7.js";import{t as W}from"./Tag-CSONWr3r.js";import{t as ee}from"./Alert-BwZ9t9Fh.js";import{t as te}from"./use-message-Di_nNlmh.js";import{A as ne,S as re,a as G,f as K,ft as q,h as J,it as Y,j as X,st as ie}from"./index-B-nu--i8.js";import{t as ae}from"./PageHeader-CI27ahcP.js";var oe=()=>(()=>{let e=V(`75be776d8875fa17`);return e[0]||=y(`svg`,{viewBox:`0 0 64 64`,class:`check-icon`},[y(`path`,{d:`M50.42,16.76L22.34,39.45l-8.1-11.46c-1.12-1.58-3.3-1.96-4.88-0.84c-1.58,1.12-1.95,3.3-0.84,4.88l10.26,14.51  c0.56,0.79,1.42,1.31,2.38,1.45c0.16,0.02,0.32,0.03,0.48,0.03c0.8,0,1.57-0.27,2.2-0.78l30.99-25.03c1.5-1.21,1.74-3.42,0.52-4.92  C54.13,15.78,51.93,15.55,50.42,16.76z`})],-1)})(),se=()=>(()=>{let e=V(`c6eed899356c8404`);return e[0]||=y(`svg`,{viewBox:`0 0 100 100`,class:`line-icon`},[y(`path`,{d:`M80.2,55.5H21.4c-2.8,0-5.1-2.5-5.1-5.5l0,0c0-3,2.3-5.5,5.1-5.5h58.7c2.8,0,5.1,2.5,5.1,5.5l0,0C85.2,53.1,82.9,55.5,80.2,55.5z`})],-1)})(),ce=a([p(`checkbox`,`
 font-size: var(--n-font-size);
 outline: none;
 cursor: pointer;
 display: inline-flex;
 flex-wrap: nowrap;
 align-items: flex-start;
 word-break: break-word;
 line-height: var(--n-size);
 --n-merged-color-table: var(--n-color-table);
 `,[j(`show-label`,`line-height: var(--n-label-line-height);`),a(`&:hover`,[p(`checkbox-box`,[n(`border`,`border: var(--n-border-checked);`)])]),a(`&:focus:not(:active)`,[p(`checkbox-box`,[n(`border`,`
 border: var(--n-border-focus);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),j(`inside-table`,[p(`checkbox-box`,`
 background-color: var(--n-merged-color-table);
 `)]),j(`checked`,[p(`checkbox-box`,`
 background-color: var(--n-color-checked);
 `,[p(`checkbox-icon`,[a(`.check-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),j(`indeterminate`,[p(`checkbox-box`,[p(`checkbox-icon`,[a(`.check-icon`,`
 opacity: 0;
 transform: scale(.5);
 `),a(`.line-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),j(`checked, indeterminate`,[a(`&:focus:not(:active)`,[p(`checkbox-box`,[n(`border`,`
 border: var(--n-border-checked);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),p(`checkbox-box`,`
 background-color: var(--n-color-checked);
 border-left: 0;
 border-top: 0;
 `,[n(`border`,{border:`var(--n-border-checked)`})])]),j(`disabled`,{cursor:`not-allowed`},[j(`checked`,[p(`checkbox-box`,`
 background-color: var(--n-color-disabled-checked);
 `,[n(`border`,{border:`var(--n-border-disabled-checked)`}),p(`checkbox-icon`,[a(`.check-icon, .line-icon`,{fill:`var(--n-check-mark-color-disabled-checked)`})])])]),p(`checkbox-box`,`
 background-color: var(--n-color-disabled);
 `,[n(`border`,`
 border: var(--n-border-disabled);
 `),p(`checkbox-icon`,[a(`.check-icon, .line-icon`,`
 fill: var(--n-check-mark-color-disabled);
 `)])]),n(`label`,`
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
 `,[n(`border`,`
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
 `,[a(`.check-icon, .line-icon`,`
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
 `),T({left:`1px`,top:`1px`})])]),n(`label`,`
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 user-select: none;
 -webkit-user-select: none;
 padding: var(--n-label-padding);
 font-weight: var(--n-label-font-weight);
 `,[a(`&:empty`,{display:`none`})])]),h(p(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-modal);
 `)),c(p(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-popover);
 `))]),le=[`id`],ue=[`tabindex`,`aria-checked`,`aria-labelledby`,`onKeyup`,`onKeydown`,`onClick`],de={...M.props,size:String,checked:{type:[Boolean,String,Number],default:void 0},defaultChecked:{type:[Boolean,String,Number],default:!1},value:[String,Number],disabled:{type:Boolean,default:void 0},indeterminate:Boolean,label:String,focusable:{type:Boolean,default:!0},checkedValue:{type:[Boolean,String,Number],default:!0},uncheckedValue:{type:[Boolean,String,Number],default:!1},"onUpdate:checked":[Function,Array],onUpdateChecked:[Function,Array],privateInsideTable:Boolean,onChange:[Function,Array]},Z=P({name:`Checkbox`,props:de,setup(e){let n=w(Q,null),r=f(null),{mergedClsPrefixRef:a,inlineThemeDisabled:s,mergedRtlRef:c,mergedComponentPropsRef:l}=F(e),u=f(e.defaultChecked),p=A(e,`checked`),m=Y(p,u),h=o(()=>{if(n){let t=n.valueSetRef.value;return t&&e.value!==void 0?t.has(e.value):!1}return m.value===e.checkedValue}),g=z(e,{mergedSize(t){let{size:r}=e;if(r!==void 0)return r;if(n){let{value:e}=n.mergedSizeRef;if(e!==void 0)return e}if(t){let{mergedSize:e}=t;if(e!==void 0)return e.value}return l?.value?.Checkbox?.size||`medium`},mergedDisabled(t){let{disabled:r}=e;if(r!==void 0)return r;if(n){if(n.disabledRef.value)return!0;let{maxRef:{value:e},checkedCountRef:t}=n;if(e!==void 0&&t.value>=e&&!h.value)return!0;let{minRef:{value:r}}=n;if(r!==void 0&&t.value<=r&&h.value)return!0}return t?t.disabled.value:!1}}),{mergedDisabledRef:_,mergedSizeRef:v}=g,y=M(`Checkbox`,`-checkbox`,ce,ne,e,a);function b(t){if(n&&e.value!==void 0)n.toggleCheckbox(!h.value,e.value);else{let{onChange:n,"onUpdate:checked":r,onUpdateChecked:a}=e,{nTriggerFormInput:o,nTriggerFormChange:s}=g,c=h.value?e.uncheckedValue:e.checkedValue;r&&i(r,c,t),a&&i(a,c,t),n&&i(n,c,t),o(),s(),u.value=c}}function S(e){_.value||b(e)}function C(e){if(!_.value)switch(e.key){case` `:case`Enter`:b(e)}}function T(e){e.key===` `&&e.preventDefault()}let E={focus:()=>{r.value?.focus()},blur:()=>{r.value?.blur()}},O=D(`Checkbox`,c,a),k=x(()=>{let{value:e}=v,{common:{cubicBezierEaseInOut:t},self:{borderRadius:n,color:r,colorChecked:i,colorDisabled:a,colorTableHeader:o,colorTableHeaderModal:s,colorTableHeaderPopover:c,checkMarkColor:l,checkMarkColorDisabled:u,border:f,borderFocus:p,borderDisabled:m,borderChecked:h,boxShadowFocus:g,textColor:_,textColorDisabled:b,checkMarkColorDisabledChecked:x,colorDisabledChecked:S,borderDisabledChecked:C,labelPadding:w,labelLineHeight:T,labelFontWeight:E,[d(`fontSize`,e)]:D,[d(`size`,e)]:O}}=y.value;return{"--n-label-line-height":T,"--n-label-font-weight":E,"--n-size":O,"--n-bezier":t,"--n-border-radius":n,"--n-border":f,"--n-border-checked":h,"--n-border-focus":p,"--n-border-disabled":m,"--n-border-disabled-checked":C,"--n-box-shadow-focus":g,"--n-color":r,"--n-color-checked":i,"--n-color-table":o,"--n-color-table-modal":s,"--n-color-table-popover":c,"--n-color-disabled":a,"--n-color-disabled-checked":S,"--n-text-color":_,"--n-text-color-disabled":b,"--n-check-mark-color":l,"--n-check-mark-color-disabled":u,"--n-check-mark-color-disabled-checked":x,"--n-font-size":D,"--n-label-padding":w}}),j=s?t(`checkbox`,x(()=>v.value[0]),k,e):void 0;return Object.assign(g,E,{rtlEnabled:O,selfRef:r,mergedClsPrefix:a,mergedDisabled:_,renderedChecked:h,mergedTheme:y,labelId:q(),handleClick:S,handleKeyUp:C,handleKeyDown:T,cssVars:s?void 0:k,themeClass:j?.themeClass,onRender:j?.onRender})},render(){let{$slots:t,renderedChecked:n,mergedDisabled:i,indeterminate:a,privateInsideTable:o,cssVars:s,labelId:c,label:l,mergedClsPrefix:u,focusable:d,handleKeyUp:f,handleKeyDown:p,handleClick:h}=this;this.onRender?.();let g=B(t.default,t=>l||t?(k(),e(`span`,{key:1,class:r(`${u}-checkbox__label`),id:c},[v(()=>l||t)],10,le)):null);return(()=>{let t=V(`70be6e74cd27cb50`);return k(),e(`div`,{ref:`selfRef`,class:r([`${u}-checkbox`,this.themeClass,this.rtlEnabled&&`${u}-checkbox--rtl`,n&&`${u}-checkbox--checked`,i&&`${u}-checkbox--disabled`,a&&`${u}-checkbox--indeterminate`,o&&`${u}-checkbox--inside-table`,g&&`${u}-checkbox--show-label`]),tabindex:i||!d?void 0:0,role:`checkbox`,"aria-checked":a?`mixed`:n,"aria-labelledby":c,style:m(s),onKeyup:f,onKeydown:p,onClick:h,onMousedown:t[0]||=()=>{ie(`selectstart`,window,e=>{e.preventDefault()},{once:!0})}},[y(`div`,{class:r(`${u}-checkbox-box-wrapper`)},[t[1]||=v(`\xA0`,-1),y(`div`,{class:r(`${u}-checkbox-box`)},[R(N,null,{default:()=>this.indeterminate?(k(),e(`div`,{key:`indeterminate`,class:r(`${u}-checkbox-icon`)},[v(()=>se())],2)):(k(),e(`div`,{key:`check`,class:r(`${u}-checkbox-icon`)},[v(()=>oe())],2))},1024),y(`div`,{class:r(`${u}-checkbox-box__border`)},null,2)],2)],2),v(()=>g)],46,ue)})()}}),Q=b(`n-checkbox-group`);P({name:`CheckboxGroup`,props:{min:Number,max:Number,size:String,options:Array,labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},value:Array,defaultValue:{type:Array,default:null},disabled:{type:Boolean,default:void 0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],onChange:[Function,Array]},setup(e){let{mergedClsPrefixRef:t}=F(e),n=z(e),{mergedSizeRef:r,mergedDisabledRef:a}=n,o=f(e.defaultValue),s=x(()=>e.value),c=Y(s,o),l=x(()=>c.value?.length||0),u=x(()=>Array.isArray(c.value)?new Set(c.value):new Set);function d(t,r){let{nTriggerFormInput:a,nTriggerFormChange:s}=n,{onChange:l,"onUpdate:value":u,onUpdateValue:d}=e;if(Array.isArray(c.value)){let e=Array.from(c.value),n=e.findIndex(e=>e===r);t?~n||(e.push(r),d&&i(d,e,{actionType:`check`,value:r}),u&&i(u,e,{actionType:`check`,value:r}),a(),s(),o.value=e,l&&i(l,e)):~n&&(e.splice(n,1),d&&i(d,e,{actionType:`uncheck`,value:r}),u&&i(u,e,{actionType:`uncheck`,value:r}),l&&i(l,e),o.value=e,a(),s())}else t?(d&&i(d,[r],{actionType:`check`,value:r}),u&&i(u,[r],{actionType:`check`,value:r}),l&&i(l,[r]),o.value=[r],a(),s()):(d&&i(d,[],{actionType:`uncheck`,value:r}),u&&i(u,[],{actionType:`uncheck`,value:r}),l&&i(l,[]),o.value=[],a(),s())}return O(Q,{checkedCountRef:l,maxRef:A(e,`max`),minRef:A(e,`min`),valueSetRef:u,disabledRef:a,mergedSizeRef:r,toggleCheckbox:d}),{mergedClsPrefix:t}},render(){let{options:t,labelField:n,valueField:i}=this.$props;return k(),e(`div`,{class:r(`${this.mergedClsPrefix}-checkbox-group`),role:`group`},[t?(k(),e(l,{key:0},[v(()=>t.map(e=>{let t=e[i];return k(),S(Z,{key:t,value:t,disabled:e.disabled,label:e[n]},null,8,[`value`,`disabled`,`label`])}))],64)):(k(),e(l,{key:1},[v(()=>this.$slots.default?.())],64))],2)}});var fe={class:`dt-page`},pe=[`href`],me={class:`dt-mono`},he={class:`list`},ge={class:`top`},_e={class:`title`},ve={class:`name`},ye={class:`id dt-mono dt-faint`},be={class:`desc dt-muted`},$={class:`facts`},xe={class:`dt-mono`},Se={class:`dt-mono`},Ce={key:0,class:`nograph dt-faint`},we={class:`dt-mono`},Te={key:1,class:`row`},Ee=[`href`,`download`],De={class:`form`},Oe={class:`filerow`},ke={class:`footer`},Ae=I(P({__name:`WorkflowsView`,setup(t){let n=te(),r=f(``),i=f(!1),a=f(null),o=x(()=>G.value?.base_url??`http://127.0.0.1:8188`),c=e=>!!e.source?.direct_output;async function d(){i.value=!0;try{await L.reloadWorkflows();let e=await L.workflows();J.value=e.workflows,K.value=e.segments,n.success(`已重新加载`)}catch(e){n.error(String(e.message))}finally{i.value=!1}}async function p(e){r.value=e;try{let t=await L.openInComfy(e);a.value={id:e,file:t.file},window.open(o.value,`_blank`,`noopener`)}catch(e){n.error(String(e.message))}finally{r.value=``}}let h=f(!1),v=f({id:``,name:``,style:`custom`,override:!1}),b=f(``),w=f(!1),T=f(null),D=[{label:`写实`,value:`realistic`},{label:`风格化`,value:`stylized`},{label:`自定义`,value:`custom`}];function O(e){let t=e.target.files?.[0];if(!t)return;let n=new FileReader;n.onload=()=>{b.value=String(n.result??``),v.value.id||(v.value.id=t.name.replace(/\.json$/i,``).toLowerCase())},n.readAsText(t)}async function A(){let e;try{e=JSON.parse(b.value)}catch{n.error(`内容不是合法 JSON`);return}w.value=!0;try{await L.importWorkflow({...v.value,graph:e});let t=await L.workflows();J.value=t.workflows,K.value=t.segments,n.success(`已导入 ${v.value.id}`),h.value=!1,b.value=``,v.value={id:``,name:``,style:`custom`,override:!1}}catch(e){n.error(String(e.message))}finally{w.value=!1}}return(t,n)=>(k(),e(`div`,fe,[R(ae,{title:`工作流`,desc:`生成台上的每个风格预设就是这里的一套工作流。可以在 ComfyUI 里改完再导回来。`},{actions:_(()=>[y(`a`,{href:o.value,target:`_blank`,rel:`noreferrer`},[R(u(X),{size:`small`},{default:_(()=>[...n[10]||=[E(`打开 ComfyUI`,-1)]]),_:1})],8,pe),R(u(X),{size:`small`,loading:i.value,onClick:d},{default:_(()=>[...n[11]||=[E(`重新加载`,-1)]]),_:1},8,[`loading`]),R(u(X),{size:`small`,type:`primary`,onClick:n[0]||=e=>h.value=!0},{default:_(()=>[...n[12]||=[E(`导入`,-1)]]),_:1})]),_:1}),a.value?(k(),S(u(ee),{key:0,type:`success`,bordered:!1,closable:``,class:`tip`,onClose:n[1]||=e=>a.value=null},{default:_(()=>[y(`b`,null,s(a.value.id),1),n[13]||=E(` 已放进 ComfyUI 的工作流列表。在 ComfyUI 左侧「工作流」面板里打开 `,-1),y(`span`,me,s(a.value.file),1),n[14]||=E(`，改完用「工作流 → 导出(API)」导出， 再回这里点「导入」覆盖同名 id 即可生效。 `,-1)]),_:1})):g(``,!0),y(`div`,he,[(k(!0),e(l,null,C(u(J),(t,i)=>(k(),e(`article`,{key:t.id,class:`wf dt-panel dt-enter`,style:m({animationDelay:`${i*40}ms`})},[y(`div`,ge,[y(`div`,_e,[y(`span`,ve,s(t.name),1),R(u(W),{size:`tiny`,bordered:!1},{default:_(()=>[E(s(t.style),1)]),_:2},1024),R(u(W),{size:`tiny`,bordered:!1},{default:_(()=>[E(s(t.resolution)+`²`,1)]),_:2},1024),t.tileable?(k(),S(u(W),{key:0,size:`tiny`,bordered:!1},{default:_(()=>[...n[15]||=[E(`无缝`,-1)]]),_:1})):g(``,!0),t.license_notice&&!t.license_notice.commercial?(k(),S(u(W),{key:1,size:`tiny`,type:`warning`,bordered:!1},{default:_(()=>[...n[16]||=[E(`不可商用`,-1)]]),_:1})):g(``,!0)]),y(`span`,ye,s(t.id)+` v`+s(t.version),1)]),y(`p`,be,s(t.description),1),y(`dl`,$,[n[19]||=y(`dt`,null,`可调参数`,-1),y(`dd`,null,s(t.params.length)+` 基础 · `+s(t.advanced.length)+` 高级`,1),c(t)?g(``,!0):(k(),e(l,{key:0},[n[17]||=y(`dt`,null,`输出通道`,-1),y(`dd`,xe,s(Object.keys(t.outputs).sort().join(` `)),1)],64)),t.node_packs?.length?(k(),e(l,{key:1},[n[18]||=y(`dt`,null,`节点包`,-1),y(`dd`,Se,s(t.node_packs.join(` · `)),1)],64)):g(``,!0)]),c(t)?(k(),e(`p`,Ce,[n[20]||=E(` 纯云端直出，本机没有节点图可编辑——参数直接发给 `,-1),y(`span`,we,s(t.source?.provider),1),n[21]||=E(`，产物拿回来就是成品。 `,-1)])):(k(),e(`div`,Te,[R(u(X),{size:`tiny`,loading:r.value===t.id,onClick:e=>p(t.id)},{default:_(()=>[...n[22]||=[E(` 在 ComfyUI 中编辑 `,-1)]]),_:1},8,[`loading`,`onClick`]),y(`a`,{href:`/api/workflows/${t.id}/template`,download:`${t.id}.json`},[R(u(X),{size:`tiny`,tertiary:``},{default:_(()=>[...n[23]||=[E(`下载模板`,-1)]]),_:1})],8,Ee)]))],4))),128))]),R(u(re),{show:h.value,"onUpdate:show":n[9]||=e=>h.value=e,preset:`card`,title:`导入工作流`,style:{"max-width":`620px`}},{footer:_(()=>[y(`div`,ke,[R(u(X),{size:`small`,onClick:n[8]||=e=>h.value=!1},{default:_(()=>[...n[29]||=[E(`取消`,-1)]]),_:1}),R(u(X),{size:`small`,type:`primary`,loading:w.value,disabled:!v.value.id||!b.value,onClick:A},{default:_(()=>[...n[30]||=[E(` 导入 `,-1)]]),_:1},8,[`loading`,`disabled`])])]),default:_(()=>[n[31]||=y(`p`,{class:`hint dt-muted`},[E(` 在 ComfyUI 里用`),y(`b`,null,`「工作流 → 导出(API)」`),E(`导出的 JSON。普通的保存/导出是 UI 格式， 缺少 DreamTexture 需要的结构，导入会被拒绝。 `)],-1),n[32]||=y(`p`,{class:`hint dt-faint`},[E(` 输出节点需命名为 `),y(`span`,{class:`dt-mono`},`dt.out.basecolor`),E(` 这样的标题， DreamTexture 靠它把产物对上通道；提示词节点命名为 `),y(`span`,{class:`dt-mono`},`dt.positive`),E(` 会自动生成对应的可调参数。 `)],-1),y(`div`,De,[y(`label`,null,[n[24]||=y(`span`,{class:`dt-label`},`id`,-1),R(u(H),{value:v.value.id,"onUpdate:value":n[2]||=e=>v.value.id=e,placeholder:`小写字母、数字、. _ -`,size:`small`},null,8,[`value`])]),y(`label`,null,[n[25]||=y(`span`,{class:`dt-label`},`名称`,-1),R(u(H),{value:v.value.name,"onUpdate:value":n[3]||=e=>v.value.name=e,placeholder:`留空则用 id`,size:`small`},null,8,[`value`])]),y(`label`,null,[n[26]||=y(`span`,{class:`dt-label`},`风格`,-1),R(u(U),{value:v.value.style,"onUpdate:value":n[4]||=e=>v.value.style=e,options:D,size:`small`},null,8,[`value`])])]),y(`div`,Oe,[R(u(X),{size:`small`,onClick:n[5]||=e=>T.value?.click()},{default:_(()=>[...n[27]||=[E(`选择 JSON 文件`,-1)]]),_:1}),R(u(Z),{checked:v.value.override,"onUpdate:checked":n[6]||=e=>v.value.override=e,size:`small`},{default:_(()=>[...n[28]||=[E(`覆盖同名工作流`,-1)]]),_:1},8,[`checked`])]),y(`input`,{ref_key:`fileInput`,ref:T,type:`file`,accept:`application/json,.json`,hidden:``,onChange:O},null,544),R(u(H),{value:b.value,"onUpdate:value":n[7]||=e=>b.value=e,type:`textarea`,autosize:{minRows:6,maxRows:12},placeholder:`也可以直接把 API 格式 JSON 粘贴到这里`,class:`ta`},null,8,[`value`])]),_:1},8,[`show`])]))}}),[[`__scopeId`,`data-v-28a3493c`]]);export{Ae as default};