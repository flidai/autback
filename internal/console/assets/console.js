var W1=globalThis,Y1=W1.ShadowRoot&&(W1.ShadyCSS===void 0||W1.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,I1=Symbol(),g0=new WeakMap;class n1{constructor(x,d,h){if(this._$cssResult$=!0,h!==I1)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=x,this.t=d}get styleSheet(){let x=this.o,d=this.t;if(Y1&&x===void 0){let h=d!==void 0&&d.length===1;h&&(x=g0.get(d)),x===void 0&&((this.o=x=new CSSStyleSheet).replaceSync(this.cssText),h&&g0.set(d,x))}return x}toString(){return this.cssText}}var X0=(x)=>new n1(typeof x=="string"?x:x+"",void 0,I1),d1=(x,...d)=>{let h=x.length===1?x[0]:d.reduce((i,p,r)=>i+((v)=>{if(v._$cssResult$===!0)return v.cssText;if(typeof v=="number")return v;throw Error("Value passed to 'css' function must be a 'css' function result: "+v+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(p)+x[r+1],x[0]);return new n1(h,x,I1)},Z0=(x,d)=>{if(Y1)x.adoptedStyleSheets=d.map((h)=>h instanceof CSSStyleSheet?h:h.styleSheet);else for(let h of d){let i=document.createElement("style"),p=W1.litNonce;p!==void 0&&i.setAttribute("nonce",p),i.textContent=h.cssText,x.appendChild(i)}},C1=Y1?(x)=>x:(x)=>x instanceof CSSStyleSheet?((d)=>{let h="";for(let i of d.cssRules)h+=i.cssText;return X0(h)})(x):x;var{is:cd,defineProperty:Gd,getOwnPropertyDescriptor:Dd,getOwnPropertyNames:Hd,getOwnPropertySymbols:Od,getPrototypeOf:fd}=Object,E1=globalThis,K0=E1.trustedTypes,sd=K0?K0.emptyScript:"",Pd=E1.reactiveElementPolyfillSupport,h1=(x,d)=>x,R1={toAttribute(x,d){switch(d){case Boolean:x=x?sd:null;break;case Object:case Array:x=x==null?x:JSON.stringify(x)}return x},fromAttribute(x,d){let h=x;switch(d){case Boolean:h=x!==null;break;case Number:h=x===null?null:Number(x);break;case Object:case Array:try{h=JSON.parse(x)}catch(i){h=null}}return h}},U0=(x,d)=>!cd(x,d),B0={attribute:!0,type:String,converter:R1,reflect:!1,useDefault:!1,hasChanged:U0};Symbol.metadata??=Symbol("metadata"),E1.litPropertyMetadata??=new WeakMap;class C extends HTMLElement{static addInitializer(x){this._$Ei(),(this.l??=[]).push(x)}static get observedAttributes(){return this.finalize(),this._$Eh&&[...this._$Eh.keys()]}static createProperty(x,d=B0){if(d.state&&(d.attribute=!1),this._$Ei(),this.prototype.hasOwnProperty(x)&&((d=Object.create(d)).wrapped=!0),this.elementProperties.set(x,d),!d.noAccessor){let h=Symbol(),i=this.getPropertyDescriptor(x,h,d);i!==void 0&&Gd(this.prototype,x,i)}}static getPropertyDescriptor(x,d,h){let{get:i,set:p}=Dd(this.prototype,x)??{get(){return this[d]},set(r){this[d]=r}};return{get:i,set(r){let v=i?.call(this);p?.call(this,r),this.requestUpdate(x,v,h)},configurable:!0,enumerable:!0}}static getPropertyOptions(x){return this.elementProperties.get(x)??B0}static _$Ei(){if(this.hasOwnProperty(h1("elementProperties")))return;let x=fd(this);x.finalize(),x.l!==void 0&&(this.l=[...x.l]),this.elementProperties=new Map(x.elementProperties)}static finalize(){if(this.hasOwnProperty(h1("finalized")))return;if(this.finalized=!0,this._$Ei(),this.hasOwnProperty(h1("properties"))){let d=this.properties,h=[...Hd(d),...Od(d)];for(let i of h)this.createProperty(i,d[i])}let x=this[Symbol.metadata];if(x!==null){let d=litPropertyMetadata.get(x);if(d!==void 0)for(let[h,i]of d)this.elementProperties.set(h,i)}this._$Eh=new Map;for(let[d,h]of this.elementProperties){let i=this._$Eu(d,h);i!==void 0&&this._$Eh.set(i,d)}this.elementStyles=this.finalizeStyles(this.styles)}static finalizeStyles(x){let d=[];if(Array.isArray(x)){let h=new Set(x.flat(1/0).reverse());for(let i of h)d.unshift(C1(i))}else x!==void 0&&d.push(C1(x));return d}static _$Eu(x,d){let h=d.attribute;return h===!1?void 0:typeof h=="string"?h:typeof x=="string"?x.toLowerCase():void 0}constructor(){super(),this._$Ep=void 0,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Em=null,this._$Ev()}_$Ev(){this._$ES=new Promise((x)=>this.enableUpdating=x),this._$AL=new Map,this._$E_(),this.requestUpdate(),this.constructor.l?.forEach((x)=>x(this))}addController(x){(this._$EO??=new Set).add(x),this.renderRoot!==void 0&&this.isConnected&&x.hostConnected?.()}removeController(x){this._$EO?.delete(x)}_$E_(){let x=new Map,d=this.constructor.elementProperties;for(let h of d.keys())this.hasOwnProperty(h)&&(x.set(h,this[h]),delete this[h]);x.size>0&&(this._$Ep=x)}createRenderRoot(){let x=this.shadowRoot??this.attachShadow(this.constructor.shadowRootOptions);return Z0(x,this.constructor.elementStyles),x}connectedCallback(){this.renderRoot??=this.createRenderRoot(),this.enableUpdating(!0),this._$EO?.forEach((x)=>x.hostConnected?.())}enableUpdating(x){}disconnectedCallback(){this._$EO?.forEach((x)=>x.hostDisconnected?.())}attributeChangedCallback(x,d,h){this._$AK(x,h)}_$ET(x,d){let h=this.constructor.elementProperties.get(x),i=this.constructor._$Eu(x,h);if(i!==void 0&&h.reflect===!0){let p=(h.converter?.toAttribute!==void 0?h.converter:R1).toAttribute(d,h.type);this._$Em=x,p==null?this.removeAttribute(i):this.setAttribute(i,p),this._$Em=null}}_$AK(x,d){let h=this.constructor,i=h._$Eh.get(x);if(i!==void 0&&this._$Em!==i){let p=h.getPropertyOptions(i),r=typeof p.converter=="function"?{fromAttribute:p.converter}:p.converter?.fromAttribute!==void 0?p.converter:R1;this._$Em=i;let v=r.fromAttribute(d,p.type);this[i]=v??this._$Ej?.get(i)??v,this._$Em=null}}requestUpdate(x,d,h,i=!1,p){if(x!==void 0){let r=this.constructor;if(i===!1&&(p=this[x]),h??=r.getPropertyOptions(x),!((h.hasChanged??U0)(p,d)||h.useDefault&&h.reflect&&p===this._$Ej?.get(x)&&!this.hasAttribute(r._$Eu(x,h))))return;this.C(x,d,h)}this.isUpdatePending===!1&&(this._$ES=this._$EP())}C(x,d,{useDefault:h,reflect:i,wrapped:p},r){h&&!(this._$Ej??=new Map).has(x)&&(this._$Ej.set(x,r??d??this[x]),p!==!0||r!==void 0)||(this._$AL.has(x)||(this.hasUpdated||h||(d=void 0),this._$AL.set(x,d)),i===!0&&this._$Em!==x&&(this._$Eq??=new Set).add(x))}async _$EP(){this.isUpdatePending=!0;try{await this._$ES}catch(d){Promise.reject(d)}let x=this.scheduleUpdate();return x!=null&&await x,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){if(!this.isUpdatePending)return;if(!this.hasUpdated){if(this.renderRoot??=this.createRenderRoot(),this._$Ep){for(let[i,p]of this._$Ep)this[i]=p;this._$Ep=void 0}let h=this.constructor.elementProperties;if(h.size>0)for(let[i,p]of h){let{wrapped:r}=p,v=this[i];r!==!0||this._$AL.has(i)||v===void 0||this.C(i,void 0,p,v)}}let x=!1,d=this._$AL;try{x=this.shouldUpdate(d),x?(this.willUpdate(d),this._$EO?.forEach((h)=>h.hostUpdate?.()),this.update(d)):this._$EM()}catch(h){throw x=!1,this._$EM(),h}x&&this._$AE(d)}willUpdate(x){}_$AE(x){this._$EO?.forEach((d)=>d.hostUpdated?.()),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(x)),this.updated(x)}_$EM(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$ES}shouldUpdate(x){return!0}update(x){this._$Eq&&=this._$Eq.forEach((d)=>this._$ET(d,this[d])),this._$EM()}updated(x){}firstUpdated(x){}}C.elementStyles=[],C.shadowRootOptions={mode:"open"},C[h1("elementProperties")]=new Map,C[h1("finalized")]=new Map,Pd?.({ReactiveElement:C}),(E1.reactiveElementVersions??=[]).push("2.1.2");var w1=globalThis,M0=(x)=>x,J1=w1.trustedTypes,L0=J1?J1.createPolicy("lit-html",{createHTML:(x)=>x}):void 0;var R=`lit$${Math.random().toFixed(9).slice(2)}$`,H0="?"+R,Vd=`<${H0}>`,u=document,p1=()=>u.createComment(""),r1=(x)=>x===null||typeof x!="object"&&typeof x!="function",F1=Array.isArray,Sd=(x)=>F1(x)||typeof x?.[Symbol.iterator]=="function";var i1=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,j0=/-->/g,A0=/>/g,m=RegExp(`>|[ 	
\f\r](?:([^\\s"'>=/]+)([ 	
\f\r]*=[ 	
\f\r]*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`,"g"),c0=/'/g,G0=/"/g,O0=/^(?:script|style|textarea|title)$/i,$1=(x)=>(d,...h)=>({_$litType$:x,strings:d,values:h}),J=$1(1),U=$1(2),Ah=$1(3),o=Symbol.for("lit-noChange"),Z=Symbol.for("lit-nothing"),D0=new WeakMap,_=u.createTreeWalker(u,129);function f0(x,d){if(!F1(x)||!x.hasOwnProperty("raw"))throw Error("invalid template strings array");return L0!==void 0?L0.createHTML(d):d}var Id=(x,d)=>{let h=x.length-1,i=[],p,r=d===2?"<svg>":d===3?"<math>":"",v=i1;for(let k=0;k<h;k++){let N=x[k],q,y,z=-1,W=0;for(;W<N.length&&(v.lastIndex=W,y=v.exec(N),y!==null);)W=v.lastIndex,v===i1?y[1]==="!--"?v=j0:y[1]!==void 0?v=A0:y[2]!==void 0?(O0.test(y[2])&&(p=RegExp("</"+y[2],"g")),v=m):y[3]!==void 0&&(v=m):v===m?y[0]===">"?(v=p??i1,z=-1):y[1]===void 0?z=-2:(z=v.lastIndex-y[2].length,q=y[1],v=y[3]===void 0?m:y[3]==='"'?G0:c0):v===G0||v===c0?v=m:v===j0||v===A0?v=i1:(v=m,p=void 0);let Q=v===m&&x[k+1].startsWith("/>")?" ":"";r+=v===i1?N+Vd:z>=0?(i.push(q),N.slice(0,z)+"$lit$"+N.slice(z)+R+Q):N+R+(z===-2?k:Q)}return[f0(x,r+(x[h]||"<?>")+(d===2?"</svg>":d===3?"</math>":"")),i]};class v1{constructor({strings:x,_$litType$:d},h){let i;this.parts=[];let p=0,r=0,v=x.length-1,k=this.parts,[N,q]=Id(x,d);if(this.el=v1.createElement(N,h),_.currentNode=this.el.content,d===2||d===3){let y=this.el.content.firstChild;y.replaceWith(...y.childNodes)}for(;(i=_.nextNode())!==null&&k.length<v;){if(i.nodeType===1){if(i.hasAttributes())for(let y of i.getAttributeNames())if(y.endsWith("$lit$")){let z=q[r++],W=i.getAttribute(y).split(R),Q=/([.?@])?(.*)/.exec(z);k.push({type:1,index:p,name:Q[2],strings:W,ctor:Q[1]==="."?P0:Q[1]==="?"?V0:Q[1]==="@"?S0:k1}),i.removeAttribute(y)}else y.startsWith(R)&&(k.push({type:6,index:p}),i.removeAttribute(y));if(O0.test(i.tagName)){let y=i.textContent.split(R),z=y.length-1;if(z>0){i.textContent=J1?J1.emptyScript:"";for(let W=0;W<z;W++)i.append(y[W],p1()),_.nextNode(),k.push({type:2,index:++p});i.append(y[z],p1())}}}else if(i.nodeType===8)if(i.data===H0)k.push({type:2,index:p});else{let y=-1;for(;(y=i.data.indexOf(R,y+1))!==-1;)k.push({type:7,index:p}),y+=R.length-1}p++}}static createElement(x,d){let h=u.createElement("template");return h.innerHTML=x,h}}function a(x,d,h=x,i){if(d===o)return d;let p=i!==void 0?h._$Co?.[i]:h._$Cl,r=r1(d)?void 0:d._$litDirective$;return p?.constructor!==r&&(p?._$AO?.(!1),r===void 0?p=void 0:(p=new r(x),p._$AT(x,h,i)),i!==void 0?(h._$Co??=[])[i]=p:h._$Cl=p),p!==void 0&&(d=a(x,p._$AS(x,d.values),p,i)),d}class s0{constructor(x,d){this._$AV=[],this._$AN=void 0,this._$AD=x,this._$AM=d}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}u(x){let{el:{content:d},parts:h}=this._$AD,i=(x?.creationScope??u).importNode(d,!0);_.currentNode=i;let p=_.nextNode(),r=0,v=0,k=h[0];for(;k!==void 0;){if(r===k.index){let N;k.type===2?N=new y1(p,p.nextSibling,this,x):k.type===1?N=new k.ctor(p,k.name,k.strings,this,x):k.type===6&&(N=new I0(p,this,x)),this._$AV.push(N),k=h[++v]}r!==k?.index&&(p=_.nextNode(),r++)}return _.currentNode=u,i}p(x){let d=0;for(let h of this._$AV)h!==void 0&&(h.strings!==void 0?(h._$AI(x,h,d),d+=h.strings.length-2):h._$AI(x[d])),d++}}class y1{get _$AU(){return this._$AM?._$AU??this._$Cv}constructor(x,d,h,i){this.type=2,this._$AH=Z,this._$AN=void 0,this._$AA=x,this._$AB=d,this._$AM=h,this.options=i,this._$Cv=i?.isConnected??!0}get parentNode(){let x=this._$AA.parentNode,d=this._$AM;return d!==void 0&&x?.nodeType===11&&(x=d.parentNode),x}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(x,d=this){x=a(this,x,d),r1(x)?x===Z||x==null||x===""?(this._$AH!==Z&&this._$AR(),this._$AH=Z):x!==this._$AH&&x!==o&&this._(x):x._$litType$!==void 0?this.$(x):x.nodeType!==void 0?this.T(x):Sd(x)?this.k(x):this._(x)}O(x){return this._$AA.parentNode.insertBefore(x,this._$AB)}T(x){this._$AH!==x&&(this._$AR(),this._$AH=this.O(x))}_(x){this._$AH!==Z&&r1(this._$AH)?this._$AA.nextSibling.data=x:this.T(u.createTextNode(x)),this._$AH=x}$(x){let{values:d,_$litType$:h}=x,i=typeof h=="number"?this._$AC(x):(h.el===void 0&&(h.el=v1.createElement(f0(h.h,h.h[0]),this.options)),h);if(this._$AH?._$AD===i)this._$AH.p(d);else{let p=new s0(i,this),r=p.u(this.options);p.p(d),this.T(r),this._$AH=p}}_$AC(x){let d=D0.get(x.strings);return d===void 0&&D0.set(x.strings,d=new v1(x)),d}k(x){F1(this._$AH)||(this._$AH=[],this._$AR());let d=this._$AH,h,i=0;for(let p of x)i===d.length?d.push(h=new y1(this.O(p1()),this.O(p1()),this,this.options)):h=d[i],h._$AI(p),i++;i<d.length&&(this._$AR(h&&h._$AB.nextSibling,i),d.length=i)}_$AR(x=this._$AA.nextSibling,d){for(this._$AP?.(!1,!0,d);x!==this._$AB;){let h=M0(x).nextSibling;M0(x).remove(),x=h}}setConnected(x){this._$AM===void 0&&(this._$Cv=x,this._$AP?.(x))}}class k1{get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}constructor(x,d,h,i,p){this.type=1,this._$AH=Z,this._$AN=void 0,this.element=x,this.name=d,this._$AM=i,this.options=p,h.length>2||h[0]!==""||h[1]!==""?(this._$AH=Array(h.length-1).fill(new String),this.strings=h):this._$AH=Z}_$AI(x,d=this,h,i){let p=this.strings,r=!1;if(p===void 0)x=a(this,x,d,0),r=!r1(x)||x!==this._$AH&&x!==o,r&&(this._$AH=x);else{let v=x,k,N;for(x=p[0],k=0;k<p.length-1;k++)N=a(this,v[h+k],d,k),N===o&&(N=this._$AH[k]),r||=!r1(N)||N!==this._$AH[k],N===Z?x=Z:x!==Z&&(x+=(N??"")+p[k+1]),this._$AH[k]=N}r&&!i&&this.j(x)}j(x){x===Z?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,x??"")}}class P0 extends k1{constructor(){super(...arguments),this.type=3}j(x){this.element[this.name]=x===Z?void 0:x}}class V0 extends k1{constructor(){super(...arguments),this.type=4}j(x){this.element.toggleAttribute(this.name,!!x&&x!==Z)}}class S0 extends k1{constructor(x,d,h,i,p){super(x,d,h,i,p),this.type=5}_$AI(x,d=this){if((x=a(this,x,d,0)??Z)===o)return;let h=this._$AH,i=x===Z&&h!==Z||x.capture!==h.capture||x.once!==h.once||x.passive!==h.passive,p=x!==Z&&(h===Z||i);i&&this.element.removeEventListener(this.name,this,h),p&&this.element.addEventListener(this.name,this,x),this._$AH=x}handleEvent(x){typeof this._$AH=="function"?this._$AH.call(this.options?.host??this.element,x):this._$AH.handleEvent(x)}}class I0{constructor(x,d,h){this.element=x,this.type=6,this._$AN=void 0,this._$AM=d,this.options=h}get _$AU(){return this._$AM._$AU}_$AI(x){a(this,x)}}var Cd=w1.litHtmlPolyfillSupport;Cd?.(v1,y1),(w1.litHtmlVersions??=[]).push("3.3.3");var C0=(x,d,h)=>{let i=h?.renderBefore??d,p=i._$litPart$;if(p===void 0){let r=h?.renderBefore??null;i._$litPart$=p=new y1(d.insertBefore(p1(),r),r,void 0,h??{})}return p._$AI(x),p};var m1=globalThis;class P extends C{constructor(){super(...arguments),this.renderOptions={host:this},this._$Do=void 0}createRenderRoot(){let x=super.createRenderRoot();return this.renderOptions.renderBefore??=x.firstChild,x}update(x){let d=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(x),this._$Do=C0(d,this.renderRoot,this.renderOptions)}connectedCallback(){super.connectedCallback(),this._$Do?.setConnected(!0)}disconnectedCallback(){super.disconnectedCallback(),this._$Do?.setConnected(!1)}render(){return o}}P._$litElement$=!0,P.finalized=!0,m1.litElementHydrateSupport?.({LitElement:P});var Rd=m1.litElementPolyfillSupport;Rd?.({LitElement:P});(m1.litElementVersions??=[]).push("4.2.2");var R0=null;function w0(){let x=new URL("/app/assets/datastar.js",window.location.href).href;return R0??=import(x),R0}var t=null,F0=null;function g1(x){class d extends x{#x=null;#d=!1;connectedCallback(){this.#d=!0,super.connectedCallback(),wd().then(async()=>{if(!this.#d)return;if(this.requestUpdate(),await this.updateComplete,await Fd(),this.#d)this.requestUpdate()})}performUpdate(){if(!this.isUpdatePending)return;let h=t;if(!h){super.performUpdate();return}this.#x?.();let i=!0;this.#x=h.effect(()=>{if(Object.keys(h.root),i){i=!1,super.performUpdate();return}this.requestUpdate()})}disconnectedCallback(){this.#d=!1,this.#x?.(),this.#x=null,super.disconnectedCallback()}signal(h,i){let p=t?.getPath(h);return _1(p===void 0?i:p)}}return d}async function wd(){if(t)return t;return F0??=w0(),t=await F0,t}async function Fd(){await Promise.resolve(),await new Promise((x)=>requestAnimationFrame(()=>x()))}function _1(x){if(Array.isArray(x))return x.map((d)=>_1(d));if(x&&typeof x==="object")return Object.fromEntries(Object.entries(x).map(([d,h])=>[d,_1(h)]));return x}var $0=d1`
  :host {
    color-scheme: dark;
    --canvas: #0b0b0d;
    --canvas-soft: #101013;
    --surface: #141417;
    --surface-raised: #19191d;
    --surface-hover: #1e1e23;
    --line: rgba(255, 255, 255, 0.09);
    --line-strong: rgba(255, 255, 255, 0.15);
    --text: #f5f3ed;
    --text-soft: #aaa7af;
    --text-faint: #74727a;
    --neutral-soft: rgba(170, 167, 175, 0.09);
    --ember: #e38242;
    --ember-soft: rgba(227, 130, 66, 0.12);
    --violet: #a99af8;
    --violet-soft: rgba(169, 154, 248, 0.12);
    --green: #70d6a2;
    --green-soft: rgba(112, 214, 162, 0.11);
    --red: #f08282;
    --red-soft: rgba(240, 130, 130, 0.11);
    --yellow: #e7c66d;
    --yellow-soft: rgba(231, 198, 109, 0.11);
    --blue: #7bb8f0;
    --blue-soft: rgba(123, 184, 240, 0.11);
    --sans: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    --mono: "SFMono-Regular", "Cascadia Code", "Roboto Mono", Consolas, monospace;
    display: block;
    min-height: 100vh;
    background: var(--canvas);
    color: var(--text);
    font-family: var(--sans);
    font-size: 14px;
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  * { box-sizing: border-box; }
  a { color: inherit; }
  a:focus-visible { outline: 2px solid var(--ember); outline-offset: 3px; }

  .shell {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 244px minmax(0, 1fr);
    background:
      radial-gradient(circle at 70% -20%, rgba(143, 91, 59, 0.08), transparent 35%),
      var(--canvas);
  }

  .sidebar {
    position: sticky;
    top: 0;
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 18px 12px 14px;
    border-right: 1px solid var(--line);
    background: rgba(12, 12, 14, 0.94);
    backdrop-filter: blur(18px);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 42px;
    padding: 0 10px;
    color: var(--text);
    font-size: 14px;
    font-weight: 680;
    letter-spacing: -0.02em;
    text-decoration: none;
  }

  .brand-mark {
    display: grid;
    width: 27px;
    height: 27px;
    place-items: center;
    border: 1px solid rgba(255, 255, 255, 0.22);
    border-radius: 50%;
    background: var(--text);
    color: #111114;
    font-family: Georgia, serif;
    font-size: 13px;
    font-weight: 800;
  }

  .nav-section { margin-top: 22px; }
  .nav-label {
    padding: 0 11px 7px;
    color: var(--text-faint);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }

  .nav-link {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 36px;
    padding: 0 10px;
    border-radius: 7px;
    color: var(--text-soft);
    font-size: 13px;
    text-decoration: none;
    transition: 140ms ease;
  }
  .nav-link:hover { background: var(--surface-hover); color: var(--text); }
  .nav-link.active { background: var(--surface-raised); color: var(--text); box-shadow: inset 0 0 0 1px var(--line); }
  .nav-link svg { width: 15px; height: 15px; color: var(--text-faint); }
  .nav-link.active svg { color: var(--ember); }
  .nav-link .count { margin-left: auto; color: var(--text-faint); font: 10px var(--mono); }

  .sidebar-foot {
    margin-top: auto;
    padding: 12px 10px 4px;
    border-top: 1px solid var(--line);
  }
  .identity { display: flex; align-items: center; gap: 10px; }
  .identity > div { min-width: 0; }
  .avatar {
    display: grid;
    width: 28px;
    height: 28px;
    place-items: center;
    border: 1px solid var(--line-strong);
    border-radius: 50%;
    background: linear-gradient(145deg, #30241f, #19191d);
    color: #f0b486;
    font-size: 11px;
    font-weight: 750;
  }
  .identity-name { color: var(--text); font-size: 12px; font-weight: 600; }
  .identity-role { color: var(--text-faint); font-size: 10px; }
  .sign-out {
    margin-left: auto;
    color: var(--text-faint);
    font-size: 10px;
    text-decoration: none;
  }
  .sign-out:hover { color: var(--text); }

  .workspace { min-width: 0; }
  .topbar {
    position: sticky;
    z-index: 5;
    top: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 60px;
    padding: 0 clamp(22px, 4vw, 54px);
    border-bottom: 1px solid var(--line);
    background: rgba(11, 11, 13, 0.82);
    backdrop-filter: blur(18px);
  }
  .breadcrumb { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--text-faint); font-size: 12px; }
  .breadcrumb strong { overflow: hidden; color: var(--text-soft); font-weight: 550; text-overflow: ellipsis; white-space: nowrap; }
  .slash { color: #45434a; }
  .live { display: inline-flex; align-items: center; gap: 7px; color: var(--text-soft); font-size: 11px; }
  .live-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); box-shadow: 0 0 0 4px var(--green-soft); }
  .live.degraded .live-dot { background: var(--yellow); box-shadow: 0 0 0 4px var(--yellow-soft); }

  .content { width: min(100%, 1440px); margin: 0 auto; padding: 38px clamp(22px, 4vw, 54px) 72px; }
  .page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 28px; }
  .eyebrow { margin: 0 0 7px; color: var(--text-faint); font-size: 10px; font-weight: 750; letter-spacing: 0.1em; text-transform: uppercase; }
  h1 { margin: 0; font-size: clamp(25px, 3vw, 34px); font-weight: 620; letter-spacing: -0.045em; line-height: 1.1; }
  .lede { max-width: 600px; margin: 10px 0 0; color: var(--text-soft); font-size: 13px; }
  .read-only {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 7px 10px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--surface);
    color: var(--text-soft);
    font-size: 11px;
    white-space: nowrap;
  }
  .read-only svg { width: 13px; color: var(--text-faint); }

  .metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 18px; }
  .metric, .panel {
    border: 1px solid var(--line);
    background: linear-gradient(180deg, rgba(255,255,255,0.018), transparent), var(--surface);
    box-shadow: 0 16px 42px rgba(0, 0, 0, 0.12);
  }
  .metric { min-height: 112px; padding: 17px 18px; border-radius: 10px; }
  .metric-top { display: flex; align-items: center; justify-content: space-between; color: var(--text-faint); font-size: 11px; }
  .metric-top svg { width: 14px; height: 14px; }
  .metric-value { margin-top: 18px; font-size: 24px; font-weight: 570; letter-spacing: -0.04em; }
  .metric-note { margin-top: 2px; color: var(--text-faint); font-size: 10px; }

  .panel { overflow: hidden; border-radius: 11px; }
  .panel-head { display: flex; min-height: 53px; align-items: center; justify-content: space-between; gap: 16px; padding: 0 18px; border-bottom: 1px solid var(--line); }
  .panel-title { display: flex; align-items: center; gap: 9px; font-size: 12px; font-weight: 620; }
  .panel-title svg { width: 14px; height: 14px; color: var(--text-faint); }
  .panel-meta { color: var(--text-faint); font: 10px var(--mono); }
  .panel-body { padding: 18px; }

  .resource-panel { margin-bottom: 14px; }
  .chart-legend { display: flex; justify-content: flex-end; gap: 22px; padding: 14px 18px 0; color: var(--text-faint); font-size: 10px; }
  .legend { display: inline-flex; align-items: center; gap: 7px; }
  .legend i { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
  .legend strong { color: var(--text-soft); font-weight: 530; }
  .legend.cpu { color: var(--violet); }
  .legend.memory { color: var(--green); }
  .resource-chart { padding: 3px 18px 14px; }
  .resource-chart svg { display: block; width: 100%; height: 230px; overflow: visible; }
  .grid-line { stroke: rgba(255,255,255,.075); stroke-width: 1; vector-effect: non-scaling-stroke; }
  .axis-label { fill: var(--text-faint); font: 9px var(--mono); }
  .series { fill: none; stroke-width: 1.7; vector-effect: non-scaling-stroke; }
  .series.cpu { stroke: var(--violet); }
  .series.memory { stroke: var(--green); opacity: .9; }
  .chart-times { display: flex; justify-content: space-between; padding-left: 42px; color: var(--text-faint); font: 9px var(--mono); }

  .badge { display: inline-flex; align-items: center; gap: 6px; padding: 3px 7px; border: 1px solid var(--line); border-radius: 999px; color: var(--text-soft); font-size: 10px; font-weight: 560; white-space: nowrap; }
  .badge::before { content: ""; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
  .badge.running, .badge.active { border-color: rgba(123,184,240,.2); background: var(--blue-soft); color: var(--blue); }
  .badge.succeeded, .badge.success, .badge.online { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.degraded { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.timed_out, .badge.lost { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
  .badge.cancelled, .badge.canceled { border-color: var(--line); background: var(--neutral-soft); color: var(--text-soft); }
  .badge.build { border-color: rgba(169,154,248,.18); background: var(--violet-soft); color: var(--violet); }

  .table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th { height: 38px; padding: 0 16px; color: var(--text-faint); font-size: 9px; font-weight: 700; letter-spacing: .08em; text-align: left; text-transform: uppercase; }
  td { height: 54px; padding: 8px 16px; border-top: 1px solid var(--line); color: var(--text-soft); font-size: 11px; }
  td.primary { min-width: 200px; color: var(--text); }
  td.primary a { display: flex; align-items: center; gap: 9px; text-decoration: none; }
  td.primary a:hover { color: var(--ember); }
  .mono { font-family: var(--mono); font-size: 10px; }
  .muted { color: var(--text-faint); }

  .empty { display: grid; min-height: 180px; place-items: center; padding: 32px; color: var(--text-faint); text-align: center; }
  .empty svg { width: 24px; margin-bottom: 10px; }
  .empty strong { display: block; color: var(--text-soft); font-size: 12px; font-weight: 580; }
  .empty span { display: block; margin-top: 4px; font-size: 10px; }

  .project-banner { position: relative; display: grid; grid-template-columns: 1fr auto; gap: 24px; margin-bottom: 18px; padding: 24px; overflow: hidden; border: 1px solid var(--line); border-radius: 11px; background: linear-gradient(115deg, rgba(227,130,66,.10), transparent 40%), var(--surface); }
  .project-banner::after { content: ""; position: absolute; right: -50px; bottom: -90px; width: 280px; height: 180px; border: 1px solid rgba(227,130,66,.12); border-radius: 50%; transform: rotate(-11deg); }
  .project-name { font-size: 19px; font-weight: 600; letter-spacing: -.03em; }
  .project-slug { margin-top: 5px; color: var(--text-faint); font: 10px var(--mono); }
  .project-facts { position: relative; z-index: 1; display: flex; align-items: center; gap: 24px; }
  .fact strong { display: block; font-size: 16px; font-weight: 580; }
  .fact span { color: var(--text-faint); font-size: 9px; letter-spacing: .06em; text-transform: uppercase; }
  .digest { overflow: hidden; max-width: 100%; color: var(--text-soft); font: 10px var(--mono); text-overflow: ellipsis; white-space: nowrap; }

  .trend-grid { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(280px, .6fr); gap: 14px; margin-bottom: 14px; }
  .trend-panel { min-height: 190px; }
  .duration-bars { display: flex; height: 136px; align-items: end; gap: 4px; padding: 22px 18px 18px; }
  .duration-bars > i { min-width: 3px; flex: 1; border-radius: 2px 2px 0 0; background: linear-gradient(180deg, var(--ember), rgba(227,130,66,.35)); }
  .duration-bars .empty { width: 100%; min-height: 90px; }
  .project-health { display: grid; grid-template-columns: 1fr 1fr; }
  .project-health > div { display: grid; place-content: center; padding: 20px; border-right: 1px solid var(--line); text-align: center; }
  .project-health > div:last-child { border-right: 0; }
  .project-health span { color: var(--text-faint); font-size: 10px; }
  .project-health strong { display: block; margin-top: 9px; font-size: 24px; font-weight: 570; letter-spacing: -.04em; }

  .detail-grid { display: grid; grid-template-columns: minmax(0, 1.3fr) minmax(280px, .7fr); gap: 14px; }
  .detail-stack { display: grid; gap: 14px; align-content: start; }
  .command { margin: 0; padding: 18px; overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; background: #0d0d10; color: #d8d5dc; font: 11px/1.7 var(--mono); white-space: pre-wrap; }
  .command .prompt { color: var(--ember); user-select: none; }
  .definition { display: grid; grid-template-columns: 120px minmax(0,1fr); gap: 0; margin: 0; }
  .definition dt, .definition dd { min-height: 42px; margin: 0; padding: 11px 14px; border-bottom: 1px solid var(--line); }
  .definition dt { color: var(--text-faint); font-size: 10px; }
  .definition dd { overflow: hidden; color: var(--text-soft); font: 10px var(--mono); text-overflow: ellipsis; }
  .definition > :nth-last-child(-n+2) { border-bottom: 0; }
  .log { max-height: 430px; margin: 0; padding: 18px; overflow: auto; background: #09090b; color: #c9c6ce; font: 10.5px/1.7 var(--mono); tab-size: 2; white-space: pre-wrap; }
  .log-note { padding: 8px 18px; border-top: 1px solid var(--line); color: var(--text-faint); font-size: 9px; }

  .audit-action { color: var(--text); font: 10px var(--mono); }
  .metadata { display: flex; max-width: 360px; flex-wrap: wrap; gap: 4px; }
  .metadata span { padding: 2px 5px; border: 1px solid var(--line); border-radius: 4px; color: var(--text-faint); font: 9px var(--mono); }

  .loading { display: grid; min-height: calc(100vh - 60px); place-items: center; color: var(--text-faint); }
  .loader { display: flex; align-items: center; gap: 9px; }
  .loader::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: var(--ember); animation: pulse 1.4s ease-in-out infinite; }
  @keyframes pulse { 50% { opacity: .28; transform: scale(.72); } }

  @media (max-width: 1040px) {
    .metrics { grid-template-columns: repeat(2, 1fr); }
    .detail-grid, .trend-grid { grid-template-columns: 1fr; }
  }

  @media (max-width: 760px) {
    .shell { display: block; }
    .sidebar { position: static; width: 100%; height: auto; padding: 10px 12px; border-right: 0; border-bottom: 1px solid var(--line); }
    .sidebar .brand { padding: 0 4px; }
    .nav-section { display: flex; gap: 4px; margin-top: 8px; overflow-x: auto; }
    .nav-section .nav-label { display: none; }
    .nav-link { flex: 0 0 auto; min-height: 32px; }
    .projects-nav { padding-bottom: 3px; }
    .sidebar-foot { display: none; }
    .topbar { top: 0; height: 50px; padding: 0 16px; }
    .content { padding: 26px 16px 56px; }
    .page-head { display: block; }
    .read-only { margin-top: 16px; }
    .metrics { grid-template-columns: 1fr 1fr; }
    .metric { min-height: 96px; padding: 14px; }
    .metric-value { margin-top: 12px; font-size: 20px; }
    .project-banner { grid-template-columns: 1fr; }
    .project-facts { justify-content: space-between; }
    .resource-chart svg { height: 190px; }
    .chart-legend { justify-content: flex-start; flex-wrap: wrap; }
    th:nth-child(3), td:nth-child(3), th:nth-child(5), td:nth-child(5), th:nth-child(6), td:nth-child(6) { display: none; }
  }

  @media (max-width: 430px) {
    .metrics { grid-template-columns: 1fr; }
    .project-facts { gap: 12px; }
    .definition { grid-template-columns: 96px minmax(0,1fr); }
  }

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { scroll-behavior: auto !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; transition-duration: .01ms !important; }
  }
`;function w(x,d=12){if(x.length<=d)return x;return`${x.slice(0,d-1)}…`}function u1(x){if(!x)return"Not configured";let d=x.includes("@")?x.split("@").at(-1):x;return d.length>23?`${d.slice(0,16)}…${d.slice(-6)}`:d}function z1(x,d=Date.now()){let h=Date.parse(x);if(!Number.isFinite(h))return"—";let i=Math.max(0,Math.round((d-h)/1000));if(i<5)return"now";if(i<60)return`${i}s ago`;let p=Math.floor(i/60);if(p<60)return`${p}m ago`;let r=Math.floor(p/60);if(r<24)return`${r}h ago`;return`${Math.floor(r/24)}d ago`}function X1(x,d,h=Date.now()){if(!x)return"—";let i=Date.parse(x),p=d?Date.parse(d):h;if(!Number.isFinite(i)||!Number.isFinite(p)||p<i)return"—";let r=p-i;if(r<1000)return`${r}ms`;let v=r/1000;if(v<60)return`${v.toFixed(v<10?1:0)}s`;return`${Math.floor(v/60)}m ${Math.floor(v%60)}s`}function m0(x){let d=x.filter((i)=>["succeeded","success","failed","cancelled"].includes(i));if(d.length===0)return"—";let h=d.filter((i)=>i==="succeeded"||i==="success").length;return`${Math.round(h/d.length*100)}%`}function b(x){if(!Number.isFinite(x)||x<=0)return"—";let d=["B","KB","MB","GB","TB"],h=Math.min(Math.floor(Math.log(x)/Math.log(1024)),d.length-1),i=x/1024**h;return`${i>=10||Number.isInteger(i)?i.toFixed(0):i.toFixed(1)} ${d[h]}`}function j(x){if(!Number.isFinite(x))return"—";return`${Math.round(Math.max(0,Math.min(1,x))*100)}%`}function e(x){if(x==null||!Number.isFinite(x)||x<0)return"—";if(x<1000)return`${Math.round(x)}ms`;let d=x/1000;if(d<60)return`${d.toFixed(d<10?1:0)}s`;return`${Math.floor(d/60)}m ${Math.floor(d%60)}s`}function o1(x,d){if(!x)return null;if(typeof x==="function")return x(d);return x}function _0(x){if("cell"in x&&x.cell)return o1(x.cell.column.columnDef.cell,x.cell.getContext());if("header"in x&&x.header)return o1(x.header.column.columnDef.header,x.header.getContext());if("footer"in x&&x.footer)return o1(x.footer.column.columnDef.footer,x.footer.getContext());return null}var Y=function(x){return x[x.None=0]="None",x[x.Mutable=1]="Mutable",x[x.Watching=2]="Watching",x[x.RecursedCheck=4]="RecursedCheck",x[x.Recursed=8]="Recursed",x[x.Dirty=16]="Dirty",x[x.Pending=32]="Pending",x}({});function u0({update:x,notify:d,unwatched:h}){return{link:i,unlink:p,propagate:r,checkDirty:v,shallowPropagate:k};function i(q,y,z){let W=y.depsTail;if(W!==void 0&&W.dep===q)return;let Q=W!==void 0?W.nextDep:y.deps;if(Q!==void 0&&Q.dep===q){Q.version=z,y.depsTail=Q;return}let T=q.subsTail;if(T!==void 0&&T.version===z&&T.sub===y)return;let n=y.depsTail=q.subsTail={version:z,dep:q,sub:y,prevDep:W,nextDep:Q,prevSub:T,nextSub:void 0};if(Q!==void 0)Q.prevDep=n;if(W!==void 0)W.nextDep=n;else y.deps=n;if(T!==void 0)T.nextSub=n;else q.subs=n}function p(q,y=q.sub){let{dep:z,prevDep:W,nextDep:Q,nextSub:T,prevSub:n}=q;if(Q!==void 0)Q.prevDep=W;else y.depsTail=W;if(W!==void 0)W.nextDep=Q;else y.deps=Q;if(T!==void 0)T.prevSub=n;else z.subsTail=n;if(n!==void 0)n.nextSub=T;else if((z.subs=T)===void 0)h(z);return Q}function r(q){let y=q.nextSub,z;x:do{let W=q.sub,Q=W.flags;if(!(Q&(Y.RecursedCheck|Y.Recursed|Y.Dirty|Y.Pending)))W.flags=Q|Y.Pending;else if(!(Q&(Y.RecursedCheck|Y.Recursed)))Q=Y.None;else if(!(Q&Y.RecursedCheck))W.flags=Q&~Y.Recursed|Y.Pending;else if(!(Q&(Y.Dirty|Y.Pending))&&N(q,W))W.flags=Q|(Y.Recursed|Y.Pending),Q&=Y.Mutable;else Q=Y.None;if(Q&Y.Watching)d(W);if(Q&Y.Mutable){let T=W.subs;if(T!==void 0){let n=(q=T).nextSub;if(n!==void 0)z={value:y,prev:z},y=n;continue}}if((q=y)!==void 0){y=q.nextSub;continue}while(z!==void 0)if(q=z.value,z=z.prev,q!==void 0){y=q.nextSub;continue x}break}while(!0)}function v(q,y){let z,W=0,Q=!1;x:do{let T=q.dep,n=T.flags;if(y.flags&Y.Dirty)Q=!0;else if((n&(Y.Mutable|Y.Dirty))===(Y.Mutable|Y.Dirty)){if(x(T)){let g=T.subs;if(g.nextSub!==void 0)k(g);Q=!0}}else if((n&(Y.Mutable|Y.Pending))===(Y.Mutable|Y.Pending)){if(q.nextSub!==void 0||q.prevSub!==void 0)z={value:q,prev:z};q=T.deps,y=T,++W;continue}if(!Q){let g=q.nextDep;if(g!==void 0){q=g;continue}}while(W--){let g=y.subs,X=g.nextSub!==void 0;if(X)q=z.value,z=z.prev;else q=g;if(Q){if(x(y)){if(X)k(g);y=q.sub;continue}Q=!1}else y.flags&=~Y.Pending;y=q.sub;let E=q.nextDep;if(E!==void 0){q=E;continue x}}return Q}while(!0)}function k(q){do{let y=q.sub,z=y.flags;if((z&(Y.Pending|Y.Dirty))===Y.Pending){if(y.flags=z|Y.Dirty,(z&(Y.Watching|Y.RecursedCheck))===Y.Watching)d(y)}}while((q=q.nextSub)!==void 0)}function N(q,y){let z=y.depsTail;while(z!==void 0){if(z===q)return!0;z=z.prevDep}return!1}}function a0(x,d,h){let i=typeof x==="object",p=i?x:void 0;return{next:(i?x.next:x)?.bind(p),error:(i?x.error:d)?.bind(p),complete:(i?x.complete:h)?.bind(p)}}var b1=[],K1=0,{link:o0,unlink:$d,propagate:md,checkDirty:t0,shallowPropagate:b0}=u0({update(x){return x._update()},notify(x){b1[a1++]=x,x.flags&=~Y.Watching},unwatched(x){if(x.depsTail!==void 0)x.depsTail=void 0,x.flags=Y.Mutable|Y.Dirty,B1(x)}}),Z1=0,a1=0,s,t1=0;function e1(x){try{++t1,x()}finally{if(!--t1)l1()}}function B1(x){let d=x.depsTail,h=d!==void 0?d.nextDep:x.deps;while(h!==void 0)h=$d(h,x)}function l1(){if(t1>0)return;while(Z1<a1){let x=b1[Z1];b1[Z1++]=void 0,x.notify()}Z1=0,a1=0}function U1(x,d){let h=typeof x==="function",i=x,p={_snapshot:h?void 0:x,subs:void 0,subsTail:void 0,deps:void 0,depsTail:void 0,flags:h?Y.None:Y.Mutable,get(){if(s!==void 0)o0(p,s,K1);return p._snapshot},subscribe(r){let v=a0(r),k={current:!1},N=_d(()=>{if(p.get(),!k.current)k.current=!0;else v.next?.(p._snapshot)});return{unsubscribe:()=>{N.stop()}}},_update(r){let v=s,k=d?.compare??Object.is;if(h)s=p,++K1,p.depsTail=void 0;else if(r===void 0)return!1;if(h)p.flags=Y.Mutable|Y.RecursedCheck;try{let N=p._snapshot,q=typeof r==="function"?r(N):r===void 0&&h?i(N):r;if(N===void 0||!k(N,q))return p._snapshot=q,!0;return!1}finally{if(s=v,h)p.flags&=~Y.RecursedCheck;B1(p)}}};if(h)p.flags=Y.Mutable|Y.Dirty,p.get=function(){let r=p.flags;if(r&Y.Dirty||r&Y.Pending&&t0(p.deps,p)){if(p._update()){let v=p.subs;if(v!==void 0)b0(v)}}else if(r&Y.Pending)p.flags=r&~Y.Pending;if(s!==void 0)o0(p,s,K1);return p._snapshot};else p.set=function(r){if(p._update(r)){let v=p.subs;if(v!==void 0)md(v),b0(v),l1()}};return p}function _d(x){let d=()=>{let i=s;s=h,++K1,h.depsTail=void 0,h.flags=Y.Watching|Y.RecursedCheck;try{return x()}finally{s=i,h.flags&=~Y.RecursedCheck,B1(h)}},h={deps:void 0,depsTail:void 0,subs:void 0,subsTail:void 0,flags:Y.Watching|Y.RecursedCheck,notify(){let i=this.flags;if(i&Y.Dirty||i&Y.Pending&&t0(this.deps,this))d();else this.flags=Y.Watching},stop(){this.flags=Y.None,this.depsTail=void 0,B1(this)}};return d(),h}function e0(){return{createOptionsStore:!0,wrapExternalAtoms:!1,addSubscription:()=>{throw Error("Feature not supported in current reactivity implementation")},unmount:()=>{throw Error("Feature not supported in current reactivity implementation")},schedule:(x)=>queueMicrotask(()=>x()),batch:e1,untrack:(x)=>x(),createReadonlyAtom:(x,d)=>{return U1(()=>x(),{compare:d===null||d===void 0?void 0:d.compare})},createWritableAtom:(x,d)=>{return U1(x,{compare:d===null||d===void 0?void 0:d.compare})}}}function V(x,d){return typeof x==="function"?x(d):x}function D(x){if(Array.isArray(x))return x.map(D);if(x&&typeof x==="object"){let d=Object.getPrototypeOf(x);if(d!==Object.prototype&&d!==null)return x;let h={},i=Object.keys(x);for(let p=0;p<i.length;p++){let r=i[p];h[r]=D(x[r])}return h}return x}function M1(x,d){return(h)=>{var i;(((i=d.options.atoms)===null||i===void 0?void 0:i[x])??d.baseAtoms[x]).set((p)=>V(h,p))}}function L1(x){return x instanceof Function}function l0(x,d){let h=[],i=(p)=>{p.forEach((r)=>{h.push(r);let v=d(r);if(v.length)i(v)})};return i(x),h}var ud=({fn:x,memoDeps:d,onAfterCompare:h,onAfterUpdate:i,onBeforeCompare:p,onBeforeUpdate:r})=>{let v=[],k;return(q)=>{p===null||p===void 0||p();let y=d===null||d===void 0?void 0:d(q),z=!y||y.length!==(v===null||v===void 0?void 0:v.length);if(!z&&y){for(let W=0;W<y.length;W++)if(y[W]!==v[W]){z=!0;break}}if(h===null||h===void 0||h(z),!z)return k;return v=y,r===null||r===void 0||r(),k=x(...y??[]),i===null||i===void 0||i(k),k}},od=(x,d)=>{x=String(x);while(x.length<d)x=" "+x;return x};function l({feature:x,fnName:d,objectId:h,onAfterUpdate:i,table:p,...r}){let v,k,N,q,y=0,z,W;function Q(g,X){var E;let S=y===0?"(1st run)":X?"(rerun #"+y+")":"(cache)";y++,console.groupCollapsed(`%c⏱ ${od(`${g.toFixed(1)} ms`,12)} %c${S}%c ${d}%c ${h?`(${d.split(".")[0]}Id: ${h})`:""}`,`font-size: .6rem; font-weight: bold; ${X?`color: hsl(
        ${Math.max(0,Math.min(120-Math.log10(g)*60,120))}deg 100% 31%);`:""} `,`color: ${y<2?"#FF00FF":"#FF1493"}`,"color: #666","color: #87CEEB"),console.info({feature:x,state:p.store.state,deps:(E=r.memoDeps)===null||E===void 0?void 0:E.toString()}),console.trace(),console.groupEnd()}let T=()=>{if(!i)return;let{schedule:g,untrack:X}=p._reactivity;g(()=>X(()=>i()))};return ud({...r,...{onAfterUpdate:()=>{T()}}})}function xx(x,d="_"){let[h,i]=x.split(d);return{fnKey:i,fnName:`${h}.${i}`,parentName:h}}function A(x,d,h){for(let[i,{fn:p,memoDeps:r}]of Object.entries(h)){let{fnKey:v,fnName:k}=xx(i);d[v]=r?l({memoDeps:r,fn:p,fnName:k,table:d,feature:x}):p}}function H(x,d,h,i){for(let[p,{fn:r,memoDeps:v}]of Object.entries(i)){let{fnKey:k,fnName:N}=xx(p);if(v){let q=`_memo_${k}`;d[k]=function(...y){if(!this[q]){let z=this;this[q]=l({memoDeps:(W)=>v(z,W),fn:(...W)=>r(z,...W),fnName:N,objectId:z.id,table:h,feature:x})}return this[q](...y)}}else d[k]=function(...q){return r(this,...q)}}}function c(x,d,h,...i){var p;return((p=x[d])===null||p===void 0?void 0:p.call(x,...i))??h(x,...i)}function dx(x){return x.row.getValue(x.column.id)}function hx(x){return x.getValue()??x.table.options.renderFallbackValue}function ix(x){return{table:x.table,column:x.column,row:x.row,cell:x,getValue:()=>x.getValue(),renderValue:()=>x.renderValue()}}var px={assignCellPrototype:(x,d)=>{H("coreCellsFeature",x,d,{cell_getValue:{fn:(h)=>dx(h)},cell_renderValue:{fn:(h)=>hx(h)},cell_getContext:{fn:(h)=>ix(h),memoDeps:(h)=>[h]}})}};function bd(x){if(!x._headerPrototype){x._headerPrototype={table:x};let i=Object.values(x._features);for(let p=0;p<i.length;p++){var d,h;(d=(h=i[p]).assignHeaderPrototype)===null||d===void 0||d.call(h,x._headerPrototype,x)}}return x._headerPrototype}function x0(x,d,h){let i=bd(x),p=Object.create(i);return p.colSpan=0,p.column=d,p.depth=h.depth,p.headerGroup=null,p.id=h.id??d.id,p.index=h.index,p.isPlaceholder=!!h.isPlaceholder,p.placeholderId=h.placeholderId,p.rowSpan=0,p.subHeaders=[],p}function rx(){return{left:[],right:[]}}function F(x){var d;let h=x.columns;return(h.length?h.some((i)=>c(i,"getIsVisible",F)):(d=x.table.atoms.columnVisibility)===null||d===void 0||(d=d.get())===null||d===void 0?void 0:d[x.id])??!0}function vx(x){return x.getAllLeafColumns().filter((d)=>c(d,"getIsVisible",F))}function d0(x,d,h,i){var p;let r=0,v=(y,z=1)=>{r=Math.max(r,z);for(let W=0;W<y.length;W++){let Q=y[W];if(c(Q,"getIsVisible",F)){if(Q.columns.length)v(Q.columns,z+1)}}};v(x);let k=[],N=(y,z)=>{let W={depth:z,id:[i,`${z}`].filter(Boolean).join("_"),headers:[]},Q=[];if(y.forEach((T)=>{let n=Q[Q.length-1],g=T.column.depth===W.depth,X,E=!1;if(g&&T.column.parent)X=T.column.parent;else X=T.column,E=!0;if(n&&n.column===X)n.subHeaders.push(T);else{let S=x0(h,X,{id:[i,z,X.id,T.id].filter(Boolean).join("_"),isPlaceholder:E,placeholderId:E?`${Q.filter(($)=>$.column===X).length}`:void 0,depth:z,index:Q.length});S.subHeaders.push(T),Q.push(S)}W.headers.push(T),T.headerGroup=W}),k.push(W),z>0)N(Q,z-1)};N(d.map((y,z)=>x0(h,y,{depth:r,index:z})),r-1),k.reverse();let q=(y)=>{let z=[];for(let W=0;W<y.length;W++){let Q=y[W];if(!c(Q.column,"getIsVisible",F))continue;let T=0,n=1/0;if(Q.subHeaders.length){let g=q(Q.subHeaders);for(let X=0;X<g.length;X++){let E=g[X];if(T+=E.colSpan,E.rowSpan<n)n=E.rowSpan}}else T=1,n=0;Q.colSpan=T,Q.rowSpan=n,z.push({colSpan:T,rowSpan:Q.rowSpan})}return z};return q(((p=k[0])===null||p===void 0?void 0:p.headers)??[]),k}function ad(x){if(!x._columnPrototype){x._columnPrototype={table:x};let i=Object.values(x._features);for(let p=0;p<i.length;p++){var d,h;(d=(h=i[p]).assignColumnPrototype)===null||d===void 0||d.call(h,x._columnPrototype,x)}}return x._columnPrototype}function yx(x,d,h,i){let p={...x.getDefaultColumnDef(),...d},r=p.accessorKey,v=p.id??(r?r.replaceAll(".","_"):void 0)??(typeof p.header==="string"?p.header:void 0),k;if(p.accessorFn)k=p.accessorFn;else if(r)if(r.includes(".")){let y=r.split(".");k=(z)=>{let W=z;for(let Q=0;Q<y.length;Q++){let T=y[Q];W=W===null||W===void 0?void 0:W[T]}return W}}else k=(y)=>y[p.accessorKey];if(!v)throw Error();let N=ad(x),q=Object.create(N);return q.accessorFn=k,q.columnDef=p,q.columns=[],q.depth=h,q.id=`${String(v)}`,q.parent=i,q}function h0(x){var d;let h=(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get();return(i)=>{let p=[];if(!(h===null||h===void 0?void 0:h.length))p=i;else{let r=new Map;for(let v=0;v<i.length;v++){let k=i[v];r.set(k.id,k)}for(let v=0;v<h.length;v++){let k=h[v],N=r.get(k);if(N)p.push(N),r.delete(k)}for(let v=0;v<i.length;v++){let k=i[v];if(r.has(k.id))p.push(k)}}return td(x,p)}}function td(x,d){var h;let i=((h=x.atoms.grouping)===null||h===void 0?void 0:h.get())??[],{groupedColumnMode:p}=x.options;if(!i.length||!p)return d;let r=d.filter((N)=>!i.includes(N.id));if(p==="remove")return r;let v=new Map;for(let N=0;N<d.length;N++){let q=d[N];v.set(q.id,q)}let k=[];for(let N=0;N<i.length;N++){let q=v.get(i[N]);if(q)k.push(q)}return[...k,...r]}function kx(x){return[x,...x.columns.flatMap((d)=>d.getFlatColumns())]}function zx(x){if(x.columns.length){let d=x.columns.flatMap((h)=>h.getLeafColumns());return c(x.table,"getOrderColumns",h0)(d)}return[x]}function qx(x){return{header:(d)=>{let h=d.header.column.columnDef;if(h.accessorKey)return h.accessorKey;if(h.accessorFn)return h.id;return null},cell:(d)=>{var h,i;return((h=d.renderValue())===null||h===void 0||(i=h.toString)===null||i===void 0?void 0:i.call(h))??null},...Object.values(x._features).reduce((d,h)=>{var i;return Object.assign(d,(i=h.getDefaultColumnDef)===null||i===void 0?void 0:i.call(h))},{}),...x.options.defaultColumn}}function Nx(x){let d=(h,i,p=0)=>{return h.map((r)=>{let v=yx(x,r,p,i),k=r;return v.columns=k.columns?d(k.columns,v,p+1):[],v})};return d(x.options.columns)}function Qx(x){return x.getAllColumns().flatMap((d)=>d.getFlatColumns())}function Tx(x){let d={},h=x.getAllFlatColumns();for(let i=0;i<h.length;i++){let p=h[i];d[p.id]=p}return d}function Wx(x){let d=x.getAllColumns().flatMap((h)=>h.getLeafColumns());return c(x,"getOrderColumns",h0)(d)}function Yx(x){let d={},h=x.getAllLeafColumns();for(let i=0;i<h.length;i++){let p=h[i];d[p.id]=p}return d}function j1(x,d){return x.getAllFlatColumnsById()[d]}var nx={assignColumnPrototype:(x,d)=>{H("coreColumnsFeature",x,d,{column_getFlatColumns:{fn:(h)=>kx(h),memoDeps:(h)=>[h.table.options.columns]},column_getLeafColumns:{fn:(h)=>zx(h),memoDeps:(h)=>{var i,p;return[(i=h.table.atoms.columnOrder)===null||i===void 0?void 0:i.get(),(p=h.table.atoms.grouping)===null||p===void 0?void 0:p.get(),h.table.options.columns,h.table.options.groupedColumnMode]}}})},constructTableAPIs:(x)=>{A("coreColumnsFeature",x,{table_getDefaultColumnDef:{fn:()=>qx(x),memoDeps:()=>[x.options.defaultColumn]},table_getAllColumns:{fn:()=>Nx(x),memoDeps:()=>[x.options.columns]},table_getAllFlatColumns:{fn:()=>Qx(x),memoDeps:()=>[x.options.columns]},table_getAllFlatColumnsById:{fn:()=>Tx(x),memoDeps:()=>[x.options.columns]},table_getAllLeafColumns:{fn:()=>Wx(x),memoDeps:()=>{var d,h;return[(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get(),(h=x.atoms.grouping)===null||h===void 0?void 0:h.get(),x.options.columns,x.options.groupedColumnMode]}},table_getAllLeafColumnsById:{fn:()=>Yx(x),memoDeps:()=>[x.getAllLeafColumns()]},table_getColumn:{fn:(d)=>j1(x,d)}})}};function Ex(x){let d=[],h=(i)=>{if(i.subHeaders.length)i.subHeaders.map(h);d.push(i)};return h(x),d}function Jx(x){return{column:x.column,header:x,table:x.column.table}}function gx(x){var d;let{left:h,right:i}=((d=x.atoms.columnPinning)===null||d===void 0?void 0:d.get())??rx(),p=x.getAllColumns(),r=c(x,"getVisibleLeafColumns",vx);if(!h.length&&!i.length)return d0(p,r,x);let v=x.getAllLeafColumnsById(),k=[];for(let y=0;y<h.length;y++){let z=v[h[y]];if(z&&c(z,"getIsVisible",F))k.push(z)}let N=[];for(let y=0;y<i.length;y++){let z=v[i[y]];if(z&&c(z,"getIsVisible",F))N.push(z)}let q=r.filter((y)=>!h.includes(y.id)&&!i.includes(y.id));return d0(p,[...k,...q,...N],x)}function Xx(x){return[...x.getHeaderGroups()].reverse()}function Zx(x){let d=x.getHeaderGroups(),h=[];for(let i=0;i<d.length;i++){let p=d[i].headers;for(let r=0;r<p.length;r++)h.push(p[r])}return h}function Kx(x){var d;let h=((d=x.getHeaderGroups()[0])===null||d===void 0?void 0:d.headers)??[],i=[];for(let p=0;p<h.length;p++){let r=h[p].getLeafHeaders();for(let v=0;v<r.length;v++)i.push(r[v])}return i}var Bx={assignHeaderPrototype:(x,d)=>{H("coreHeadersFeature",x,d,{header_getLeafHeaders:{fn:(h)=>Ex(h),memoDeps:(h)=>[h.column.table.options.columns]},header_getContext:{fn:(h)=>Jx(h),memoDeps:(h)=>[h.column.table.options.columns]}})},constructTableAPIs:(x)=>{A("coreHeadersFeature",x,{table_getHeaderGroups:{fn:()=>gx(x),memoDeps:()=>{var d,h,i,p;return[x.options.columns,(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get(),(h=x.atoms.grouping)===null||h===void 0?void 0:h.get(),(i=x.atoms.columnPinning)===null||i===void 0?void 0:i.get(),(p=x.atoms.columnVisibility)===null||p===void 0?void 0:p.get(),x.options.groupedColumnMode]}},table_getFooterGroups:{fn:()=>Xx(x),memoDeps:()=>[x.getHeaderGroups()]},table_getFlatHeaders:{fn:()=>Zx(x),memoDeps:()=>[x.getHeaderGroups()]},table_getLeafHeaders:{fn:()=>Kx(x),memoDeps:()=>[x.getHeaderGroups()]}})}};function ed(x){if(!x._rowPrototype){x._rowPrototype={table:x};let i=Object.values(x._features);for(let p=0;p<i.length;p++){var d,h;(d=(h=i[p]).assignRowPrototype)===null||d===void 0||d.call(h,x._rowPrototype,x)}}return x._rowPrototype}var q1=(x,d,h,i,p,r,v)=>{let k=ed(x),N=Object.create(k);N._uniqueValuesCache={},N._valuesCache={},N.depth=p,N.id=d,N.index=i,N.original=h,N.parentId=v,N.subRows=r??[];let q=Object.values(x._features);for(let W=0;W<q.length;W++){var y,z;(y=(z=q[W]).initRowInstanceData)===null||y===void 0||y.call(z,N)}return N};var i0=0;function A1(x){if(x.options.autoResetAll??x.options.autoResetPageIndex??!x.options.manualPagination)dh(x)}function ld(x,d){var h,i;let p=(r)=>{return V(d,r)};return(h=(i=x.options).onPaginationChange)===null||h===void 0?void 0:h.call(i,p)}function xh(x,d){ld(x,(h)=>{let i=V(d,h.pageIndex),p=typeof x.options.pageCount>"u"||x.options.pageCount===-1?Number.MAX_SAFE_INTEGER:x.options.pageCount-1;return i=Math.max(0,Math.min(i,p)),{...h,pageIndex:i}})}function dh(x,d){var h,i;let p=((h=x.atoms.pagination)===null||h===void 0||(h=h.get())===null||h===void 0?void 0:h.pageIndex)??i0,r=d?i0:((i=x.initialState.pagination)===null||i===void 0?void 0:i.pageIndex)??i0;if(r===p)return;xh(x,r)}function N1(){return(x)=>{return l({feature:"coreRowModelsFeature",table:x,fnName:"table.getCoreRowModel",memoDeps:()=>[x.options.data],fn:()=>hh(x,x.options.data),onAfterUpdate:()=>A1(x)})}}function hh(x,d){let h={rows:[],flatRows:[],rowsById:{}},i=(p,r=0,v)=>{let k=[];for(let q=0;q<p.length;q++){let y=p[q],z=q1(x,x.getRowId(y,q,v),y,q,r,void 0,v===null||v===void 0?void 0:v.id);if(h.flatRows.push(z),h.rowsById[z.id]=z,k.push(z),x.options.getSubRows){var N;if(z.originalSubRows=x.options.getSubRows(y,q),(N=z.originalSubRows)===null||N===void 0?void 0:N.length)z.subRows=i(z.originalSubRows,r+1,z)}}return k};return h.rows=i(d),h}function Ux(x){if(!x._rowModels.coreRowModel){var d,h;x._rowModels.coreRowModel=((d=(h=x.options.features).coreRowModel)===null||d===void 0?void 0:d.call(h,x))??N1()(x)}return x._rowModels.coreRowModel()}function Mx(x){return x.getCoreRowModel()}function Lx(x){if(!x._rowModels.filteredRowModel){var d,h;x._rowModels.filteredRowModel=(d=(h=x.options.features).filteredRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualFiltering||!x._rowModels.filteredRowModel)return x.getPreFilteredRowModel();return x._rowModels.filteredRowModel()}function jx(x){return x.getFilteredRowModel()}function Ax(x){if(!x._rowModels.groupedRowModel){var d,h;x._rowModels.groupedRowModel=(d=(h=x.options.features).groupedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualGrouping||!x._rowModels.groupedRowModel)return x.getPreGroupedRowModel();return x._rowModels.groupedRowModel()}function cx(x){return x.getGroupedRowModel()}function Gx(x){if(!x._rowModels.sortedRowModel){var d,h;x._rowModels.sortedRowModel=(d=(h=x.options.features).sortedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualSorting||!x._rowModels.sortedRowModel)return x.getPreSortedRowModel();return x._rowModels.sortedRowModel()}function Dx(x){return x.getSortedRowModel()}function Hx(x){if(!x._rowModels.expandedRowModel){var d,h;x._rowModels.expandedRowModel=(d=(h=x.options.features).expandedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualExpanding||!x._rowModels.expandedRowModel)return x.getPreExpandedRowModel();return x._rowModels.expandedRowModel()}function Ox(x){return x.getExpandedRowModel()}function fx(x){if(!x._rowModels.paginatedRowModel){var d,h;x._rowModels.paginatedRowModel=(d=(h=x.options.features).paginatedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualPagination||!x._rowModels.paginatedRowModel)return x.getPrePaginatedRowModel();return x._rowModels.paginatedRowModel()}function sx(x){return x.getPaginatedRowModel()}var Px={constructTableAPIs:(x)=>{A("coreRowModelsFeature",x,{table_getCoreRowModel:{fn:()=>Ux(x)},table_getPreFilteredRowModel:{fn:()=>Mx(x)},table_getFilteredRowModel:{fn:()=>Lx(x)},table_getPreGroupedRowModel:{fn:()=>jx(x)},table_getGroupedRowModel:{fn:()=>Ax(x)},table_getPreSortedRowModel:{fn:()=>cx(x)},table_getSortedRowModel:{fn:()=>Gx(x)},table_getPreExpandedRowModel:{fn:()=>Dx(x)},table_getExpandedRowModel:{fn:()=>Hx(x)},table_getPrePaginatedRowModel:{fn:()=>Ox(x)},table_getPaginatedRowModel:{fn:()=>fx(x)},table_getRowModel:{fn:()=>sx(x)}})}};function ih(x){if(!x._cellPrototype){x._cellPrototype={table:x};let i=Object.values(x._features);for(let p=0;p<i.length;p++){var d,h;(d=(h=i[p]).assignCellPrototype)===null||d===void 0||d.call(h,x._cellPrototype,x)}}return x._cellPrototype}function Vx(x,d,h){let i=ih(h),p=Object.create(i);return p.column=x,p.id=`${d.id}_${x.id}`,p.row=d,p}function Sx(x,d){if(x._valuesCache.hasOwnProperty(d))return x._valuesCache[d];let h=x.table.getColumn(d);if(!(h===null||h===void 0?void 0:h.accessorFn))return;return x._valuesCache[d]=h.accessorFn(x.original,x.index),x._valuesCache[d]}function Ix(x,d){if(x._uniqueValuesCache.hasOwnProperty(d))return x._uniqueValuesCache[d];let h=x.table.getColumn(d);if(!(h===null||h===void 0?void 0:h.accessorFn))return;if(!h.columnDef.getUniqueValues)return x._uniqueValuesCache[d]=[x.getValue(d)],x._uniqueValuesCache[d];return x._uniqueValuesCache[d]=h.columnDef.getUniqueValues(x.original,x.index),x._uniqueValuesCache[d]}function Cx(x,d){return x.getValue(d)??x.table.options.renderFallbackValue}function Rx(x){return l0(x.subRows,(d)=>d.subRows)}function wx(x){return x.parentId?x.table.getRow(x.parentId,!0):void 0}function Fx(x){let d=[],h=x;while(!0){let i=h.getParentRow();if(!i)break;d.push(i),h=i}return d.reverse()}function $x(x){let d=x.table.getAllLeafColumns(),h=Array(d.length);for(let i=0;i<d.length;i++)h[i]=Vx(d[i],x,x.table);return h}function mx(x){let d={},h=x.getAllCells();for(let i=0;i<h.length;i++){let p=h[i];d[p.column.id]=p}return d}function _x(x,d,h,i){var p,r;return((p=(r=d.options).getRowId)===null||p===void 0?void 0:p.call(r,x,h,i))??`${i?[i.id,h].join("."):h}`}function ux(x,d,h){let i=(h?x.getPrePaginatedRowModel():x.getRowModel()).rowsById[d];if(!i){if(i=x.getCoreRowModel().rowsById[d],!i)throw Error()}return i}var ox={assignRowPrototype:(x,d)=>{H("coreRowsFeature",x,d,{row_getAllCellsByColumnId:{fn:(h)=>mx(h),memoDeps:(h)=>[h.getAllCells()]},row_getAllCells:{fn:(h)=>$x(h),memoDeps:(h)=>[h.table.getAllLeafColumns()]},row_getLeafRows:{fn:(h)=>Rx(h)},row_getParentRow:{fn:(h)=>wx(h)},row_getParentRows:{fn:(h)=>Fx(h)},row_getUniqueValues:{fn:(h,i)=>Ix(h,i)},row_getValue:{fn:(h,i)=>Sx(h,i)},row_renderValue:{fn:(h,i)=>Cx(h,i)}})},constructTableAPIs:(x)=>{A("coreRowsFeature",x,{table_getRowId:{fn:(d,h,i)=>_x(d,x,h,i)},table_getRow:{fn:(d,h)=>ux(x,d,h)}})}};function p0(x){let d=x.options.state;if(!d)return;x._reactivity.batch(()=>{for(let h in d){let i=x.baseAtoms[h];if(!i)continue;let p=d[h];if(p!==i.get())i.set(()=>p)}})}function bx(x){let d=D(x.initialState);x._reactivity.batch(()=>{let h=Object.keys(d);for(let i=0;i<h.length;i++){let p=h[i];x.baseAtoms[p].set(d[p])}})}function ph(x,d){if(x.options.mergeOptions)return x.options.mergeOptions(x.options,d);return{...x.options,...d}}function ax(x,d){let h=V(d,x.options),{features:i,atoms:p,initialState:r}=x.options,v=Object.assign(ph(x,h),{features:i,atoms:p,initialState:r});if(x.optionsStore)x.optionsStore.set(()=>v);else x.options=v;p0(x)}var tx={constructTableAPIs:(x)=>{A("coreTablesFeature",x,{table_reset:{fn:()=>bx(x)},table_setOptions:{fn:(d)=>ax(x,d)}})}};var ex={coreCellsFeature:px,coreColumnsFeature:nx,coreHeadersFeature:Bx,coreRowModelsFeature:Px,coreRowsFeature:ox,coreTablesFeature:tx};function r0(x){return x}function lx(x){let d=x;if(Object.defineProperty(x,"state",{get(){return x.get()}}),"set"in x)d.setState=x.set.bind(x);return d}function xd(x,d={}){return Object.values(x).forEach((h)=>{var i;d=((i=h.getInitialState)===null||i===void 0?void 0:i.call(h,d))??d}),D(d)}function v0(x){let d=x.features.coreReactivityFeature,{aggregationFns:h,columnMeta:i,coreRowModel:p,expandedRowModel:r,facetedMinMaxValues:v,facetedRowModel:k,facetedUniqueValues:N,filterFns:q,filterMeta:y,filteredRowModel:z,groupedRowModel:W,paginatedRowModel:Q,sortFns:T,sortedRowModel:n,tableMeta:g,...X}=x.features,E={_reactivity:d,_features:{...ex,...X},_rowModels:{},_rowModelFns:{aggregationFns:h,filterFns:q,sortFns:T},baseAtoms:{},atoms:{}},S=Object.values(E._features),$={...S.reduce((K,B)=>{var L;return Object.assign(K,(L=B.getDefaultTableOptions)===null||L===void 0?void 0:L.call(B,E))},{}),...x};if(d.wrapExternalAtoms&&$.atoms)for(let[K,B]of Object.entries($.atoms)){let L=B,I=d.createWritableAtom(L.get(),{debugName:`externalAtom/${K}`});$.atoms[K]=I;let V1=!1,jd=L.subscribe((S1)=>{if(V1)return;I.set(S1)}),Ad=I.subscribe((S1)=>{V1=!0,L.set(S1),V1=!1});d.addSubscription(jd),d.addSubscription(Ad)}if(d.createOptionsStore)E.optionsStore=d.createWritableAtom($,{debugName:"table/optionsStore"}),Object.defineProperty(E,"options",{configurable:!0,enumerable:!0,get(){return E.optionsStore.get()},set(K){E.optionsStore.set(()=>K)}});else E.options=$;E.initialState=xd(E._features,E.options.initialState);let T1=Object.keys(E.initialState);for(let K=0;K<T1.length;K++){let B=T1[K];E.baseAtoms[B]=d.createWritableAtom(E.initialState[B],{debugName:`table/baseAtoms/${B}`}),E.atoms[B]=d.createReadonlyAtom(()=>{let L=E.options.atoms,I=L===null||L===void 0?void 0:L[B];if(I)return I.get();return E.baseAtoms[B].get()},{debugName:`table/atoms/${B}`})}p0(E),E.store=lx(d.createReadonlyAtom(()=>{let K={};for(let B=0;B<T1.length;B++){let L=T1[B];K[L]=E.atoms[L].get()}return K},{debugName:"table/store"}));for(let K=0;K<S.length;K++){var P1,J0;(P1=(J0=S[K]).constructTableAPIs)===null||P1===void 0||P1.call(J0,E)}return E}var y0=Object.assign((x,d,h)=>{return x.getValue(d)===h},{autoRemove:(x)=>M(x)}),dd=Object.assign((x,d,h)=>{return x.getValue(d)==h},{autoRemove:(x)=>M(x)}),hd=Object.assign((x,d,h)=>{var i;return Boolean((i=x.getValue(d))===null||i===void 0?void 0:i.toString().includes(String(h)))},{autoRemove:(x)=>M(x)}),c1=Object.assign((x,d,h)=>{var i;return Boolean((i=x.getValue(d))===null||i===void 0?void 0:i.toString().toLowerCase().includes(String(h).toLowerCase()))},{autoRemove:(x)=>M(x)}),id=Object.assign((x,d,h)=>{var i;return((i=x.getValue(d))===null||i===void 0?void 0:i.toString().toLowerCase())===String(h).toLowerCase()},{autoRemove:(x)=>M(x)}),rh=Object.assign((x,d,h)=>{var i;return((i=x.getValue(d))===null||i===void 0?void 0:i.toString())===String(h)},{autoRemove:(x)=>M(x)}),G1=Object.assign((x,d,h)=>{let i=x.getValue(d),p=i===null||i===void 0?0:+i,r=Number(h);if(!isNaN(r)&&!isNaN(p))return p>r;return(i??"").toString().toLowerCase().trim()>String(h).toLowerCase().trim()},{resolveFilterValue:(x)=>M(x)}),k0=Object.assign((x,d,h)=>{return G1(x,d,h)||y0(x,d,h)},{resolveFilterValue:(x)=>M(x)}),pd=Object.assign((x,d,h)=>{return!k0(x,d,h)},{resolveFilterValue:(x)=>M(x)}),rd=Object.assign((x,d,h)=>{return!G1(x,d,h)},{resolveFilterValue:(x)=>M(x)}),vh=Object.assign((x,d,h)=>(["",void 0].includes(h[0])||G1(x,d,h[0]))&&(!isNaN(Number(h[0]))&&!isNaN(Number(h[1]))&&Number(h[0])>Number(h[1])||["",void 0].includes(h[1])||pd(x,d,h[1])),{autoRemove:(x)=>!x}),yh=Object.assign((x,d,h)=>(["",void 0].includes(h[0])||k0(x,d,h[0]))&&(!isNaN(Number(h[0]))&&!isNaN(Number(h[1]))&&Number(h[0])>Number(h[1])||["",void 0].includes(h[1])||rd(x,d,h[1])),{autoRemove:(x)=>!x}),vd=Object.assign((x,d,h)=>{let[i,p]=h,r=x.getValue(d);return r>=i&&r<=p},{resolveFilterValue:(x)=>{let[d,h]=x,i=typeof d!=="number"?parseFloat(d):d,p=typeof h!=="number"?parseFloat(h):h,r=d===null||Number.isNaN(i)?-1/0:i,v=h===null||Number.isNaN(p)?1/0:p;if(r>v){let k=r;r=v,v=k}return[r,v]},autoRemove:(x)=>M(x)||M(x[0])&&M(x[1])}),yd=(x,d,h)=>{return h.some((i)=>x.getValue(d)===i)},kd=Object.assign((x,d,h)=>{return h.some((i)=>x.getValue(d).includes(i))},{autoRemove:(x)=>M(x)||!(x===null||x===void 0?void 0:x.length)}),zd=Object.assign((x,d,h)=>{let i=x.getValue(d);if(!Array.isArray(i))return!1;return!h.some((p)=>!i.includes(p))},{autoRemove:(x)=>M(x)||!(x===null||x===void 0?void 0:x.length)}),qd=Object.assign((x,d,h)=>{let i=x.getValue(d);if(!Array.isArray(i))return!1;return h.some((p)=>i.includes(p))},{autoRemove:(x)=>M(x)||!(x===null||x===void 0?void 0:x.length)}),z0={arrIncludes:kd,arrIncludesAll:zd,arrHas:yd,arrIncludesSome:qd,between:vh,betweenInclusive:yh,equals:y0,equalsString:id,inNumberRange:vd,includesString:c1,includesStringSensitive:hd,weakEquals:dd};function M(x){return x===void 0||x===null||x===""}function Nd(){return[]}function q0(x){let d=x.table._rowModelFns.filterFns,h=x.table.getCoreRowModel().flatRows[0],i=h?h.getValue(x.id):void 0;if(typeof i==="string")return d===null||d===void 0?void 0:d.includesString;if(typeof i==="number")return d===null||d===void 0?void 0:d.inNumberRange;if(typeof i==="boolean")return d===null||d===void 0?void 0:d.equals;if(i!==null&&typeof i==="object")return d===null||d===void 0?void 0:d.equals;if(Array.isArray(i))return d===null||d===void 0?void 0:d.arrIncludes;return d===null||d===void 0?void 0:d.weakEquals}function x1(x){let d=null,h=x.table._rowModelFns.filterFns;return d=L1(x.columnDef.filterFn)?x.columnDef.filterFn:x.columnDef.filterFn==="auto"?q0(x):h===null||h===void 0?void 0:h[x.columnDef.filterFn],d??void 0}function Qd(x){return(x.columnDef.enableColumnFilter??!0)&&(x.table.options.enableColumnFilters??!0)&&(x.table.options.enableFilters??!0)&&!!x.accessorFn}function Td(x){return N0(x)>-1}function Wd(x){var d;return(d=x.table.atoms.columnFilters)===null||d===void 0||(d=d.get())===null||d===void 0||(d=d.find((h)=>h.id===x.id))===null||d===void 0?void 0:d.value}function N0(x){var d;return((d=x.table.atoms.columnFilters)===null||d===void 0||(d=d.get())===null||d===void 0?void 0:d.findIndex((h)=>h.id===x.id))??-1}function Yd(x,d){D1(x.table,(h)=>{let i=x1(x),p=h.find((k)=>k.id===x.id),r=V(d,p?p.value:void 0);if(Ed(i,r,x))return h.filter((k)=>k.id!==x.id);let v={id:x.id,value:r};if(p)return h.map((k)=>{if(k.id===x.id)return v;return k});if(h.length)return[...h,v];return[v]})}function D1(x,d){var h,i;let p=x.getAllLeafColumnsById(),r=(v)=>{return V(d,v).filter((k)=>{let N=p[k.id];if(N){if(Ed(x1(N),k.value,N))return!1}return!0})};(h=(i=x.options).onColumnFiltersChange)===null||h===void 0||h.call(i,r)}function nd(x,d){D1(x,d?[]:D(x.initialState.columnFilters??[]))}function Ed(x,d,h){return(x&&x.autoRemove?x.autoRemove(d,h):!1)||typeof d>"u"||typeof d==="string"&&!d}var Q0={getInitialState:(x)=>{return{columnFilters:Nd(),...x}},getDefaultColumnDef:()=>{return{filterFn:"auto"}},getDefaultTableOptions:(x)=>{return{onColumnFiltersChange:M1("columnFilters",x),filterFromLeafRows:!1,maxLeafRowFilterDepth:100}},assignColumnPrototype:(x,d)=>{H("columnFilteringFeature",x,d,{column_getAutoFilterFn:{fn:(h)=>q0(h)},column_getFilterFn:{fn:(h)=>x1(h)},column_getCanFilter:{fn:(h)=>Qd(h)},column_getIsFiltered:{fn:(h)=>Td(h)},column_getFilterValue:{fn:(h)=>Wd(h)},column_getFilterIndex:{fn:(h)=>N0(h)},column_setFilterValue:{fn:(h,i)=>Yd(h,i)}})},initRowInstanceData:(x)=>{x.columnFilters={},x.columnFiltersMeta={}},constructTableAPIs:(x)=>{A("columnFilteringFeature",x,{table_setColumnFilters:{fn:(d)=>D1(x,d)},table_resetColumnFilters:{fn:(d)=>nd(x,d)}})}};function H1(x){var d,h;return(x.columnDef.enableGlobalFilter??!0)&&(x.table.options.enableGlobalFilter??!0)&&(x.table.options.enableFilters??!0)&&(((d=(h=x.table.options).getColumnCanGlobalFilter)===null||d===void 0?void 0:d.call(h,x))??!0)&&!!x.accessorFn}function T0(){return c1}function O1(x){let{globalFilterFn:d}=x.options,h=x._rowModelFns.filterFns;return L1(d)?d:d==="auto"?T0():h===null||h===void 0?void 0:h[d]}function W0(x,d){var h,i;(h=(i=x.options).onGlobalFilterChange)===null||h===void 0||h.call(i,d)}function Jd(x,d){W0(x,d?void 0:D(x.initialState.globalFilter))}var Y0={getInitialState:(x)=>{return{globalFilter:void 0,...x}},getDefaultTableOptions:(x)=>{return{onGlobalFilterChange:M1("globalFilter",x),globalFilterFn:"auto",getColumnCanGlobalFilter:(d)=>{var h;let i=(h=x.getCoreRowModel().flatRows[0])===null||h===void 0||(h=h.getAllCellsByColumnId()[d.id])===null||h===void 0?void 0:h.getValue();return typeof i==="string"||typeof i==="number"}}},assignColumnPrototype:(x,d)=>{H("globalFilteringFeature",x,d,{column_getCanGlobalFilter:{fn:(h)=>H1(h)}})},constructTableAPIs:(x)=>{A("globalFilteringFeature",x,{table_getGlobalAutoFilterFn:{fn:()=>T0()},table_getGlobalFilterFn:{fn:()=>O1(x)},table_setGlobalFilter:{fn:(d)=>W0(x,d)},table_resetGlobalFilter:{fn:(d)=>Jd(x,d)}})}};function gd(x,d,h){if(h.options.filterFromLeafRows)return kh(x,d,h);return zh(x,d,h)}function kh(x,d,h){let i=[],p={},r=h.options.maxLeafRowFilterDepth??100,v=(k,N=0)=>{let q=[];for(let y of k){let z=q1(h,y.id,y.original,y.index,y.depth,void 0,y.parentId);if(z.columnFilters=y.columnFilters,y.subRows.length&&N<r){if(z.subRows=v(y.subRows,N+1),y=z,d(y)&&!z.subRows.length){q.push(y),p[y.id]=y,i.push(y);continue}if(d(y)||z.subRows.length){q.push(y),p[y.id]=y,i.push(y);continue}}else if(y=z,d(y))q.push(y),p[y.id]=y,i.push(y)}return q};return{rows:v(x),flatRows:i,rowsById:p}}function zh(x,d,h){let i=[],p={},r=h.options.maxLeafRowFilterDepth??100,v=(k,N=0)=>{let q=[];for(let y of k)if(d(y)){if(y.subRows.length&&N<r){let z=q1(h,y.id,y.original,y.index,y.depth,void 0,y.parentId);z.subRows=v(y.subRows,N+1),y=z}q.push(y),i.push(y),p[y.id]=y}return q};return{rows:v(x),flatRows:i,rowsById:p}}function n0(){return(x)=>{let d=x;return l({feature:"columnFilteringFeature",table:d,fnName:"table.getFilteredRowModel",memoDeps:()=>{var h,i;return[d.getPreFilteredRowModel(),(h=d.atoms.columnFilters)===null||h===void 0?void 0:h.get(),(i=d.atoms.globalFilter)===null||i===void 0?void 0:i.get()]},fn:()=>qh(d),onAfterUpdate:()=>A1(d)})}}function qh(x){var d,h;let i=x.getPreFilteredRowModel(),p=(d=x.atoms.columnFilters)===null||d===void 0?void 0:d.get(),r=(h=x.atoms.globalFilter)===null||h===void 0?void 0:h.get();if(!i.rows.length||!(p===null||p===void 0?void 0:p.length)&&!r){let Q=i.flatRows;for(let T=0;T<Q.length;T++){let n=Q[T];n.columnFilters={},n.columnFiltersMeta={}}return i}let v=[],k=[];p===null||p===void 0||p.forEach((Q)=>{var T;let n=j1(x,Q.id);if(!n)return;let g=x1(n);v.push({id:Q.id,filterFn:g,resolvedValue:((T=g.resolveFilterValue)===null||T===void 0?void 0:T.call(g,Q.value))??Q.value})});let N=(p===null||p===void 0?void 0:p.map((Q)=>Q.id))??[],q=O1(x),y=x.getAllLeafColumns().filter((Q)=>H1(Q));if(r&&q&&y.length)N.push("__global__"),y.forEach((Q)=>{var T;k.push({id:Q.id,filterFn:q,resolvedValue:((T=q.resolveFilterValue)===null||T===void 0?void 0:T.call(q,r))??r})});let z=i.flatRows;for(let Q=0;Q<z.length;Q++){let T=z[Q];if(T.columnFilters={},v.length)for(let n=0;n<v.length;n++){let g=v[n],X=g.id;T.columnFilters[X]=g.filterFn(T,X,g.resolvedValue,(E)=>{!T.columnFiltersMeta?T.columnFiltersMeta={}:T.columnFiltersMeta[X]=E})}if(k.length){for(let n=0;n<k.length;n++){let g=k[n],X=g.id;if(g.filterFn(T,X,g.resolvedValue,(E)=>{!T.columnFiltersMeta?T.columnFiltersMeta={}:T.columnFiltersMeta[X]=E})){T.columnFilters.__global__=!0;break}}if(T.columnFilters.__global__!==!0)T.columnFilters.__global__=!1}}let W=(Q)=>{for(let T=0;T<N.length;T++)if(Q.columnFilters[N[T]]===!1)return!1;return!0};return gd(i.rows,W,x)}var E0=class{constructor(x){this._table=null,this._notifier=0,(this.host=x).addController(this)}table(x,d){if(!this._table){let p={...x,features:{coreReactivityFeature:e0(),...x.features},mergeOptions:(r,v)=>{return{...r,...v}}};this._table=v0(p),this._setupSubscriptions()}this._table.setOptions((p)=>({...p,...x}));let h=this._table,i=function(r){let v=(r.source??h.store).get(),k=r.selector!==void 0?r.selector(v):v;if(typeof r.children==="function")return r.children(k);return r.children};return{...this._table,Subscribe:i,FlexRender:_0,get state(){return(d===null||d===void 0?void 0:d(h.store.state))??h.store.state}}}_setupSubscriptions(){if(this._table&&!this._storeSubscription)this._storeSubscription=this._table.store.subscribe(()=>{this._notifier++,this.host.requestUpdate()}),this._optionsSubscription=this._table.optionsStore.subscribe(()=>{this._notifier++,this.host.requestUpdate()})}hostConnected(){this._setupSubscriptions()}hostDisconnected(){var x,d;(x=this._storeSubscription)===null||x===void 0||x.unsubscribe(),this._storeSubscription=void 0,(d=this._optionsSubscription)===null||d===void 0||d.unsubscribe(),this._optionsSubscription=void 0}};var Nh={sampleCount:0,cpuAverage:0,cpuPeak:0,memoryAverage:0,memoryPeak:0,memoryBytesPeak:0};function Zd(x,d){let h=new Map;for(let i of x)h.set(Xd(i.kind,i.id),{...i,resources:{...i.resources},queuePosition:null,orderGroup:2});for(let i of d){let p=Xd(i.kind,i.id),r=h.get(p);h.set(p,{...r??Qh(i),status:i.status==="active"?"running":i.status,startedAt:i.leasedAt??r?.startedAt,queuePosition:i.position,orderGroup:i.status==="active"?0:1})}return[...h.values()].sort(Th)}function Qh(x){return{kind:x.kind,id:x.id,project:x.project,projectName:x.projectName,status:x.status,command:"",image:"",createdAt:x.acceptedAt,startedAt:x.leasedAt,finishedAt:void 0,exitCode:void 0,queueWaitMillis:void 0,resources:{...Nh}}}function Th(x,d){if(x.orderGroup!==d.orderGroup)return x.orderGroup-d.orderGroup;if(x.orderGroup<2)return(x.queuePosition??0)-(d.queuePosition??0);return Date.parse(d.createdAt)-Date.parse(x.createdAt)||d.id.localeCompare(x.id)}function Xd(x,d){return`${x}:${d}`}var Kd=d1`
  :host {
    --surface: #141417;
    --surface-hover: #1e1e23;
    --canvas-soft: #101013;
    --line: rgba(255,255,255,.09);
    --line-strong: rgba(255,255,255,.15);
    --text: #f5f3ed;
    --text-soft: #aaa7af;
    --text-faint: #74727a;
    --neutral-soft: rgba(170,167,175,.09);
    --ember: #e38242;
    --blue: #7bb8f0;
    --blue-soft: rgba(123,184,240,.11);
    --green: #70d6a2;
    --green-soft: rgba(112,214,162,.11);
    --red: #f08282;
    --red-soft: rgba(240,130,130,.11);
    --yellow: #e7c66d;
    --yellow-soft: rgba(231,198,109,.11);
    --mono: "SFMono-Regular", "Cascadia Code", "Roboto Mono", Consolas, monospace;
    display: block;
    color: var(--text);
  }
  * { box-sizing: border-box; }
  .runs-panel { overflow: hidden; border: 1px solid var(--line); border-radius: 11px; background: linear-gradient(180deg,rgba(255,255,255,.018),transparent),var(--surface); box-shadow: 0 16px 42px rgba(0,0,0,.12); }
  .runs-head { display: flex; min-height: 66px; align-items: center; justify-content: space-between; gap: 20px; padding: 10px 14px 10px 18px; border-bottom: 1px solid var(--line); }
  .runs-head > div:first-child { display: flex; align-items: baseline; gap: 10px; white-space: nowrap; }
  .runs-head strong { font-size: 12px; font-weight: 620; }
  .runs-head span { color: var(--text-faint); font: 10px var(--mono); }
  .runs-tools { display: flex; justify-content: flex-end; gap: 7px; width: min(100%, 620px); }
  label { min-width: 0; }
  .search { flex: 1 1 240px; }
  input, select { width: 100%; height: 34px; border: 1px solid var(--line); border-radius: 7px; outline: none; background: #0e0e11; color: var(--text-soft); font: inherit; font-size: 11px; }
  input { padding: 0 11px; }
  select { min-width: 128px; padding: 0 28px 0 10px; }
  input::placeholder { color: var(--text-faint); }
  input:focus, select:focus { border-color: rgba(227,130,66,.5); box-shadow: 0 0 0 3px rgba(227,130,66,.09); }
  .table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th { height: 38px; padding: 0 16px; color: var(--text-faint); font-size: 9px; font-weight: 700; letter-spacing: .08em; text-align: left; text-transform: uppercase; }
  td { height: 56px; padding: 8px 16px; border-top: 1px solid var(--line); color: var(--text-soft); font-size: 11px; }
  tbody tr { transition: background 120ms ease; }
  tbody tr:hover { background: rgba(255,255,255,.018); }
  td.primary { min-width: 220px; color: var(--text); }
  td.primary a { display: flex; align-items: center; gap: 9px; color: inherit; text-decoration: none; }
  td.primary a:hover { color: var(--ember); }
  .kind-icon { display: grid; flex: 0 0 auto; width: 27px; height: 27px; place-items: center; border: 1px solid var(--line); border-radius: 6px; background: var(--canvas-soft); color: var(--text-faint); font: 10px var(--mono); }
  .mono { font-family: var(--mono); font-size: 10px; }
  .muted { color: var(--text-faint); }
  .badge { display: inline-flex; align-items: center; gap: 6px; padding: 3px 7px; border: 1px solid var(--line); border-radius: 999px; color: var(--text-soft); font-size: 10px; font-weight: 560; white-space: nowrap; }
  .badge::before { content: ""; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
  .badge.running, .badge.active { border-color: rgba(123,184,240,.2); background: var(--blue-soft); color: var(--blue); }
  .badge.succeeded, .badge.success, .badge.online { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.running::before { width: 8px; height: 8px; border: 1.5px solid currentColor; border-right-color: transparent; background: transparent; animation: status-spin .75s linear infinite; }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.timed_out, .badge.lost { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
  .badge.cancelled, .badge.canceled { border-color: var(--line); background: var(--neutral-soft); color: var(--text-soft); }
  .position { display: inline-grid; min-width: 20px; height: 20px; margin-left: 6px; place-items: center; border: 1px solid var(--line); border-radius: 5px; }
  .empty { display: grid; min-height: 180px; place-content: center; padding: 32px; text-align: center; }
  .empty strong { font-size: 12px; font-weight: 580; }
  .empty span { display: block; margin-top: 4px; color: var(--text-faint); font-size: 10px; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
  @keyframes status-spin { to { transform: rotate(360deg); } }
  @media (max-width: 900px) {
    .runs-head { align-items: stretch; flex-direction: column; padding: 14px 16px; }
    .runs-tools { width: 100%; justify-content: stretch; }
    .search { flex-basis: auto; }
    th:nth-child(3), td:nth-child(3), th:nth-child(5), td:nth-child(5), th:nth-child(6), td:nth-child(6) { display: none; }
  }
  @media (max-width: 620px) {
    .runs-tools { flex-wrap: wrap; }
    .search { flex: 1 0 100%; }
    label:not(.search) { flex: 1; }
    select { min-width: 0; }
    th:nth-child(7), td:nth-child(7) { display: none; }
  }
  @media (prefers-reduced-motion: reduce) {
    .badge.running::before { border-right-color: currentColor; animation: none; }
  }
`;var Wh=r0({columnFilteringFeature:Q0,globalFilteringFeature:Y0,filteredRowModel:n0(),filterFns:z0}),Yh=[{id:"search",accessorFn:(x)=>[x.id,x.command,x.projectName,x.project,x.status,x.kind].join(" "),enableGlobalFilter:!0},{id:"status",accessorFn:(x)=>x.status,filterFn:"equalsString",enableGlobalFilter:!1},{id:"kind",accessorFn:(x)=>x.kind,filterFn:"equalsString",enableGlobalFilter:!1}],nh={now:""};class Bd extends g1(P){static styles=Kd;tableController=new E0(this);query="";statusFilter="";kindFilter="";rowsFingerprint="";rowsCache=[];render(){let x=this.signal("operations",[]),d=this.signal("queue",[]),h=this.signal("clock",nh),i=this.rows(x,d),p=[...this.statusFilter?[{id:"status",value:this.statusFilter}]:[],...this.kindFilter?[{id:"kind",value:this.kindFilter}]:[]],v=this.tableController.table({features:Wh,columns:Yh,data:i,getCoreRowModel:N1(),globalFilterFn:"includesString",getColumnCanGlobalFilter:(N)=>N.id==="search",state:{globalFilter:this.query,columnFilters:p}}).getRowModel().rows.map((N)=>N.original),k=Date.parse(h.now);return J`
      <article class="runs-panel">
        <header class="runs-head">
          <div><strong>Runs</strong><span>${v.length===i.length?`${i.length} total`:`${v.length} of ${i.length}`}</span></div>
          <div class="runs-tools">
            <label class="search"><span class="sr-only">Search runs</span><input type="search" placeholder="Search runs…" .value=${this.query} @input=${this.onSearch}></label>
            <label><span class="sr-only">Filter by status</span><select .value=${this.statusFilter} @change=${this.onStatusFilter}>
              <option value="">All statuses</option><option value="running">Running</option><option value="queued">Queued</option><option value="succeeded">Succeeded</option><option value="failed">Failed</option><option value="cancelled">Cancelled</option>
            </select></label>
            <label><span class="sr-only">Filter by kind</span><select .value=${this.kindFilter} @change=${this.onKindFilter}>
              <option value="">Jobs and builds</option><option value="job">Jobs</option><option value="build">Builds</option>
            </select></label>
          </div>
        </header>
        ${v.length===0?this.empty(i.length>0):J`
          <div class="table-wrap"><table>
            <thead><tr><th>Run</th><th>Status</th><th>Project</th><th>Duration</th><th>CPU peak</th><th>Memory peak</th><th>Created</th></tr></thead>
            <tbody>${v.map((N)=>this.row(N,k))}</tbody>
          </table></div>
        `}
      </article>
    `}rows(x,d){let h=JSON.stringify([x,d]);if(h!==this.rowsFingerprint)this.rowsFingerprint=h,this.rowsCache=Zd(x,d);return this.rowsCache}row(x,d){return J`<tr>
      <td class="primary"><a href=${Eh(x.kind,x.id)}><span class="kind-icon">${x.kind==="build"?"◇":"›_"}</span><span><span class="mono">${w(x.id,22)}</span><br><span class="muted">${x.command||Jh(x.kind)}</span></span></a></td>
      <td><span class="badge ${x.status}">${x.status}</span>${x.queuePosition!=null?J`<span class="position">${x.queuePosition}</span>`:""}</td>
      <td>${x.projectName}</td>
      <td class="mono">${X1(x.startedAt,x.finishedAt,d)}</td>
      <td class="mono">${x.resources.sampleCount?j(x.resources.cpuPeak):"—"}</td>
      <td class="mono">${x.resources.sampleCount?b(x.resources.memoryBytesPeak):"—"}</td>
      <td>${z1(x.createdAt,d)}</td>
    </tr>`}empty(x){return J`<div class="empty"><strong>${x?"No matching runs":"No runs yet"}</strong><span>${x?"Try a different search or filter.":"Submit a repository command with autback exec."}</span></div>`}onSearch=(x)=>{this.query=x.currentTarget.value,this.requestUpdate()};onStatusFilter=(x)=>{this.statusFilter=x.currentTarget.value,this.requestUpdate()};onKindFilter=(x)=>{this.kindFilter=x.currentTarget.value,this.requestUpdate()}}function Eh(x,d){return`/app/runs/${encodeURIComponent(x)}/${encodeURIComponent(d)}`}function Jh(x){return x?x[0].toUpperCase()+x.slice(1):"—"}customElements.define("autback-runs-table",Bd);var gh={samples:[],sampleCount:0,activeSampleCount:0,cpuCores:0,memoryTotalBytes:0,diskUsageBytes:0,diskTotalBytes:0,busyRatio:0,cpuAverage:0,cpuPeak:0,memoryAverage:0,memoryPeak:0,memoryBytesPeak:0,queueWaitP95Millis:0,diskInodesUsed:0,diskInodesTotal:0,cpuPressurePeak:0,memoryPressurePeak:0,ioPressurePeak:0,memoryHighEvents:0,oomEvents:0,oomKills:0,pidsCurrent:0,pidsLimit:0,unattributedPressureSamples:0},f={session:{user:"",admin:!1,projects:[]},service:{name:"Autback",version:"",control:"CLI only",admission:"One at a time",startedAt:""},worker:{status:"connecting",capacity:"1 operation",activeId:"",updatedAt:""},clock:{now:""},resources:gh,queue:[],operations:[],operation:null,log:{available:!1,truncated:!1,content:""},audit:[],status:{ready:!1,route:"",message:"Connecting",updatedAt:""}};class Ld extends g1(P){static styles=$0;get routeKind(){return this.getAttribute("route-kind")||"overview"}get project(){return this.getAttribute("project")||""}get operationID(){return this.getAttribute("operation-id")||""}get humanAuth(){return this.getAttribute("human-auth")==="true"}render(){let x=this.signals();return J`<div class="shell">
      ${this.sidebar(x)}
      <section class="workspace">
        ${this.topbar(x)}
        ${x.status.ready?J`<main class="content" id="content">${this.page(x)}</main>`:J`<main class="loading" id="content"><div class="loader">Opening console</div></main>`}
      </section>
    </div>`}signals(){return{session:this.signal("session",f.session),service:this.signal("service",f.service),worker:this.signal("worker",f.worker),resources:this.signal("resources",f.resources),clock:this.signal("clock",f.clock),queue:this.signal("queue",f.queue),operations:this.signal("operations",f.operations),operation:this.signal("operation",f.operation),log:this.signal("log",f.log),audit:this.signal("audit",f.audit),status:this.signal("status",f.status)}}sidebar(x){return J`<aside class="sidebar" aria-label="Console navigation">
      <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
      <nav class="nav-section" aria-label="Primary">
        <div class="nav-label">Console</div>
        ${this.navLink("/app","overview","Runs","activity")}
        ${this.navLink("/app/audit","audit","Audit log","shield")}
      </nav>
      <nav class="nav-section projects-nav" aria-label="Projects">
        <div class="nav-label">Projects</div>
        ${x.session.projects.map((d)=>J`<a class="nav-link ${this.routeKind==="project"&&this.project===d.slug?"active":""}" href=${`/app/projects/${encodeURIComponent(d.slug)}`}>
          ${G("cube")}<span>${d.name}</span><span class="count">${d.trusts}</span>
        </a>`)}
      </nav>
      <div class="sidebar-foot"><div class="identity"><span class="avatar">${Bh(x.session.user)}</span><div>
        <div class="identity-name">${x.session.user||"Connecting"}</div><div class="identity-role">${x.session.admin?"Administrator":"Member"}</div>
      </div>${this.humanAuth?J`<a class="sign-out" href="/auth/logout">Sign out</a>`:Z}</div></div>
    </aside>`}navLink(x,d,h,i){return J`<a class="nav-link ${this.routeKind===d?"active":""}" href=${x}>${G(i)}<span>${h}</span></a>`}topbar(x){let d=this.routeKind==="project"?this.project:this.routeKind==="operation"?w(this.operationID,18):this.routeKind==="audit"?"Audit log":"Runs";return J`<header class="topbar">
      <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${d}</strong></div>
      <div class="live ${x.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${x.status.message}</span></div>
    </header>`}page(x){let d=Date.parse(x.clock.now);switch(this.routeKind){case"project":return this.projectPage(x);case"operation":return this.runPage(x,d);case"audit":return this.auditPage(x,d);default:return this.overview(x)}}overview(x){return J`
      ${this.pageHead("Shared runner","Runs and capacity","See what is running, what is next, and whether the machine has room to do more.")}
      ${this.resourceChart(x.resources,"Runner utilization")}
      ${this.resourceMetrics(x.resources)}
      <autback-runs-table></autback-runs-table>
    `}projectPage(x){let d=x.session.projects.find((h)=>h.slug===this.project);if(!d)return this.notFound("You do not have access to this project.");return J`
      ${this.pageHead("Project",d.name,"Runs, demand, and runner use for this project.")}
      <section class="project-banner">
        <div><div class="project-name">${d.name}</div><div class="project-slug">${d.slug}</div><p class="digest">${u1(d.activeImage)}</p></div>
        <div class="project-facts">
          <div class="fact"><strong>${d.members}</strong><span>Members</span></div>
          <div class="fact"><strong>${d.trusts}</strong><span>GitHub trusts</span></div>
          <div class="fact"><strong>${d.allowImageOverrides?"Flexible":"Pinned"}</strong><span>Runner image</span></div>
        </div>
      </section>
      ${this.projectTrends(x.operations)}
      ${this.resourceChart(x.resources,"Resource utilization")}
      <autback-runs-table></autback-runs-table>
    `}runPage(x,d){let h=x.operation;if(!h)return this.notFound("You do not have access to this run.");let i=h.command||`${s1(h.kind)} ${w(h.id,18)}`;return J`
      ${this.pageHead(`${s1(h.kind)} run`,i,`${h.projectName} · ${w(h.id,26)}`)}
      ${this.resourceChart(x.resources,"Resource utilization")}
      <section class="metrics" aria-label="Run summary">
        ${O("Status",h.status,s1(h.kind),"pulse")}
        ${O("Queue wait",e(h.queueWaitMillis),"before starting","queue")}
        ${O("Duration",X1(h.startedAt,h.finishedAt,d),h.startedAt?"elapsed time":"not started","clock")}
        ${O("Exit code",h.exitCode==null?"—":String(h.exitCode),h.finishedAt?"result":"pending","terminal")}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel"><header class="panel-head"><div class="panel-title">${G("terminal")}Command</div><span class="badge ${h.status}">${h.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${h.command||"docker buildx build"}</pre></div>
          </article>
          ${this.logPanel(x,h)}
        </div>
        <div class="detail-stack">${this.runSummaryPanel(h,d)}${this.provenancePanel(h)}
          <article class="panel"><header class="panel-head"><div class="panel-title">${G("terminal")}Continue in CLI</div><span class="panel-meta">CLI</span></header>
            <div class="panel-body"><p class="lede">View the full log or inspect this run from your terminal.</p><pre class="command"><span class="prompt">$</span> autback ${h.kind==="job"?"logs":"build status"} ${h.id}</pre></div>
          </article>
        </div>
      </section>
    `}resourceMetrics(x){let d=x.unattributedPressureSamples?`${x.unattributedPressureSamples} host-only samples`:"attributed to active work";return J`<section class="metrics" aria-label="Runner capacity summary">
      ${O("Busy",j(x.busyRatio),"of the selected hour","pulse")}
      ${O("CPU while active",j(x.cpuAverage),`${j(x.cpuPeak)} peak`,"cpu")}
      ${O("Memory while active",j(x.memoryAverage),`${b(x.memoryBytesPeak)} peak`,"memory")}
      ${O("Queue wait p95",e(x.queueWaitP95Millis),"recent runs","queue")}
      ${O("Memory pressure",j(x.memoryPressurePeak),`${x.memoryHighEvents} high events`,"memory")}
      ${O("I/O pressure",j(x.ioPressurePeak),`${j(x.cpuPressurePeak)} CPU pressure`,"cpu")}
      ${O("OOM kills",String(x.oomKills),d,"terminal")}
      ${O("Processes",x.pidsLimit?`${x.pidsCurrent} / ${x.pidsLimit}`:String(x.pidsCurrent),"host cgroup","pulse")}
    </section>`}resourceChart(x,d){let h=Ud(x.samples,(v)=>v.cpuUtilization),i=Ud(x.samples,(v)=>v.memoryUtilization),p=x.samples.at(0),r=x.samples.at(-1);return J`<article class="panel resource-panel">
      <header class="panel-head"><div class="panel-title">${G("activity")}${d}</div>
        <span class="panel-meta">${Xh(x)}</span></header>
      ${x.samples.length<2?Q1("activity","Collecting runner data","Utilization will appear after the next samples arrive."):J`
        <div class="chart-legend">
          <span class="legend cpu"><i></i>CPU <strong>${j(x.cpuAverage)} avg · ${j(x.cpuPeak)} peak</strong></span>
          <span class="legend memory"><i></i>Memory <strong>${j(x.memoryAverage)} avg · ${j(x.memoryPeak)} peak</strong></span>
        </div>
        <div class="resource-chart">
          <svg viewBox="0 0 900 230" preserveAspectRatio="none" role="img" aria-label="CPU and memory utilization over time">
            ${[0,0.25,0.5,0.75,1].map((v)=>U`<line class="grid-line" x1="42" y1=${f1(v)} x2="892" y2=${f1(v)}></line><text class="axis-label" x="4" y=${f1(v)+4}>${Math.round(v*100)}%</text>`)}
            <polyline class="series memory" points=${i}></polyline>
            <polyline class="series cpu" points=${h}></polyline>
          </svg>
          <div class="chart-times"><span>${Md(p?.observedAt)}</span><span>${Md(r?.observedAt)}</span></div>
        </div>
      `}
    </article>`}projectTrends(x){let d=x.filter((p)=>p.startedAt&&p.finishedAt).slice(0,20).reverse(),h=d.map((p)=>Date.parse(p.finishedAt)-Date.parse(p.startedAt)),i=Math.max(...h,1);return J`<section class="trend-grid">
      <article class="panel trend-panel"><header class="panel-head"><div class="panel-title">${G("clock")}Run duration</div><span class="panel-meta">Last ${d.length}</span></header>
        <div class="duration-bars">${h.length===0?Q1("clock","No completed runs","Duration history will appear here."):h.map((p)=>J`<i style=${`height:${Math.max(5,p/i*100)}%`} title=${e(p)}></i>`)}</div>
      </article>
      <article class="panel project-health"><div><span>Success rate</span><strong>${m0(x.map((p)=>p.status))}</strong></div><div><span>Queue wait p95</span><strong>${e(Zh(x.map((p)=>p.queueWaitMillis)))}</strong></div></article>
    </section>`}runSummaryPanel(x,d){return J`<article class="panel"><header class="panel-head"><div class="panel-title">${G("activity")}Run summary</div><span class="panel-meta">${x.resources.sampleCount} samples</span></header>
      <dl class="definition"><dt>Started</dt><dd>${x.startedAt?z1(x.startedAt,d):"—"}</dd><dt>CPU peak</dt><dd>${j(x.resources.cpuPeak)}</dd><dt>Memory peak</dt><dd>${b(x.resources.memoryBytesPeak)}</dd><dt>Queue wait</dt><dd>${e(x.queueWaitMillis)}</dd></dl>
    </article>`}logPanel(x,d){let h=!d.finishedAt&&!["succeeded","failed","cancelled","timed_out","lost"].includes(d.status),i=J`Older lines remain available with <span class="mono">autback logs ${d.id}</span>.`;return J`<article class="panel"><header class="panel-head"><div class="panel-title">${G("terminal")}Output</div><span class="panel-meta">${x.log.available?h?"Following":"Complete":"Unavailable"}</span></header>
      ${x.log.available?J`<pre class="log">${x.log.content||"Waiting for output…"}</pre>${x.log.truncated?J`<div class="log-note">${h?J`Following live output. ${i}`:i}</div>`:Z}`:Q1("terminal","No output available",d.kind==="build"?"Build progress remains in the invoking terminal.":"The runner has not produced output yet.")}
    </article>`}provenancePanel(x){let d=x.caches?.length?x.caches.map((h)=>h.name).join(", "):"None declared";return J`<article class="panel"><header class="panel-head"><div class="panel-title">${G("fingerprint")}Provenance</div><span class="panel-meta">Inputs</span></header>
      <dl class="definition"><dt>Run</dt><dd>${x.id}</dd><dt>Project</dt><dd>${x.project}</dd><dt>Image</dt><dd title=${x.image}>${u1(x.image)}</dd><dt>Workdir</dt><dd>${x.workingDirectory||"—"}</dd><dt>Root</dt><dd>${x.rootDigest||"—"}</dd><dt>Caches</dt><dd>${d}</dd></dl>
    </article>`}auditPage(x,d){return J`${this.pageHead("Governance","Audit log","Project, access, image, job, and build activity across Autback.")}
      <article class="panel"><header class="panel-head"><div class="panel-title">${G("shield")}Recent events</div><span class="panel-meta">${x.audit.length} records</span></header>
      ${x.audit.length===0?Q1("shield","No audit events yet","Changes made with the Autback CLI will appear here."):this.auditTable(x.audit,d)}</article>`}auditTable(x,d){return J`<div class="table-wrap"><table><thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${x.map((h)=>J`<tr><td><span class="audit-action">${h.action}</span>${Kh(h)}</td><td>${h.actor}</td><td>${h.project||"Service"}</td><td class="mono">${w(h.target,18)}</td><td>${z1(h.createdAt,d)}</td></tr>`)}</tbody>
    </table></div>`}pageHead(x,d,h){return J`<header class="page-head"><div><p class="eyebrow">${x}</p><h1>${d}</h1><p class="lede">${h}</p></div><div class="read-only">${G("eye")}CLI-managed</div></header>`}notFound(x){return J`${this.pageHead("Not found","Unavailable",x)}<article class="panel">${Q1("shield","Nothing to show","Return to the console overview.")}</article>`}}function G(x){let d={activity:U`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`,clock:U`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,cpu:U`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,cube:U`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`,disk:U`<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/>`,eye:U`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,fingerprint:U`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,memory:U`<rect x="5" y="7" width="14" height="10" rx="2"/><path d="M8 3v4M12 3v4M16 3v4M8 17v4M12 17v4M16 17v4M9 11h6"/>`,pulse:U`<path d="M3 12h4l2-5 4 10 2-5h6"/>`,queue:U`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,shield:U`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`,terminal:U`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`,trend:U`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`};return U`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${d[x]}</svg>`}function O(x,d,h,i){return J`<article class="metric"><div class="metric-top"><span>${x}</span>${G(i)}</div><div class="metric-value">${s1(d)}</div><div class="metric-note">${h}</div></article>`}function Q1(x,d,h){return J`<div class="empty"><div>${G(x)}<strong>${d}</strong><span>${h}</span></div></div>`}function Xh(x){return x.cpuCores?`${x.cpuCores} vCPU · ${b(x.memoryTotalBytes)} · ${b(x.diskTotalBytes)} disk`:"Waiting for capacity data"}function f1(x){return 216-Math.max(0,Math.min(1,x))*196}function Ud(x,d){if(x.length===0)return"";return x.map((h,i)=>`${42+i/Math.max(1,x.length-1)*850},${f1(d(h))}`).join(" ")}function Md(x){if(!x)return"—";return new Intl.DateTimeFormat(void 0,{hour:"2-digit",minute:"2-digit"}).format(new Date(x))}function Zh(x){let d=x.filter((h)=>h!=null&&Number.isFinite(h)).sort((h,i)=>h-i);return d.length?d[Math.ceil(d.length*0.95)-1]:void 0}function Kh(x){let d=Object.entries(x.metadata??{}).slice(0,3);return d.length===0?Z:J`<div class="metadata">${d.map(([h,i])=>J`<span>${h}=${w(i,28)}</span>`)}</div>`}function s1(x){return x?x[0].toUpperCase()+x.slice(1):"—"}function Bh(x){return x.split(/\s+/).filter(Boolean).slice(0,2).map((d)=>d[0]?.toUpperCase()).join("")||"A"}customElements.define("autback-console",Ld);
