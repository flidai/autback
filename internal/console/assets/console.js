var W1=globalThis,Y1=W1.ShadowRoot&&(W1.ShadyCSS===void 0||W1.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,I1=Symbol(),Z0=new WeakMap;class J1{constructor(x,d,h){if(this._$cssResult$=!0,h!==I1)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=x,this.t=d}get styleSheet(){let x=this.o,d=this.t;if(Y1&&x===void 0){let h=d!==void 0&&d.length===1;h&&(x=Z0.get(d)),x===void 0&&((this.o=x=new CSSStyleSheet).replaceSync(this.cssText),h&&Z0.set(d,x))}return x}toString(){return this.cssText}}var K0=(x)=>new J1(typeof x=="string"?x:x+"",void 0,I1),d1=(x,...d)=>{let h=x.length===1?x[0]:d.reduce((p,i,v)=>p+((y)=>{if(y._$cssResult$===!0)return y.cssText;if(typeof y=="number")return y;throw Error("Value passed to 'css' function must be a 'css' function result: "+y+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(i)+x[v+1],x[0]);return new J1(h,x,I1)},B0=(x,d)=>{if(Y1)x.adoptedStyleSheets=d.map((h)=>h instanceof CSSStyleSheet?h:h.styleSheet);else for(let h of d){let p=document.createElement("style"),i=W1.litNonce;i!==void 0&&p.setAttribute("nonce",i),p.textContent=h.cssText,x.appendChild(p)}},C1=Y1?(x)=>x:(x)=>x instanceof CSSStyleSheet?((d)=>{let h="";for(let p of d.cssRules)h+=p.cssText;return K0(h)})(x):x;var{is:Dd,defineProperty:Hd,getOwnPropertyDescriptor:Od,getOwnPropertyNames:nd,getOwnPropertySymbols:cd,getPrototypeOf:Vd}=Object,E1=globalThis,U0=E1.trustedTypes,fd=U0?U0.emptyScript:"",Pd=E1.reactiveElementPolyfillSupport,h1=(x,d)=>x,R1={toAttribute(x,d){switch(d){case Boolean:x=x?fd:null;break;case Object:case Array:x=x==null?x:JSON.stringify(x)}return x},fromAttribute(x,d){let h=x;switch(d){case Boolean:h=x!==null;break;case Number:h=x===null?null:Number(x);break;case Object:case Array:try{h=JSON.parse(x)}catch(p){h=null}}return h}},M0=(x,d)=>!Dd(x,d),j0={attribute:!0,type:String,converter:R1,reflect:!1,useDefault:!1,hasChanged:M0};Symbol.metadata??=Symbol("metadata"),E1.litPropertyMetadata??=new WeakMap;class I extends HTMLElement{static addInitializer(x){this._$Ei(),(this.l??=[]).push(x)}static get observedAttributes(){return this.finalize(),this._$Eh&&[...this._$Eh.keys()]}static createProperty(x,d=j0){if(d.state&&(d.attribute=!1),this._$Ei(),this.prototype.hasOwnProperty(x)&&((d=Object.create(d)).wrapped=!0),this.elementProperties.set(x,d),!d.noAccessor){let h=Symbol(),p=this.getPropertyDescriptor(x,h,d);p!==void 0&&Hd(this.prototype,x,p)}}static getPropertyDescriptor(x,d,h){let{get:p,set:i}=Od(this.prototype,x)??{get(){return this[d]},set(v){this[d]=v}};return{get:p,set(v){let y=p?.call(this);i?.call(this,v),this.requestUpdate(x,y,h)},configurable:!0,enumerable:!0}}static getPropertyOptions(x){return this.elementProperties.get(x)??j0}static _$Ei(){if(this.hasOwnProperty(h1("elementProperties")))return;let x=Vd(this);x.finalize(),x.l!==void 0&&(this.l=[...x.l]),this.elementProperties=new Map(x.elementProperties)}static finalize(){if(this.hasOwnProperty(h1("finalized")))return;if(this.finalized=!0,this._$Ei(),this.hasOwnProperty(h1("properties"))){let d=this.properties,h=[...nd(d),...cd(d)];for(let p of h)this.createProperty(p,d[p])}let x=this[Symbol.metadata];if(x!==null){let d=litPropertyMetadata.get(x);if(d!==void 0)for(let[h,p]of d)this.elementProperties.set(h,p)}this._$Eh=new Map;for(let[d,h]of this.elementProperties){let p=this._$Eu(d,h);p!==void 0&&this._$Eh.set(p,d)}this.elementStyles=this.finalizeStyles(this.styles)}static finalizeStyles(x){let d=[];if(Array.isArray(x)){let h=new Set(x.flat(1/0).reverse());for(let p of h)d.unshift(C1(p))}else x!==void 0&&d.push(C1(x));return d}static _$Eu(x,d){let h=d.attribute;return h===!1?void 0:typeof h=="string"?h:typeof x=="string"?x.toLowerCase():void 0}constructor(){super(),this._$Ep=void 0,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Em=null,this._$Ev()}_$Ev(){this._$ES=new Promise((x)=>this.enableUpdating=x),this._$AL=new Map,this._$E_(),this.requestUpdate(),this.constructor.l?.forEach((x)=>x(this))}addController(x){(this._$EO??=new Set).add(x),this.renderRoot!==void 0&&this.isConnected&&x.hostConnected?.()}removeController(x){this._$EO?.delete(x)}_$E_(){let x=new Map,d=this.constructor.elementProperties;for(let h of d.keys())this.hasOwnProperty(h)&&(x.set(h,this[h]),delete this[h]);x.size>0&&(this._$Ep=x)}createRenderRoot(){let x=this.shadowRoot??this.attachShadow(this.constructor.shadowRootOptions);return B0(x,this.constructor.elementStyles),x}connectedCallback(){this.renderRoot??=this.createRenderRoot(),this.enableUpdating(!0),this._$EO?.forEach((x)=>x.hostConnected?.())}enableUpdating(x){}disconnectedCallback(){this._$EO?.forEach((x)=>x.hostDisconnected?.())}attributeChangedCallback(x,d,h){this._$AK(x,h)}_$ET(x,d){let h=this.constructor.elementProperties.get(x),p=this.constructor._$Eu(x,h);if(p!==void 0&&h.reflect===!0){let i=(h.converter?.toAttribute!==void 0?h.converter:R1).toAttribute(d,h.type);this._$Em=x,i==null?this.removeAttribute(p):this.setAttribute(p,i),this._$Em=null}}_$AK(x,d){let h=this.constructor,p=h._$Eh.get(x);if(p!==void 0&&this._$Em!==p){let i=h.getPropertyOptions(p),v=typeof i.converter=="function"?{fromAttribute:i.converter}:i.converter?.fromAttribute!==void 0?i.converter:R1;this._$Em=p;let y=v.fromAttribute(d,i.type);this[p]=y??this._$Ej?.get(p)??y,this._$Em=null}}requestUpdate(x,d,h,p=!1,i){if(x!==void 0){let v=this.constructor;if(p===!1&&(i=this[x]),h??=v.getPropertyOptions(x),!((h.hasChanged??M0)(i,d)||h.useDefault&&h.reflect&&i===this._$Ej?.get(x)&&!this.hasAttribute(v._$Eu(x,h))))return;this.C(x,d,h)}this.isUpdatePending===!1&&(this._$ES=this._$EP())}C(x,d,{useDefault:h,reflect:p,wrapped:i},v){h&&!(this._$Ej??=new Map).has(x)&&(this._$Ej.set(x,v??d??this[x]),i!==!0||v!==void 0)||(this._$AL.has(x)||(this.hasUpdated||h||(d=void 0),this._$AL.set(x,d)),p===!0&&this._$Em!==x&&(this._$Eq??=new Set).add(x))}async _$EP(){this.isUpdatePending=!0;try{await this._$ES}catch(d){Promise.reject(d)}let x=this.scheduleUpdate();return x!=null&&await x,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){if(!this.isUpdatePending)return;if(!this.hasUpdated){if(this.renderRoot??=this.createRenderRoot(),this._$Ep){for(let[p,i]of this._$Ep)this[p]=i;this._$Ep=void 0}let h=this.constructor.elementProperties;if(h.size>0)for(let[p,i]of h){let{wrapped:v}=i,y=this[p];v!==!0||this._$AL.has(p)||y===void 0||this.C(p,void 0,i,y)}}let x=!1,d=this._$AL;try{x=this.shouldUpdate(d),x?(this.willUpdate(d),this._$EO?.forEach((h)=>h.hostUpdate?.()),this.update(d)):this._$EM()}catch(h){throw x=!1,this._$EM(),h}x&&this._$AE(d)}willUpdate(x){}_$AE(x){this._$EO?.forEach((d)=>d.hostUpdated?.()),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(x)),this.updated(x)}_$EM(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$ES}shouldUpdate(x){return!0}update(x){this._$Eq&&=this._$Eq.forEach((d)=>this._$ET(d,this[d])),this._$EM()}updated(x){}firstUpdated(x){}}I.elementStyles=[],I.shadowRootOptions={mode:"open"},I[h1("elementProperties")]=new Map,I[h1("finalized")]=new Map,Pd?.({ReactiveElement:I}),(E1.reactiveElementVersions??=[]).push("2.1.2");var w1=globalThis,g0=(x)=>x,X1=w1.trustedTypes,L0=X1?X1.createPolicy("lit-html",{createHTML:(x)=>x}):void 0;var C=`lit$${Math.random().toFixed(9).slice(2)}$`,n0="?"+C,Sd=`<${n0}>`,u=document,i1=()=>u.createComment(""),v1=(x)=>x===null||typeof x!="object"&&typeof x!="function",F1=Array.isArray,sd=(x)=>F1(x)||typeof x?.[Symbol.iterator]=="function";var p1=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,A0=/-->/g,G0=/>/g,m=RegExp(`>|[ 	
\f\r](?:([^\\s"'>=/]+)([ 	
\f\r]*=[ 	
\f\r]*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`,"g"),D0=/'/g,H0=/"/g,c0=/^(?:script|style|textarea|title)$/i,$1=(x)=>(d,...h)=>({_$litType$:x,strings:d,values:h}),X=$1(1),M=$1(2),Gh=$1(3),b=Symbol.for("lit-noChange"),B=Symbol.for("lit-nothing"),O0=new WeakMap,_=u.createTreeWalker(u,129);function V0(x,d){if(!F1(x)||!x.hasOwnProperty("raw"))throw Error("invalid template strings array");return L0!==void 0?L0.createHTML(d):d}var Id=(x,d)=>{let h=x.length-1,p=[],i,v=d===2?"<svg>":d===3?"<math>":"",y=p1;for(let r=0;r<h;r++){let N=x[r],q,k,z=-1,W=0;for(;W<N.length&&(y.lastIndex=W,k=y.exec(N),k!==null);)W=y.lastIndex,y===p1?k[1]==="!--"?y=A0:k[1]!==void 0?y=G0:k[2]!==void 0?(c0.test(k[2])&&(i=RegExp("</"+k[2],"g")),y=m):k[3]!==void 0&&(y=m):y===m?k[0]===">"?(y=i??p1,z=-1):k[1]===void 0?z=-2:(z=y.lastIndex-k[2].length,q=k[1],y=k[3]===void 0?m:k[3]==='"'?H0:D0):y===H0||y===D0?y=m:y===A0||y===G0?y=p1:(y=m,i=void 0);let Q=y===m&&x[r+1].startsWith("/>")?" ":"";v+=y===p1?N+Sd:z>=0?(p.push(q),N.slice(0,z)+"$lit$"+N.slice(z)+C+Q):N+C+(z===-2?r:Q)}return[V0(x,v+(x[h]||"<?>")+(d===2?"</svg>":d===3?"</math>":"")),p]};class y1{constructor({strings:x,_$litType$:d},h){let p;this.parts=[];let i=0,v=0,y=x.length-1,r=this.parts,[N,q]=Id(x,d);if(this.el=y1.createElement(N,h),_.currentNode=this.el.content,d===2||d===3){let k=this.el.content.firstChild;k.replaceWith(...k.childNodes)}for(;(p=_.nextNode())!==null&&r.length<y;){if(p.nodeType===1){if(p.hasAttributes())for(let k of p.getAttributeNames())if(k.endsWith("$lit$")){let z=q[v++],W=p.getAttribute(k).split(C),Q=/([.?@])?(.*)/.exec(z);r.push({type:1,index:i,name:Q[2],strings:W,ctor:Q[1]==="."?P0:Q[1]==="?"?S0:Q[1]==="@"?s0:r1}),p.removeAttribute(k)}else k.startsWith(C)&&(r.push({type:6,index:i}),p.removeAttribute(k));if(c0.test(p.tagName)){let k=p.textContent.split(C),z=k.length-1;if(z>0){p.textContent=X1?X1.emptyScript:"";for(let W=0;W<z;W++)p.append(k[W],i1()),_.nextNode(),r.push({type:2,index:++i});p.append(k[z],i1())}}}else if(p.nodeType===8)if(p.data===n0)r.push({type:2,index:i});else{let k=-1;for(;(k=p.data.indexOf(C,k+1))!==-1;)r.push({type:7,index:i}),k+=C.length-1}i++}}static createElement(x,d){let h=u.createElement("template");return h.innerHTML=x,h}}function a(x,d,h=x,p){if(d===b)return d;let i=p!==void 0?h._$Co?.[p]:h._$Cl,v=v1(d)?void 0:d._$litDirective$;return i?.constructor!==v&&(i?._$AO?.(!1),v===void 0?i=void 0:(i=new v(x),i._$AT(x,h,p)),p!==void 0?(h._$Co??=[])[p]=i:h._$Cl=i),i!==void 0&&(d=a(x,i._$AS(x,d.values),i,p)),d}class f0{constructor(x,d){this._$AV=[],this._$AN=void 0,this._$AD=x,this._$AM=d}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}u(x){let{el:{content:d},parts:h}=this._$AD,p=(x?.creationScope??u).importNode(d,!0);_.currentNode=p;let i=_.nextNode(),v=0,y=0,r=h[0];for(;r!==void 0;){if(v===r.index){let N;r.type===2?N=new k1(i,i.nextSibling,this,x):r.type===1?N=new r.ctor(i,r.name,r.strings,this,x):r.type===6&&(N=new I0(i,this,x)),this._$AV.push(N),r=h[++y]}v!==r?.index&&(i=_.nextNode(),v++)}return _.currentNode=u,p}p(x){let d=0;for(let h of this._$AV)h!==void 0&&(h.strings!==void 0?(h._$AI(x,h,d),d+=h.strings.length-2):h._$AI(x[d])),d++}}class k1{get _$AU(){return this._$AM?._$AU??this._$Cv}constructor(x,d,h,p){this.type=2,this._$AH=B,this._$AN=void 0,this._$AA=x,this._$AB=d,this._$AM=h,this.options=p,this._$Cv=p?.isConnected??!0}get parentNode(){let x=this._$AA.parentNode,d=this._$AM;return d!==void 0&&x?.nodeType===11&&(x=d.parentNode),x}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(x,d=this){x=a(this,x,d),v1(x)?x===B||x==null||x===""?(this._$AH!==B&&this._$AR(),this._$AH=B):x!==this._$AH&&x!==b&&this._(x):x._$litType$!==void 0?this.$(x):x.nodeType!==void 0?this.T(x):sd(x)?this.k(x):this._(x)}O(x){return this._$AA.parentNode.insertBefore(x,this._$AB)}T(x){this._$AH!==x&&(this._$AR(),this._$AH=this.O(x))}_(x){this._$AH!==B&&v1(this._$AH)?this._$AA.nextSibling.data=x:this.T(u.createTextNode(x)),this._$AH=x}$(x){let{values:d,_$litType$:h}=x,p=typeof h=="number"?this._$AC(x):(h.el===void 0&&(h.el=y1.createElement(V0(h.h,h.h[0]),this.options)),h);if(this._$AH?._$AD===p)this._$AH.p(d);else{let i=new f0(p,this),v=i.u(this.options);i.p(d),this.T(v),this._$AH=i}}_$AC(x){let d=O0.get(x.strings);return d===void 0&&O0.set(x.strings,d=new y1(x)),d}k(x){F1(this._$AH)||(this._$AH=[],this._$AR());let d=this._$AH,h,p=0;for(let i of x)p===d.length?d.push(h=new k1(this.O(i1()),this.O(i1()),this,this.options)):h=d[p],h._$AI(i),p++;p<d.length&&(this._$AR(h&&h._$AB.nextSibling,p),d.length=p)}_$AR(x=this._$AA.nextSibling,d){for(this._$AP?.(!1,!0,d);x!==this._$AB;){let h=g0(x).nextSibling;g0(x).remove(),x=h}}setConnected(x){this._$AM===void 0&&(this._$Cv=x,this._$AP?.(x))}}class r1{get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}constructor(x,d,h,p,i){this.type=1,this._$AH=B,this._$AN=void 0,this.element=x,this.name=d,this._$AM=p,this.options=i,h.length>2||h[0]!==""||h[1]!==""?(this._$AH=Array(h.length-1).fill(new String),this.strings=h):this._$AH=B}_$AI(x,d=this,h,p){let i=this.strings,v=!1;if(i===void 0)x=a(this,x,d,0),v=!v1(x)||x!==this._$AH&&x!==b,v&&(this._$AH=x);else{let y=x,r,N;for(x=i[0],r=0;r<i.length-1;r++)N=a(this,y[h+r],d,r),N===b&&(N=this._$AH[r]),v||=!v1(N)||N!==this._$AH[r],N===B?x=B:x!==B&&(x+=(N??"")+i[r+1]),this._$AH[r]=N}v&&!p&&this.j(x)}j(x){x===B?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,x??"")}}class P0 extends r1{constructor(){super(...arguments),this.type=3}j(x){this.element[this.name]=x===B?void 0:x}}class S0 extends r1{constructor(){super(...arguments),this.type=4}j(x){this.element.toggleAttribute(this.name,!!x&&x!==B)}}class s0 extends r1{constructor(x,d,h,p,i){super(x,d,h,p,i),this.type=5}_$AI(x,d=this){if((x=a(this,x,d,0)??B)===b)return;let h=this._$AH,p=x===B&&h!==B||x.capture!==h.capture||x.once!==h.once||x.passive!==h.passive,i=x!==B&&(h===B||p);p&&this.element.removeEventListener(this.name,this,h),i&&this.element.addEventListener(this.name,this,x),this._$AH=x}handleEvent(x){typeof this._$AH=="function"?this._$AH.call(this.options?.host??this.element,x):this._$AH.handleEvent(x)}}class I0{constructor(x,d,h){this.element=x,this.type=6,this._$AN=void 0,this._$AM=d,this.options=h}get _$AU(){return this._$AM._$AU}_$AI(x){a(this,x)}}var Cd=w1.litHtmlPolyfillSupport;Cd?.(y1,k1),(w1.litHtmlVersions??=[]).push("3.3.3");var C0=(x,d,h)=>{let p=h?.renderBefore??d,i=p._$litPart$;if(i===void 0){let v=h?.renderBefore??null;p._$litPart$=i=new k1(d.insertBefore(i1(),v),v,void 0,h??{})}return i._$AI(x),i};var m1=globalThis;class f extends I{constructor(){super(...arguments),this.renderOptions={host:this},this._$Do=void 0}createRenderRoot(){let x=super.createRenderRoot();return this.renderOptions.renderBefore??=x.firstChild,x}update(x){let d=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(x),this._$Do=C0(d,this.renderRoot,this.renderOptions)}connectedCallback(){super.connectedCallback(),this._$Do?.setConnected(!0)}disconnectedCallback(){super.disconnectedCallback(),this._$Do?.setConnected(!1)}render(){return b}}f._$litElement$=!0,f.finalized=!0,m1.litElementHydrateSupport?.({LitElement:f});var Rd=m1.litElementPolyfillSupport;Rd?.({LitElement:f});(m1.litElementVersions??=[]).push("4.2.2");var R0=null;function w0(){let x=new URL("/app/assets/datastar.js",window.location.href).href;return R0??=import(x),R0}var t=null,F0=null;function Z1(x){class d extends x{#x=null;#d=!1;connectedCallback(){this.#d=!0,super.connectedCallback(),wd().then(async()=>{if(!this.#d)return;if(this.requestUpdate(),await this.updateComplete,await Fd(),this.#d)this.requestUpdate()})}performUpdate(){if(!this.isUpdatePending)return;let h=t;if(!h){super.performUpdate();return}this.#x?.();let p=!0;this.#x=h.effect(()=>{if(Object.keys(h.root),p){p=!1,super.performUpdate();return}this.requestUpdate()})}disconnectedCallback(){this.#d=!1,this.#x?.(),this.#x=null,super.disconnectedCallback()}signal(h,p){let i=t?.getPath(h);return _1(i===void 0?p:i)}}return d}async function wd(){if(t)return t;return F0??=w0(),t=await F0,t}async function Fd(){await Promise.resolve(),await new Promise((x)=>requestAnimationFrame(()=>x()))}function _1(x){if(Array.isArray(x))return x.map((d)=>_1(d));if(x&&typeof x==="object")return Object.fromEntries(Object.entries(x).map(([d,h])=>[d,_1(h)]));return x}var $0=d1`
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
  .badge.success, .badge.running, .badge.active, .badge.online { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.cancelled, .badge.degraded { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
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
`;function R(x,d=12){if(x.length<=d)return x;return`${x.slice(0,d-1)}…`}function u1(x){if(!x)return"Not configured";let d=x.includes("@")?x.split("@").at(-1):x;return d.length>23?`${d.slice(0,16)}…${d.slice(-6)}`:d}function z1(x,d=Date.now()){let h=Date.parse(x);if(!Number.isFinite(h))return"—";let p=Math.max(0,Math.round((d-h)/1000));if(p<5)return"now";if(p<60)return`${p}s ago`;let i=Math.floor(p/60);if(i<60)return`${i}m ago`;let v=Math.floor(i/60);if(v<24)return`${v}h ago`;return`${Math.floor(v/24)}d ago`}function K1(x,d,h=Date.now()){if(!x)return"—";let p=Date.parse(x),i=d?Date.parse(d):h;if(!Number.isFinite(p)||!Number.isFinite(i)||i<p)return"—";let v=i-p;if(v<1000)return`${v}ms`;let y=v/1000;if(y<60)return`${y.toFixed(y<10?1:0)}s`;return`${Math.floor(y/60)}m ${Math.floor(y%60)}s`}function m0(x){let d=x.filter((p)=>["succeeded","success","failed","cancelled"].includes(p));if(d.length===0)return"—";let h=d.filter((p)=>p==="succeeded"||p==="success").length;return`${Math.round(h/d.length*100)}%`}function o(x){if(!Number.isFinite(x)||x<=0)return"—";let d=["B","KB","MB","GB","TB"],h=Math.min(Math.floor(Math.log(x)/Math.log(1024)),d.length-1),p=x/1024**h;return`${p>=10||Number.isInteger(p)?p.toFixed(0):p.toFixed(1)} ${d[h]}`}function H(x){if(!Number.isFinite(x))return"—";return`${Math.round(Math.max(0,Math.min(1,x))*100)}%`}function e(x){if(x==null||!Number.isFinite(x)||x<0)return"—";if(x<1000)return`${Math.round(x)}ms`;let d=x/1000;if(d<60)return`${d.toFixed(d<10?1:0)}s`;return`${Math.floor(d/60)}m ${Math.floor(d%60)}s`}function b1(x,d){if(!x)return null;if(typeof x==="function")return x(d);return x}function _0(x){if("cell"in x&&x.cell)return b1(x.cell.column.columnDef.cell,x.cell.getContext());if("header"in x&&x.header)return b1(x.header.column.columnDef.header,x.header.getContext());if("footer"in x&&x.footer)return b1(x.footer.column.columnDef.footer,x.footer.getContext());return null}var Y=function(x){return x[x.None=0]="None",x[x.Mutable=1]="Mutable",x[x.Watching=2]="Watching",x[x.RecursedCheck=4]="RecursedCheck",x[x.Recursed=8]="Recursed",x[x.Dirty=16]="Dirty",x[x.Pending=32]="Pending",x}({});function u0({update:x,notify:d,unwatched:h}){return{link:p,unlink:i,propagate:v,checkDirty:y,shallowPropagate:r};function p(q,k,z){let W=k.depsTail;if(W!==void 0&&W.dep===q)return;let Q=W!==void 0?W.nextDep:k.deps;if(Q!==void 0&&Q.dep===q){Q.version=z,k.depsTail=Q;return}let T=q.subsTail;if(T!==void 0&&T.version===z&&T.sub===k)return;let J=k.depsTail=q.subsTail={version:z,dep:q,sub:k,prevDep:W,nextDep:Q,prevSub:T,nextSub:void 0};if(Q!==void 0)Q.prevDep=J;if(W!==void 0)W.nextDep=J;else k.deps=J;if(T!==void 0)T.nextSub=J;else q.subs=J}function i(q,k=q.sub){let{dep:z,prevDep:W,nextDep:Q,nextSub:T,prevSub:J}=q;if(Q!==void 0)Q.prevDep=W;else k.depsTail=W;if(W!==void 0)W.nextDep=Q;else k.deps=Q;if(T!==void 0)T.prevSub=J;else z.subsTail=J;if(J!==void 0)J.nextSub=T;else if((z.subs=T)===void 0)h(z);return Q}function v(q){let k=q.nextSub,z;x:do{let W=q.sub,Q=W.flags;if(!(Q&(Y.RecursedCheck|Y.Recursed|Y.Dirty|Y.Pending)))W.flags=Q|Y.Pending;else if(!(Q&(Y.RecursedCheck|Y.Recursed)))Q=Y.None;else if(!(Q&Y.RecursedCheck))W.flags=Q&~Y.Recursed|Y.Pending;else if(!(Q&(Y.Dirty|Y.Pending))&&N(q,W))W.flags=Q|(Y.Recursed|Y.Pending),Q&=Y.Mutable;else Q=Y.None;if(Q&Y.Watching)d(W);if(Q&Y.Mutable){let T=W.subs;if(T!==void 0){let J=(q=T).nextSub;if(J!==void 0)z={value:k,prev:z},k=J;continue}}if((q=k)!==void 0){k=q.nextSub;continue}while(z!==void 0)if(q=z.value,z=z.prev,q!==void 0){k=q.nextSub;continue x}break}while(!0)}function y(q,k){let z,W=0,Q=!1;x:do{let T=q.dep,J=T.flags;if(k.flags&Y.Dirty)Q=!0;else if((J&(Y.Mutable|Y.Dirty))===(Y.Mutable|Y.Dirty)){if(x(T)){let Z=T.subs;if(Z.nextSub!==void 0)r(Z);Q=!0}}else if((J&(Y.Mutable|Y.Pending))===(Y.Mutable|Y.Pending)){if(q.nextSub!==void 0||q.prevSub!==void 0)z={value:q,prev:z};q=T.deps,k=T,++W;continue}if(!Q){let Z=q.nextDep;if(Z!==void 0){q=Z;continue}}while(W--){let Z=k.subs,K=Z.nextSub!==void 0;if(K)q=z.value,z=z.prev;else q=Z;if(Q){if(x(k)){if(K)r(Z);k=q.sub;continue}Q=!1}else k.flags&=~Y.Pending;k=q.sub;let E=q.nextDep;if(E!==void 0){q=E;continue x}}return Q}while(!0)}function r(q){do{let k=q.sub,z=k.flags;if((z&(Y.Pending|Y.Dirty))===Y.Pending){if(k.flags=z|Y.Dirty,(z&(Y.Watching|Y.RecursedCheck))===Y.Watching)d(k)}}while((q=q.nextSub)!==void 0)}function N(q,k){let z=k.depsTail;while(z!==void 0){if(z===q)return!0;z=z.prevDep}return!1}}function a0(x,d,h){let p=typeof x==="object",i=p?x:void 0;return{next:(p?x.next:x)?.bind(i),error:(p?x.error:d)?.bind(i),complete:(p?x.complete:h)?.bind(i)}}var o1=[],U1=0,{link:b0,unlink:$d,propagate:md,checkDirty:t0,shallowPropagate:o0}=u0({update(x){return x._update()},notify(x){o1[a1++]=x,x.flags&=~Y.Watching},unwatched(x){if(x.depsTail!==void 0)x.depsTail=void 0,x.flags=Y.Mutable|Y.Dirty,j1(x)}}),B1=0,a1=0,V,t1=0;function e1(x){try{++t1,x()}finally{if(!--t1)l1()}}function j1(x){let d=x.depsTail,h=d!==void 0?d.nextDep:x.deps;while(h!==void 0)h=$d(h,x)}function l1(){if(t1>0)return;while(B1<a1){let x=o1[B1];o1[B1++]=void 0,x.notify()}B1=0,a1=0}function M1(x,d){let h=typeof x==="function",p=x,i={_snapshot:h?void 0:x,subs:void 0,subsTail:void 0,deps:void 0,depsTail:void 0,flags:h?Y.None:Y.Mutable,get(){if(V!==void 0)b0(i,V,U1);return i._snapshot},subscribe(v){let y=a0(v),r={current:!1},N=_d(()=>{if(i.get(),!r.current)r.current=!0;else y.next?.(i._snapshot)});return{unsubscribe:()=>{N.stop()}}},_update(v){let y=V,r=d?.compare??Object.is;if(h)V=i,++U1,i.depsTail=void 0;else if(v===void 0)return!1;if(h)i.flags=Y.Mutable|Y.RecursedCheck;try{let N=i._snapshot,q=typeof v==="function"?v(N):v===void 0&&h?p(N):v;if(N===void 0||!r(N,q))return i._snapshot=q,!0;return!1}finally{if(V=y,h)i.flags&=~Y.RecursedCheck;j1(i)}}};if(h)i.flags=Y.Mutable|Y.Dirty,i.get=function(){let v=i.flags;if(v&Y.Dirty||v&Y.Pending&&t0(i.deps,i)){if(i._update()){let y=i.subs;if(y!==void 0)o0(y)}}else if(v&Y.Pending)i.flags=v&~Y.Pending;if(V!==void 0)b0(i,V,U1);return i._snapshot};else i.set=function(v){if(i._update(v)){let y=i.subs;if(y!==void 0)md(y),o0(y),l1()}};return i}function _d(x){let d=()=>{let p=V;V=h,++U1,h.depsTail=void 0,h.flags=Y.Watching|Y.RecursedCheck;try{return x()}finally{V=p,h.flags&=~Y.RecursedCheck,j1(h)}},h={deps:void 0,depsTail:void 0,subs:void 0,subsTail:void 0,flags:Y.Watching|Y.RecursedCheck,notify(){let p=this.flags;if(p&Y.Dirty||p&Y.Pending&&t0(this.deps,this))d();else this.flags=Y.Watching},stop(){this.flags=Y.None,this.depsTail=void 0,j1(this)}};return d(),h}function e0(){return{createOptionsStore:!0,wrapExternalAtoms:!1,addSubscription:()=>{throw Error("Feature not supported in current reactivity implementation")},unmount:()=>{throw Error("Feature not supported in current reactivity implementation")},schedule:(x)=>queueMicrotask(()=>x()),batch:e1,untrack:(x)=>x(),createReadonlyAtom:(x,d)=>{return M1(()=>x(),{compare:d===null||d===void 0?void 0:d.compare})},createWritableAtom:(x,d)=>{return M1(x,{compare:d===null||d===void 0?void 0:d.compare})}}}function P(x,d){return typeof x==="function"?x(d):x}function O(x){if(Array.isArray(x))return x.map(O);if(x&&typeof x==="object"){let d=Object.getPrototypeOf(x);if(d!==Object.prototype&&d!==null)return x;let h={},p=Object.keys(x);for(let i=0;i<p.length;i++){let v=p[i];h[v]=O(x[v])}return h}return x}function g1(x,d){return(h)=>{var p;(((p=d.options.atoms)===null||p===void 0?void 0:p[x])??d.baseAtoms[x]).set((i)=>P(h,i))}}function L1(x){return x instanceof Function}function l0(x,d){let h=[],p=(i)=>{i.forEach((v)=>{h.push(v);let y=d(v);if(y.length)p(y)})};return p(x),h}var ud=({fn:x,memoDeps:d,onAfterCompare:h,onAfterUpdate:p,onBeforeCompare:i,onBeforeUpdate:v})=>{let y=[],r;return(q)=>{i===null||i===void 0||i();let k=d===null||d===void 0?void 0:d(q),z=!k||k.length!==(y===null||y===void 0?void 0:y.length);if(!z&&k){for(let W=0;W<k.length;W++)if(k[W]!==y[W]){z=!0;break}}if(h===null||h===void 0||h(z),!z)return r;return y=k,v===null||v===void 0||v(),r=x(...k??[]),p===null||p===void 0||p(r),r}},bd=(x,d)=>{x=String(x);while(x.length<d)x=" "+x;return x};function l({feature:x,fnName:d,objectId:h,onAfterUpdate:p,table:i,...v}){let y,r,N,q,k=0,z,W;function Q(Z,K){var E;let S=k===0?"(1st run)":K?"(rerun #"+k+")":"(cache)";k++,console.groupCollapsed(`%c⏱ ${bd(`${Z.toFixed(1)} ms`,12)} %c${S}%c ${d}%c ${h?`(${d.split(".")[0]}Id: ${h})`:""}`,`font-size: .6rem; font-weight: bold; ${K?`color: hsl(
        ${Math.max(0,Math.min(120-Math.log10(Z)*60,120))}deg 100% 31%);`:""} `,`color: ${k<2?"#FF00FF":"#FF1493"}`,"color: #666","color: #87CEEB"),console.info({feature:x,state:i.store.state,deps:(E=v.memoDeps)===null||E===void 0?void 0:E.toString()}),console.trace(),console.groupEnd()}let T=()=>{if(!p)return;let{schedule:Z,untrack:K}=i._reactivity;Z(()=>K(()=>p()))};return ud({...v,...{onAfterUpdate:()=>{T()}}})}function xx(x,d="_"){let[h,p]=x.split(d);return{fnKey:p,fnName:`${h}.${p}`,parentName:h}}function A(x,d,h){for(let[p,{fn:i,memoDeps:v}]of Object.entries(h)){let{fnKey:y,fnName:r}=xx(p);d[y]=v?l({memoDeps:v,fn:i,fnName:r,table:d,feature:x}):i}}function n(x,d,h,p){for(let[i,{fn:v,memoDeps:y}]of Object.entries(p)){let{fnKey:r,fnName:N}=xx(i);if(y){let q=`_memo_${r}`;d[r]=function(...k){if(!this[q]){let z=this;this[q]=l({memoDeps:(W)=>y(z,W),fn:(...W)=>v(z,...W),fnName:N,objectId:z.id,table:h,feature:x})}return this[q](...k)}}else d[r]=function(...q){return v(this,...q)}}}function G(x,d,h,...p){var i;return((i=x[d])===null||i===void 0?void 0:i.call(x,...p))??h(x,...p)}function dx(x){return x.row.getValue(x.column.id)}function hx(x){return x.getValue()??x.table.options.renderFallbackValue}function px(x){return{table:x.table,column:x.column,row:x.row,cell:x,getValue:()=>x.getValue(),renderValue:()=>x.renderValue()}}var ix={assignCellPrototype:(x,d)=>{n("coreCellsFeature",x,d,{cell_getValue:{fn:(h)=>dx(h)},cell_renderValue:{fn:(h)=>hx(h)},cell_getContext:{fn:(h)=>px(h),memoDeps:(h)=>[h]}})}};function od(x){if(!x._headerPrototype){x._headerPrototype={table:x};let p=Object.values(x._features);for(let i=0;i<p.length;i++){var d,h;(d=(h=p[i]).assignHeaderPrototype)===null||d===void 0||d.call(h,x._headerPrototype,x)}}return x._headerPrototype}function x0(x,d,h){let p=od(x),i=Object.create(p);return i.colSpan=0,i.column=d,i.depth=h.depth,i.headerGroup=null,i.id=h.id??d.id,i.index=h.index,i.isPlaceholder=!!h.isPlaceholder,i.placeholderId=h.placeholderId,i.rowSpan=0,i.subHeaders=[],i}function vx(){return{left:[],right:[]}}function w(x){var d;let h=x.columns;return(h.length?h.some((p)=>G(p,"getIsVisible",w)):(d=x.table.atoms.columnVisibility)===null||d===void 0||(d=d.get())===null||d===void 0?void 0:d[x.id])??!0}function yx(x){return x.getAllLeafColumns().filter((d)=>G(d,"getIsVisible",w))}function d0(x,d,h,p){var i;let v=0,y=(k,z=1)=>{v=Math.max(v,z);for(let W=0;W<k.length;W++){let Q=k[W];if(G(Q,"getIsVisible",w)){if(Q.columns.length)y(Q.columns,z+1)}}};y(x);let r=[],N=(k,z)=>{let W={depth:z,id:[p,`${z}`].filter(Boolean).join("_"),headers:[]},Q=[];if(k.forEach((T)=>{let J=Q[Q.length-1],Z=T.column.depth===W.depth,K,E=!1;if(Z&&T.column.parent)K=T.column.parent;else K=T.column,E=!0;if(J&&J.column===K)J.subHeaders.push(T);else{let S=x0(h,K,{id:[p,z,K.id,T.id].filter(Boolean).join("_"),isPlaceholder:E,placeholderId:E?`${Q.filter(($)=>$.column===K).length}`:void 0,depth:z,index:Q.length});S.subHeaders.push(T),Q.push(S)}W.headers.push(T),T.headerGroup=W}),r.push(W),z>0)N(Q,z-1)};N(d.map((k,z)=>x0(h,k,{depth:v,index:z})),v-1),r.reverse();let q=(k)=>{let z=[];for(let W=0;W<k.length;W++){let Q=k[W];if(!G(Q.column,"getIsVisible",w))continue;let T=0,J=1/0;if(Q.subHeaders.length){let Z=q(Q.subHeaders);for(let K=0;K<Z.length;K++){let E=Z[K];if(T+=E.colSpan,E.rowSpan<J)J=E.rowSpan}}else T=1,J=0;Q.colSpan=T,Q.rowSpan=J,z.push({colSpan:T,rowSpan:Q.rowSpan})}return z};return q(((i=r[0])===null||i===void 0?void 0:i.headers)??[]),r}function ad(x){if(!x._columnPrototype){x._columnPrototype={table:x};let p=Object.values(x._features);for(let i=0;i<p.length;i++){var d,h;(d=(h=p[i]).assignColumnPrototype)===null||d===void 0||d.call(h,x._columnPrototype,x)}}return x._columnPrototype}function kx(x,d,h,p){let i={...x.getDefaultColumnDef(),...d},v=i.accessorKey,y=i.id??(v?v.replaceAll(".","_"):void 0)??(typeof i.header==="string"?i.header:void 0),r;if(i.accessorFn)r=i.accessorFn;else if(v)if(v.includes(".")){let k=v.split(".");r=(z)=>{let W=z;for(let Q=0;Q<k.length;Q++){let T=k[Q];W=W===null||W===void 0?void 0:W[T]}return W}}else r=(k)=>k[i.accessorKey];if(!y)throw Error();let N=ad(x),q=Object.create(N);return q.accessorFn=r,q.columnDef=i,q.columns=[],q.depth=h,q.id=`${String(y)}`,q.parent=p,q}function h0(x){var d;let h=(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get();return(p)=>{let i=[];if(!(h===null||h===void 0?void 0:h.length))i=p;else{let v=new Map;for(let y=0;y<p.length;y++){let r=p[y];v.set(r.id,r)}for(let y=0;y<h.length;y++){let r=h[y],N=v.get(r);if(N)i.push(N),v.delete(r)}for(let y=0;y<p.length;y++){let r=p[y];if(v.has(r.id))i.push(r)}}return td(x,i)}}function td(x,d){var h;let p=((h=x.atoms.grouping)===null||h===void 0?void 0:h.get())??[],{groupedColumnMode:i}=x.options;if(!p.length||!i)return d;let v=d.filter((N)=>!p.includes(N.id));if(i==="remove")return v;let y=new Map;for(let N=0;N<d.length;N++){let q=d[N];y.set(q.id,q)}let r=[];for(let N=0;N<p.length;N++){let q=y.get(p[N]);if(q)r.push(q)}return[...r,...v]}function rx(x){return[x,...x.columns.flatMap((d)=>d.getFlatColumns())]}function zx(x){if(x.columns.length){let d=x.columns.flatMap((h)=>h.getLeafColumns());return G(x.table,"getOrderColumns",h0)(d)}return[x]}function qx(x){return{header:(d)=>{let h=d.header.column.columnDef;if(h.accessorKey)return h.accessorKey;if(h.accessorFn)return h.id;return null},cell:(d)=>{var h,p;return((h=d.renderValue())===null||h===void 0||(p=h.toString)===null||p===void 0?void 0:p.call(h))??null},...Object.values(x._features).reduce((d,h)=>{var p;return Object.assign(d,(p=h.getDefaultColumnDef)===null||p===void 0?void 0:p.call(h))},{}),...x.options.defaultColumn}}function Nx(x){let d=(h,p,i=0)=>{return h.map((v)=>{let y=kx(x,v,i,p),r=v;return y.columns=r.columns?d(r.columns,y,i+1):[],y})};return d(x.options.columns)}function Qx(x){return x.getAllColumns().flatMap((d)=>d.getFlatColumns())}function Tx(x){let d={},h=x.getAllFlatColumns();for(let p=0;p<h.length;p++){let i=h[p];d[i.id]=i}return d}function Wx(x){let d=x.getAllColumns().flatMap((h)=>h.getLeafColumns());return G(x,"getOrderColumns",h0)(d)}function Yx(x){let d={},h=x.getAllLeafColumns();for(let p=0;p<h.length;p++){let i=h[p];d[i.id]=i}return d}function A1(x,d){return x.getAllFlatColumnsById()[d]}var Jx={assignColumnPrototype:(x,d)=>{n("coreColumnsFeature",x,d,{column_getFlatColumns:{fn:(h)=>rx(h),memoDeps:(h)=>[h.table.options.columns]},column_getLeafColumns:{fn:(h)=>zx(h),memoDeps:(h)=>{var p,i;return[(p=h.table.atoms.columnOrder)===null||p===void 0?void 0:p.get(),(i=h.table.atoms.grouping)===null||i===void 0?void 0:i.get(),h.table.options.columns,h.table.options.groupedColumnMode]}}})},constructTableAPIs:(x)=>{A("coreColumnsFeature",x,{table_getDefaultColumnDef:{fn:()=>qx(x),memoDeps:()=>[x.options.defaultColumn]},table_getAllColumns:{fn:()=>Nx(x),memoDeps:()=>[x.options.columns]},table_getAllFlatColumns:{fn:()=>Qx(x),memoDeps:()=>[x.options.columns]},table_getAllFlatColumnsById:{fn:()=>Tx(x),memoDeps:()=>[x.options.columns]},table_getAllLeafColumns:{fn:()=>Wx(x),memoDeps:()=>{var d,h;return[(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get(),(h=x.atoms.grouping)===null||h===void 0?void 0:h.get(),x.options.columns,x.options.groupedColumnMode]}},table_getAllLeafColumnsById:{fn:()=>Yx(x),memoDeps:()=>[x.getAllLeafColumns()]},table_getColumn:{fn:(d)=>A1(x,d)}})}};function Ex(x){let d=[],h=(p)=>{if(p.subHeaders.length)p.subHeaders.map(h);d.push(p)};return h(x),d}function Xx(x){return{column:x.column,header:x,table:x.column.table}}function Zx(x){var d;let{left:h,right:p}=((d=x.atoms.columnPinning)===null||d===void 0?void 0:d.get())??vx(),i=x.getAllColumns(),v=G(x,"getVisibleLeafColumns",yx);if(!h.length&&!p.length)return d0(i,v,x);let y=x.getAllLeafColumnsById(),r=[];for(let k=0;k<h.length;k++){let z=y[h[k]];if(z&&G(z,"getIsVisible",w))r.push(z)}let N=[];for(let k=0;k<p.length;k++){let z=y[p[k]];if(z&&G(z,"getIsVisible",w))N.push(z)}let q=v.filter((k)=>!h.includes(k.id)&&!p.includes(k.id));return d0(i,[...r,...q,...N],x)}function Kx(x){return[...x.getHeaderGroups()].reverse()}function Bx(x){let d=x.getHeaderGroups(),h=[];for(let p=0;p<d.length;p++){let i=d[p].headers;for(let v=0;v<i.length;v++)h.push(i[v])}return h}function Ux(x){var d;let h=((d=x.getHeaderGroups()[0])===null||d===void 0?void 0:d.headers)??[],p=[];for(let i=0;i<h.length;i++){let v=h[i].getLeafHeaders();for(let y=0;y<v.length;y++)p.push(v[y])}return p}var jx={assignHeaderPrototype:(x,d)=>{n("coreHeadersFeature",x,d,{header_getLeafHeaders:{fn:(h)=>Ex(h),memoDeps:(h)=>[h.column.table.options.columns]},header_getContext:{fn:(h)=>Xx(h),memoDeps:(h)=>[h.column.table.options.columns]}})},constructTableAPIs:(x)=>{A("coreHeadersFeature",x,{table_getHeaderGroups:{fn:()=>Zx(x),memoDeps:()=>{var d,h,p,i;return[x.options.columns,(d=x.atoms.columnOrder)===null||d===void 0?void 0:d.get(),(h=x.atoms.grouping)===null||h===void 0?void 0:h.get(),(p=x.atoms.columnPinning)===null||p===void 0?void 0:p.get(),(i=x.atoms.columnVisibility)===null||i===void 0?void 0:i.get(),x.options.groupedColumnMode]}},table_getFooterGroups:{fn:()=>Kx(x),memoDeps:()=>[x.getHeaderGroups()]},table_getFlatHeaders:{fn:()=>Bx(x),memoDeps:()=>[x.getHeaderGroups()]},table_getLeafHeaders:{fn:()=>Ux(x),memoDeps:()=>[x.getHeaderGroups()]}})}};function ed(x){if(!x._rowPrototype){x._rowPrototype={table:x};let p=Object.values(x._features);for(let i=0;i<p.length;i++){var d,h;(d=(h=p[i]).assignRowPrototype)===null||d===void 0||d.call(h,x._rowPrototype,x)}}return x._rowPrototype}var q1=(x,d,h,p,i,v,y)=>{let r=ed(x),N=Object.create(r);N._uniqueValuesCache={},N._valuesCache={},N.depth=i,N.id=d,N.index=p,N.original=h,N.parentId=y,N.subRows=v??[];let q=Object.values(x._features);for(let W=0;W<q.length;W++){var k,z;(k=(z=q[W]).initRowInstanceData)===null||k===void 0||k.call(z,N)}return N};var p0=0;function G1(x){if(x.options.autoResetAll??x.options.autoResetPageIndex??!x.options.manualPagination)dh(x)}function ld(x,d){var h,p;let i=(v)=>{return P(d,v)};return(h=(p=x.options).onPaginationChange)===null||h===void 0?void 0:h.call(p,i)}function xh(x,d){ld(x,(h)=>{let p=P(d,h.pageIndex),i=typeof x.options.pageCount>"u"||x.options.pageCount===-1?Number.MAX_SAFE_INTEGER:x.options.pageCount-1;return p=Math.max(0,Math.min(p,i)),{...h,pageIndex:p}})}function dh(x,d){var h,p;let i=((h=x.atoms.pagination)===null||h===void 0||(h=h.get())===null||h===void 0?void 0:h.pageIndex)??p0,v=d?p0:((p=x.initialState.pagination)===null||p===void 0?void 0:p.pageIndex)??p0;if(v===i)return;xh(x,v)}function N1(){return(x)=>{return l({feature:"coreRowModelsFeature",table:x,fnName:"table.getCoreRowModel",memoDeps:()=>[x.options.data],fn:()=>hh(x,x.options.data),onAfterUpdate:()=>G1(x)})}}function hh(x,d){let h={rows:[],flatRows:[],rowsById:{}},p=(i,v=0,y)=>{let r=[];for(let q=0;q<i.length;q++){let k=i[q],z=q1(x,x.getRowId(k,q,y),k,q,v,void 0,y===null||y===void 0?void 0:y.id);if(h.flatRows.push(z),h.rowsById[z.id]=z,r.push(z),x.options.getSubRows){var N;if(z.originalSubRows=x.options.getSubRows(k,q),(N=z.originalSubRows)===null||N===void 0?void 0:N.length)z.subRows=p(z.originalSubRows,v+1,z)}}return r};return h.rows=p(d),h}function Mx(x){if(!x._rowModels.coreRowModel){var d,h;x._rowModels.coreRowModel=((d=(h=x.options.features).coreRowModel)===null||d===void 0?void 0:d.call(h,x))??N1()(x)}return x._rowModels.coreRowModel()}function gx(x){return x.getCoreRowModel()}function Lx(x){if(!x._rowModels.filteredRowModel){var d,h;x._rowModels.filteredRowModel=(d=(h=x.options.features).filteredRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualFiltering||!x._rowModels.filteredRowModel)return x.getPreFilteredRowModel();return x._rowModels.filteredRowModel()}function Ax(x){return x.getFilteredRowModel()}function Gx(x){if(!x._rowModels.groupedRowModel){var d,h;x._rowModels.groupedRowModel=(d=(h=x.options.features).groupedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualGrouping||!x._rowModels.groupedRowModel)return x.getPreGroupedRowModel();return x._rowModels.groupedRowModel()}function Dx(x){return x.getGroupedRowModel()}function Hx(x){if(!x._rowModels.sortedRowModel){var d,h;x._rowModels.sortedRowModel=(d=(h=x.options.features).sortedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualSorting||!x._rowModels.sortedRowModel)return x.getPreSortedRowModel();return x._rowModels.sortedRowModel()}function Ox(x){return x.getSortedRowModel()}function nx(x){if(!x._rowModels.expandedRowModel){var d,h;x._rowModels.expandedRowModel=(d=(h=x.options.features).expandedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualExpanding||!x._rowModels.expandedRowModel)return x.getPreExpandedRowModel();return x._rowModels.expandedRowModel()}function cx(x){return x.getExpandedRowModel()}function Vx(x){if(!x._rowModels.paginatedRowModel){var d,h;x._rowModels.paginatedRowModel=(d=(h=x.options.features).paginatedRowModel)===null||d===void 0?void 0:d.call(h,x)}if(x.options.manualPagination||!x._rowModels.paginatedRowModel)return x.getPrePaginatedRowModel();return x._rowModels.paginatedRowModel()}function fx(x){return x.getPaginatedRowModel()}var Px={constructTableAPIs:(x)=>{A("coreRowModelsFeature",x,{table_getCoreRowModel:{fn:()=>Mx(x)},table_getPreFilteredRowModel:{fn:()=>gx(x)},table_getFilteredRowModel:{fn:()=>Lx(x)},table_getPreGroupedRowModel:{fn:()=>Ax(x)},table_getGroupedRowModel:{fn:()=>Gx(x)},table_getPreSortedRowModel:{fn:()=>Dx(x)},table_getSortedRowModel:{fn:()=>Hx(x)},table_getPreExpandedRowModel:{fn:()=>Ox(x)},table_getExpandedRowModel:{fn:()=>nx(x)},table_getPrePaginatedRowModel:{fn:()=>cx(x)},table_getPaginatedRowModel:{fn:()=>Vx(x)},table_getRowModel:{fn:()=>fx(x)}})}};function ph(x){if(!x._cellPrototype){x._cellPrototype={table:x};let p=Object.values(x._features);for(let i=0;i<p.length;i++){var d,h;(d=(h=p[i]).assignCellPrototype)===null||d===void 0||d.call(h,x._cellPrototype,x)}}return x._cellPrototype}function Sx(x,d,h){let p=ph(h),i=Object.create(p);return i.column=x,i.id=`${d.id}_${x.id}`,i.row=d,i}function sx(x,d){if(x._valuesCache.hasOwnProperty(d))return x._valuesCache[d];let h=x.table.getColumn(d);if(!(h===null||h===void 0?void 0:h.accessorFn))return;return x._valuesCache[d]=h.accessorFn(x.original,x.index),x._valuesCache[d]}function Ix(x,d){if(x._uniqueValuesCache.hasOwnProperty(d))return x._uniqueValuesCache[d];let h=x.table.getColumn(d);if(!(h===null||h===void 0?void 0:h.accessorFn))return;if(!h.columnDef.getUniqueValues)return x._uniqueValuesCache[d]=[x.getValue(d)],x._uniqueValuesCache[d];return x._uniqueValuesCache[d]=h.columnDef.getUniqueValues(x.original,x.index),x._uniqueValuesCache[d]}function Cx(x,d){return x.getValue(d)??x.table.options.renderFallbackValue}function Rx(x){return l0(x.subRows,(d)=>d.subRows)}function wx(x){return x.parentId?x.table.getRow(x.parentId,!0):void 0}function Fx(x){let d=[],h=x;while(!0){let p=h.getParentRow();if(!p)break;d.push(p),h=p}return d.reverse()}function $x(x){let d=x.table.getAllLeafColumns(),h=Array(d.length);for(let p=0;p<d.length;p++)h[p]=Sx(d[p],x,x.table);return h}function mx(x){let d={},h=x.getAllCells();for(let p=0;p<h.length;p++){let i=h[p];d[i.column.id]=i}return d}function _x(x,d,h,p){var i,v;return((i=(v=d.options).getRowId)===null||i===void 0?void 0:i.call(v,x,h,p))??`${p?[p.id,h].join("."):h}`}function ux(x,d,h){let p=(h?x.getPrePaginatedRowModel():x.getRowModel()).rowsById[d];if(!p){if(p=x.getCoreRowModel().rowsById[d],!p)throw Error()}return p}var bx={assignRowPrototype:(x,d)=>{n("coreRowsFeature",x,d,{row_getAllCellsByColumnId:{fn:(h)=>mx(h),memoDeps:(h)=>[h.getAllCells()]},row_getAllCells:{fn:(h)=>$x(h),memoDeps:(h)=>[h.table.getAllLeafColumns()]},row_getLeafRows:{fn:(h)=>Rx(h)},row_getParentRow:{fn:(h)=>wx(h)},row_getParentRows:{fn:(h)=>Fx(h)},row_getUniqueValues:{fn:(h,p)=>Ix(h,p)},row_getValue:{fn:(h,p)=>sx(h,p)},row_renderValue:{fn:(h,p)=>Cx(h,p)}})},constructTableAPIs:(x)=>{A("coreRowsFeature",x,{table_getRowId:{fn:(d,h,p)=>_x(d,x,h,p)},table_getRow:{fn:(d,h)=>ux(x,d,h)}})}};function i0(x){let d=x.options.state;if(!d)return;x._reactivity.batch(()=>{for(let h in d){let p=x.baseAtoms[h];if(!p)continue;let i=d[h];if(i!==p.get())p.set(()=>i)}})}function ox(x){let d=O(x.initialState);x._reactivity.batch(()=>{let h=Object.keys(d);for(let p=0;p<h.length;p++){let i=h[p];x.baseAtoms[i].set(d[i])}})}function ih(x,d){if(x.options.mergeOptions)return x.options.mergeOptions(x.options,d);return{...x.options,...d}}function ax(x,d){let h=P(d,x.options),{features:p,atoms:i,initialState:v}=x.options,y=Object.assign(ih(x,h),{features:p,atoms:i,initialState:v});if(x.optionsStore)x.optionsStore.set(()=>y);else x.options=y;i0(x)}var tx={constructTableAPIs:(x)=>{A("coreTablesFeature",x,{table_reset:{fn:()=>ox(x)},table_setOptions:{fn:(d)=>ax(x,d)}})}};var ex={coreCellsFeature:ix,coreColumnsFeature:Jx,coreHeadersFeature:jx,coreRowModelsFeature:Px,coreRowsFeature:bx,coreTablesFeature:tx};function v0(x){return x}function lx(x){let d=x;if(Object.defineProperty(x,"state",{get(){return x.get()}}),"set"in x)d.setState=x.set.bind(x);return d}function xd(x,d={}){return Object.values(x).forEach((h)=>{var p;d=((p=h.getInitialState)===null||p===void 0?void 0:p.call(h,d))??d}),O(d)}function y0(x){let d=x.features.coreReactivityFeature,{aggregationFns:h,columnMeta:p,coreRowModel:i,expandedRowModel:v,facetedMinMaxValues:y,facetedRowModel:r,facetedUniqueValues:N,filterFns:q,filterMeta:k,filteredRowModel:z,groupedRowModel:W,paginatedRowModel:Q,sortFns:T,sortedRowModel:J,tableMeta:Z,...K}=x.features,E={_reactivity:d,_features:{...ex,...K},_rowModels:{},_rowModelFns:{aggregationFns:h,filterFns:q,sortFns:T},baseAtoms:{},atoms:{}},S=Object.values(E._features),$={...S.reduce((U,j)=>{var L;return Object.assign(U,(L=j.getDefaultTableOptions)===null||L===void 0?void 0:L.call(j,E))},{}),...x};if(d.wrapExternalAtoms&&$.atoms)for(let[U,j]of Object.entries($.atoms)){let L=j,s=d.createWritableAtom(L.get(),{debugName:`externalAtom/${U}`});$.atoms[U]=s;let S1=!1,Ad=L.subscribe((s1)=>{if(S1)return;s.set(s1)}),Gd=s.subscribe((s1)=>{S1=!0,L.set(s1),S1=!1});d.addSubscription(Ad),d.addSubscription(Gd)}if(d.createOptionsStore)E.optionsStore=d.createWritableAtom($,{debugName:"table/optionsStore"}),Object.defineProperty(E,"options",{configurable:!0,enumerable:!0,get(){return E.optionsStore.get()},set(U){E.optionsStore.set(()=>U)}});else E.options=$;E.initialState=xd(E._features,E.options.initialState);let T1=Object.keys(E.initialState);for(let U=0;U<T1.length;U++){let j=T1[U];E.baseAtoms[j]=d.createWritableAtom(E.initialState[j],{debugName:`table/baseAtoms/${j}`}),E.atoms[j]=d.createReadonlyAtom(()=>{let L=E.options.atoms,s=L===null||L===void 0?void 0:L[j];if(s)return s.get();return E.baseAtoms[j].get()},{debugName:`table/atoms/${j}`})}i0(E),E.store=lx(d.createReadonlyAtom(()=>{let U={};for(let j=0;j<T1.length;j++){let L=T1[j];U[L]=E.atoms[L].get()}return U},{debugName:"table/store"}));for(let U=0;U<S.length;U++){var P1,X0;(P1=(X0=S[U]).constructTableAPIs)===null||P1===void 0||P1.call(X0,E)}return E}var k0=Object.assign((x,d,h)=>{return x.getValue(d)===h},{autoRemove:(x)=>g(x)}),dd=Object.assign((x,d,h)=>{return x.getValue(d)==h},{autoRemove:(x)=>g(x)}),hd=Object.assign((x,d,h)=>{var p;return Boolean((p=x.getValue(d))===null||p===void 0?void 0:p.toString().includes(String(h)))},{autoRemove:(x)=>g(x)}),D1=Object.assign((x,d,h)=>{var p;return Boolean((p=x.getValue(d))===null||p===void 0?void 0:p.toString().toLowerCase().includes(String(h).toLowerCase()))},{autoRemove:(x)=>g(x)}),pd=Object.assign((x,d,h)=>{var p;return((p=x.getValue(d))===null||p===void 0?void 0:p.toString().toLowerCase())===String(h).toLowerCase()},{autoRemove:(x)=>g(x)}),vh=Object.assign((x,d,h)=>{var p;return((p=x.getValue(d))===null||p===void 0?void 0:p.toString())===String(h)},{autoRemove:(x)=>g(x)}),H1=Object.assign((x,d,h)=>{let p=x.getValue(d),i=p===null||p===void 0?0:+p,v=Number(h);if(!isNaN(v)&&!isNaN(i))return i>v;return(p??"").toString().toLowerCase().trim()>String(h).toLowerCase().trim()},{resolveFilterValue:(x)=>g(x)}),r0=Object.assign((x,d,h)=>{return H1(x,d,h)||k0(x,d,h)},{resolveFilterValue:(x)=>g(x)}),id=Object.assign((x,d,h)=>{return!r0(x,d,h)},{resolveFilterValue:(x)=>g(x)}),vd=Object.assign((x,d,h)=>{return!H1(x,d,h)},{resolveFilterValue:(x)=>g(x)}),yh=Object.assign((x,d,h)=>(["",void 0].includes(h[0])||H1(x,d,h[0]))&&(!isNaN(Number(h[0]))&&!isNaN(Number(h[1]))&&Number(h[0])>Number(h[1])||["",void 0].includes(h[1])||id(x,d,h[1])),{autoRemove:(x)=>!x}),kh=Object.assign((x,d,h)=>(["",void 0].includes(h[0])||r0(x,d,h[0]))&&(!isNaN(Number(h[0]))&&!isNaN(Number(h[1]))&&Number(h[0])>Number(h[1])||["",void 0].includes(h[1])||vd(x,d,h[1])),{autoRemove:(x)=>!x}),yd=Object.assign((x,d,h)=>{let[p,i]=h,v=x.getValue(d);return v>=p&&v<=i},{resolveFilterValue:(x)=>{let[d,h]=x,p=typeof d!=="number"?parseFloat(d):d,i=typeof h!=="number"?parseFloat(h):h,v=d===null||Number.isNaN(p)?-1/0:p,y=h===null||Number.isNaN(i)?1/0:i;if(v>y){let r=v;v=y,y=r}return[v,y]},autoRemove:(x)=>g(x)||g(x[0])&&g(x[1])}),kd=(x,d,h)=>{return h.some((p)=>x.getValue(d)===p)},rd=Object.assign((x,d,h)=>{return h.some((p)=>x.getValue(d).includes(p))},{autoRemove:(x)=>g(x)||!(x===null||x===void 0?void 0:x.length)}),zd=Object.assign((x,d,h)=>{let p=x.getValue(d);if(!Array.isArray(p))return!1;return!h.some((i)=>!p.includes(i))},{autoRemove:(x)=>g(x)||!(x===null||x===void 0?void 0:x.length)}),qd=Object.assign((x,d,h)=>{let p=x.getValue(d);if(!Array.isArray(p))return!1;return h.some((i)=>p.includes(i))},{autoRemove:(x)=>g(x)||!(x===null||x===void 0?void 0:x.length)}),z0={arrIncludes:rd,arrIncludesAll:zd,arrHas:kd,arrIncludesSome:qd,between:yh,betweenInclusive:kh,equals:k0,equalsString:pd,inNumberRange:yd,includesString:D1,includesStringSensitive:hd,weakEquals:dd};function g(x){return x===void 0||x===null||x===""}function Nd(){return[]}function q0(x){let d=x.table._rowModelFns.filterFns,h=x.table.getCoreRowModel().flatRows[0],p=h?h.getValue(x.id):void 0;if(typeof p==="string")return d===null||d===void 0?void 0:d.includesString;if(typeof p==="number")return d===null||d===void 0?void 0:d.inNumberRange;if(typeof p==="boolean")return d===null||d===void 0?void 0:d.equals;if(p!==null&&typeof p==="object")return d===null||d===void 0?void 0:d.equals;if(Array.isArray(p))return d===null||d===void 0?void 0:d.arrIncludes;return d===null||d===void 0?void 0:d.weakEquals}function x1(x){let d=null,h=x.table._rowModelFns.filterFns;return d=L1(x.columnDef.filterFn)?x.columnDef.filterFn:x.columnDef.filterFn==="auto"?q0(x):h===null||h===void 0?void 0:h[x.columnDef.filterFn],d??void 0}function Qd(x){return(x.columnDef.enableColumnFilter??!0)&&(x.table.options.enableColumnFilters??!0)&&(x.table.options.enableFilters??!0)&&!!x.accessorFn}function Td(x){return N0(x)>-1}function Wd(x){var d;return(d=x.table.atoms.columnFilters)===null||d===void 0||(d=d.get())===null||d===void 0||(d=d.find((h)=>h.id===x.id))===null||d===void 0?void 0:d.value}function N0(x){var d;return((d=x.table.atoms.columnFilters)===null||d===void 0||(d=d.get())===null||d===void 0?void 0:d.findIndex((h)=>h.id===x.id))??-1}function Yd(x,d){O1(x.table,(h)=>{let p=x1(x),i=h.find((r)=>r.id===x.id),v=P(d,i?i.value:void 0);if(Ed(p,v,x))return h.filter((r)=>r.id!==x.id);let y={id:x.id,value:v};if(i)return h.map((r)=>{if(r.id===x.id)return y;return r});if(h.length)return[...h,y];return[y]})}function O1(x,d){var h,p;let i=x.getAllLeafColumnsById(),v=(y)=>{return P(d,y).filter((r)=>{let N=i[r.id];if(N){if(Ed(x1(N),r.value,N))return!1}return!0})};(h=(p=x.options).onColumnFiltersChange)===null||h===void 0||h.call(p,v)}function Jd(x,d){O1(x,d?[]:O(x.initialState.columnFilters??[]))}function Ed(x,d,h){return(x&&x.autoRemove?x.autoRemove(d,h):!1)||typeof d>"u"||typeof d==="string"&&!d}var Q0={getInitialState:(x)=>{return{columnFilters:Nd(),...x}},getDefaultColumnDef:()=>{return{filterFn:"auto"}},getDefaultTableOptions:(x)=>{return{onColumnFiltersChange:g1("columnFilters",x),filterFromLeafRows:!1,maxLeafRowFilterDepth:100}},assignColumnPrototype:(x,d)=>{n("columnFilteringFeature",x,d,{column_getAutoFilterFn:{fn:(h)=>q0(h)},column_getFilterFn:{fn:(h)=>x1(h)},column_getCanFilter:{fn:(h)=>Qd(h)},column_getIsFiltered:{fn:(h)=>Td(h)},column_getFilterValue:{fn:(h)=>Wd(h)},column_getFilterIndex:{fn:(h)=>N0(h)},column_setFilterValue:{fn:(h,p)=>Yd(h,p)}})},initRowInstanceData:(x)=>{x.columnFilters={},x.columnFiltersMeta={}},constructTableAPIs:(x)=>{A("columnFilteringFeature",x,{table_setColumnFilters:{fn:(d)=>O1(x,d)},table_resetColumnFilters:{fn:(d)=>Jd(x,d)}})}};function n1(x){var d,h;return(x.columnDef.enableGlobalFilter??!0)&&(x.table.options.enableGlobalFilter??!0)&&(x.table.options.enableFilters??!0)&&(((d=(h=x.table.options).getColumnCanGlobalFilter)===null||d===void 0?void 0:d.call(h,x))??!0)&&!!x.accessorFn}function T0(){return D1}function c1(x){let{globalFilterFn:d}=x.options,h=x._rowModelFns.filterFns;return L1(d)?d:d==="auto"?T0():h===null||h===void 0?void 0:h[d]}function W0(x,d){var h,p;(h=(p=x.options).onGlobalFilterChange)===null||h===void 0||h.call(p,d)}function Xd(x,d){W0(x,d?void 0:O(x.initialState.globalFilter))}var Y0={getInitialState:(x)=>{return{globalFilter:void 0,...x}},getDefaultTableOptions:(x)=>{return{onGlobalFilterChange:g1("globalFilter",x),globalFilterFn:"auto",getColumnCanGlobalFilter:(d)=>{var h;let p=(h=x.getCoreRowModel().flatRows[0])===null||h===void 0||(h=h.getAllCellsByColumnId()[d.id])===null||h===void 0?void 0:h.getValue();return typeof p==="string"||typeof p==="number"}}},assignColumnPrototype:(x,d)=>{n("globalFilteringFeature",x,d,{column_getCanGlobalFilter:{fn:(h)=>n1(h)}})},constructTableAPIs:(x)=>{A("globalFilteringFeature",x,{table_getGlobalAutoFilterFn:{fn:()=>T0()},table_getGlobalFilterFn:{fn:()=>c1(x)},table_setGlobalFilter:{fn:(d)=>W0(x,d)},table_resetGlobalFilter:{fn:(d)=>Xd(x,d)}})}};function Zd(x,d,h){if(h.options.filterFromLeafRows)return rh(x,d,h);return zh(x,d,h)}function rh(x,d,h){let p=[],i={},v=h.options.maxLeafRowFilterDepth??100,y=(r,N=0)=>{let q=[];for(let k of r){let z=q1(h,k.id,k.original,k.index,k.depth,void 0,k.parentId);if(z.columnFilters=k.columnFilters,k.subRows.length&&N<v){if(z.subRows=y(k.subRows,N+1),k=z,d(k)&&!z.subRows.length){q.push(k),i[k.id]=k,p.push(k);continue}if(d(k)||z.subRows.length){q.push(k),i[k.id]=k,p.push(k);continue}}else if(k=z,d(k))q.push(k),i[k.id]=k,p.push(k)}return q};return{rows:y(x),flatRows:p,rowsById:i}}function zh(x,d,h){let p=[],i={},v=h.options.maxLeafRowFilterDepth??100,y=(r,N=0)=>{let q=[];for(let k of r)if(d(k)){if(k.subRows.length&&N<v){let z=q1(h,k.id,k.original,k.index,k.depth,void 0,k.parentId);z.subRows=y(k.subRows,N+1),k=z}q.push(k),p.push(k),i[k.id]=k}return q};return{rows:y(x),flatRows:p,rowsById:i}}function J0(){return(x)=>{let d=x;return l({feature:"columnFilteringFeature",table:d,fnName:"table.getFilteredRowModel",memoDeps:()=>{var h,p;return[d.getPreFilteredRowModel(),(h=d.atoms.columnFilters)===null||h===void 0?void 0:h.get(),(p=d.atoms.globalFilter)===null||p===void 0?void 0:p.get()]},fn:()=>qh(d),onAfterUpdate:()=>G1(d)})}}function qh(x){var d,h;let p=x.getPreFilteredRowModel(),i=(d=x.atoms.columnFilters)===null||d===void 0?void 0:d.get(),v=(h=x.atoms.globalFilter)===null||h===void 0?void 0:h.get();if(!p.rows.length||!(i===null||i===void 0?void 0:i.length)&&!v){let Q=p.flatRows;for(let T=0;T<Q.length;T++){let J=Q[T];J.columnFilters={},J.columnFiltersMeta={}}return p}let y=[],r=[];i===null||i===void 0||i.forEach((Q)=>{var T;let J=A1(x,Q.id);if(!J)return;let Z=x1(J);y.push({id:Q.id,filterFn:Z,resolvedValue:((T=Z.resolveFilterValue)===null||T===void 0?void 0:T.call(Z,Q.value))??Q.value})});let N=(i===null||i===void 0?void 0:i.map((Q)=>Q.id))??[],q=c1(x),k=x.getAllLeafColumns().filter((Q)=>n1(Q));if(v&&q&&k.length)N.push("__global__"),k.forEach((Q)=>{var T;r.push({id:Q.id,filterFn:q,resolvedValue:((T=q.resolveFilterValue)===null||T===void 0?void 0:T.call(q,v))??v})});let z=p.flatRows;for(let Q=0;Q<z.length;Q++){let T=z[Q];if(T.columnFilters={},y.length)for(let J=0;J<y.length;J++){let Z=y[J],K=Z.id;T.columnFilters[K]=Z.filterFn(T,K,Z.resolvedValue,(E)=>{!T.columnFiltersMeta?T.columnFiltersMeta={}:T.columnFiltersMeta[K]=E})}if(r.length){for(let J=0;J<r.length;J++){let Z=r[J],K=Z.id;if(Z.filterFn(T,K,Z.resolvedValue,(E)=>{!T.columnFiltersMeta?T.columnFiltersMeta={}:T.columnFiltersMeta[K]=E})){T.columnFilters.__global__=!0;break}}if(T.columnFilters.__global__!==!0)T.columnFilters.__global__=!1}}let W=(Q)=>{for(let T=0;T<N.length;T++)if(Q.columnFilters[N[T]]===!1)return!1;return!0};return Zd(p.rows,W,x)}var E0=class{constructor(x){this._table=null,this._notifier=0,(this.host=x).addController(this)}table(x,d){if(!this._table){let i={...x,features:{coreReactivityFeature:e0(),...x.features},mergeOptions:(v,y)=>{return{...v,...y}}};this._table=y0(i),this._setupSubscriptions()}this._table.setOptions((i)=>({...i,...x}));let h=this._table,p=function(v){let y=(v.source??h.store).get(),r=v.selector!==void 0?v.selector(y):y;if(typeof v.children==="function")return v.children(r);return v.children};return{...this._table,Subscribe:p,FlexRender:_0,get state(){return(d===null||d===void 0?void 0:d(h.store.state))??h.store.state}}}_setupSubscriptions(){if(this._table&&!this._storeSubscription)this._storeSubscription=this._table.store.subscribe(()=>{this._notifier++,this.host.requestUpdate()}),this._optionsSubscription=this._table.optionsStore.subscribe(()=>{this._notifier++,this.host.requestUpdate()})}hostConnected(){this._setupSubscriptions()}hostDisconnected(){var x,d;(x=this._storeSubscription)===null||x===void 0||x.unsubscribe(),this._storeSubscription=void 0,(d=this._optionsSubscription)===null||d===void 0||d.unsubscribe(),this._optionsSubscription=void 0}};var Nh={sampleCount:0,cpuAverage:0,cpuPeak:0,memoryAverage:0,memoryPeak:0,memoryBytesPeak:0};function Bd(x,d){let h=new Map;for(let p of x)h.set(Kd(p.kind,p.id),{...p,resources:{...p.resources},queuePosition:null,orderGroup:2});for(let p of d){let i=Kd(p.kind,p.id),v=h.get(i);h.set(i,{...v??Qh(p),status:p.status,startedAt:p.leasedAt??v?.startedAt,queuePosition:p.position,orderGroup:p.status==="active"?0:1})}return[...h.values()].sort(Th)}function Qh(x){return{kind:x.kind,id:x.id,project:x.project,projectName:x.projectName,status:x.status,command:"",image:"",createdAt:x.acceptedAt,startedAt:x.leasedAt,finishedAt:void 0,exitCode:void 0,queueWaitMillis:void 0,resources:{...Nh}}}function Th(x,d){if(x.orderGroup!==d.orderGroup)return x.orderGroup-d.orderGroup;if(x.orderGroup<2)return(x.queuePosition??0)-(d.queuePosition??0);return Date.parse(d.createdAt)-Date.parse(x.createdAt)||d.id.localeCompare(x.id)}function Kd(x,d){return`${x}:${d}`}var Ud=d1`
  :host {
    --surface: #141417;
    --surface-hover: #1e1e23;
    --canvas-soft: #101013;
    --line: rgba(255,255,255,.09);
    --line-strong: rgba(255,255,255,.15);
    --text: #f5f3ed;
    --text-soft: #aaa7af;
    --text-faint: #74727a;
    --ember: #e38242;
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
  .badge.succeeded, .badge.success, .badge.running, .badge.active { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.cancelled { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
  .position { display: inline-grid; min-width: 20px; height: 20px; margin-left: 6px; place-items: center; border: 1px solid var(--line); border-radius: 5px; }
  .empty { display: grid; min-height: 180px; place-content: center; padding: 32px; text-align: center; }
  .empty strong { font-size: 12px; font-weight: 580; }
  .empty span { display: block; margin-top: 4px; color: var(--text-faint); font-size: 10px; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
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
`;var Wh=v0({columnFilteringFeature:Q0,globalFilteringFeature:Y0,filteredRowModel:J0(),filterFns:z0}),Yh=[{id:"search",accessorFn:(x)=>[x.id,x.command,x.projectName,x.project,x.status,x.kind].join(" "),enableGlobalFilter:!0},{id:"status",accessorFn:(x)=>x.status,filterFn:"equalsString",enableGlobalFilter:!1},{id:"kind",accessorFn:(x)=>x.kind,filterFn:"equalsString",enableGlobalFilter:!1}],Jh={now:""};class jd extends Z1(f){static styles=Ud;tableController=new E0(this);query="";statusFilter="";kindFilter="";rowsFingerprint="";rowsCache=[];render(){let x=this.signal("operations",[]),d=this.signal("queue",[]),h=this.signal("clock",Jh),p=this.rows(x,d),i=[...this.statusFilter?[{id:"status",value:this.statusFilter}]:[],...this.kindFilter?[{id:"kind",value:this.kindFilter}]:[]],y=this.tableController.table({features:Wh,columns:Yh,data:p,getCoreRowModel:N1(),globalFilterFn:"includesString",getColumnCanGlobalFilter:(N)=>N.id==="search",state:{globalFilter:this.query,columnFilters:i}}).getRowModel().rows.map((N)=>N.original),r=Date.parse(h.now);return X`
      <article class="runs-panel">
        <header class="runs-head">
          <div><strong>Runs</strong><span>${y.length===p.length?`${p.length} total`:`${y.length} of ${p.length}`}</span></div>
          <div class="runs-tools">
            <label class="search"><span class="sr-only">Search runs</span><input type="search" placeholder="Search runs…" .value=${this.query} @input=${this.onSearch}></label>
            <label><span class="sr-only">Filter by status</span><select .value=${this.statusFilter} @change=${this.onStatusFilter}>
              <option value="">All statuses</option><option value="active">Active</option><option value="queued">Queued</option><option value="succeeded">Succeeded</option><option value="failed">Failed</option><option value="cancelled">Cancelled</option>
            </select></label>
            <label><span class="sr-only">Filter by kind</span><select .value=${this.kindFilter} @change=${this.onKindFilter}>
              <option value="">Jobs and builds</option><option value="job">Jobs</option><option value="build">Builds</option>
            </select></label>
          </div>
        </header>
        ${y.length===0?this.empty(p.length>0):X`
          <div class="table-wrap"><table>
            <thead><tr><th>Run</th><th>Status</th><th>Project</th><th>Duration</th><th>CPU peak</th><th>Memory peak</th><th>Created</th></tr></thead>
            <tbody>${y.map((N)=>this.row(N,r))}</tbody>
          </table></div>
        `}
      </article>
    `}rows(x,d){let h=JSON.stringify([x,d]);if(h!==this.rowsFingerprint)this.rowsFingerprint=h,this.rowsCache=Bd(x,d);return this.rowsCache}row(x,d){return X`<tr>
      <td class="primary"><a href=${Eh(x.kind,x.id)}><span class="kind-icon">${x.kind==="build"?"◇":"›_"}</span><span><span class="mono">${R(x.id,22)}</span><br><span class="muted">${x.command||Xh(x.kind)}</span></span></a></td>
      <td><span class="badge ${x.status}">${x.status}</span>${x.queuePosition!=null?X`<span class="position">${x.queuePosition}</span>`:""}</td>
      <td>${x.projectName}</td>
      <td class="mono">${K1(x.startedAt,x.finishedAt,d)}</td>
      <td class="mono">${x.resources.sampleCount?H(x.resources.cpuPeak):"—"}</td>
      <td class="mono">${x.resources.sampleCount?o(x.resources.memoryBytesPeak):"—"}</td>
      <td>${z1(x.createdAt,d)}</td>
    </tr>`}empty(x){return X`<div class="empty"><strong>${x?"No matching runs":"No runs yet"}</strong><span>${x?"Try a different search or filter.":"Submit a repository command with autback exec."}</span></div>`}onSearch=(x)=>{this.query=x.currentTarget.value,this.requestUpdate()};onStatusFilter=(x)=>{this.statusFilter=x.currentTarget.value,this.requestUpdate()};onKindFilter=(x)=>{this.kindFilter=x.currentTarget.value,this.requestUpdate()}}function Eh(x,d){return`/app/runs/${encodeURIComponent(x)}/${encodeURIComponent(d)}`}function Xh(x){return x?x[0].toUpperCase()+x.slice(1):"—"}customElements.define("autback-runs-table",jd);var Zh={samples:[],sampleCount:0,activeSampleCount:0,cpuCores:0,memoryTotalBytes:0,diskUsageBytes:0,diskTotalBytes:0,busyRatio:0,cpuAverage:0,cpuPeak:0,memoryAverage:0,memoryPeak:0,memoryBytesPeak:0,queueWaitP95Millis:0},c={session:{user:"",admin:!1,projects:[]},service:{name:"Autback",version:"",control:"CLI only",admission:"One at a time",startedAt:""},worker:{status:"connecting",capacity:"1 operation",activeId:"",updatedAt:""},clock:{now:""},resources:Zh,queue:[],operations:[],operation:null,log:{available:!1,truncated:!1,content:""},audit:[],status:{ready:!1,route:"",message:"Connecting",updatedAt:""}};class Ld extends Z1(f){static styles=$0;get routeKind(){return this.getAttribute("route-kind")||"overview"}get project(){return this.getAttribute("project")||""}get operationID(){return this.getAttribute("operation-id")||""}render(){let x=this.signals();return X`<div class="shell">
      ${this.sidebar(x)}
      <section class="workspace">
        ${this.topbar(x)}
        ${x.status.ready?X`<main class="content" id="content">${this.page(x)}</main>`:X`<main class="loading" id="content"><div class="loader">Opening console</div></main>`}
      </section>
    </div>`}signals(){return{session:this.signal("session",c.session),service:this.signal("service",c.service),worker:this.signal("worker",c.worker),resources:this.signal("resources",c.resources),clock:this.signal("clock",c.clock),queue:this.signal("queue",c.queue),operations:this.signal("operations",c.operations),operation:this.signal("operation",c.operation),log:this.signal("log",c.log),audit:this.signal("audit",c.audit),status:this.signal("status",c.status)}}sidebar(x){return X`<aside class="sidebar" aria-label="Console navigation">
      <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
      <nav class="nav-section" aria-label="Primary">
        <div class="nav-label">Console</div>
        ${this.navLink("/app","overview","Runs","activity")}
        ${this.navLink("/app/audit","audit","Audit log","shield")}
      </nav>
      <nav class="nav-section projects-nav" aria-label="Projects">
        <div class="nav-label">Projects</div>
        ${x.session.projects.map((d)=>X`<a class="nav-link ${this.routeKind==="project"&&this.project===d.slug?"active":""}" href=${`/app/projects/${encodeURIComponent(d.slug)}`}>
          ${D("cube")}<span>${d.name}</span><span class="count">${d.trusts}</span>
        </a>`)}
      </nav>
      <div class="sidebar-foot"><div class="identity"><span class="avatar">${jh(x.session.user)}</span><div>
        <div class="identity-name">${x.session.user||"Connecting"}</div><div class="identity-role">${x.session.admin?"Administrator":"Member"}</div>
      </div></div></div>
    </aside>`}navLink(x,d,h,p){return X`<a class="nav-link ${this.routeKind===d?"active":""}" href=${x}>${D(p)}<span>${h}</span></a>`}topbar(x){let d=this.routeKind==="project"?this.project:this.routeKind==="operation"?R(this.operationID,18):this.routeKind==="audit"?"Audit log":"Runs";return X`<header class="topbar">
      <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${d}</strong></div>
      <div class="live ${x.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${x.status.message}</span></div>
    </header>`}page(x){let d=Date.parse(x.clock.now);switch(this.routeKind){case"project":return this.projectPage(x);case"operation":return this.runPage(x,d);case"audit":return this.auditPage(x,d);default:return this.overview(x)}}overview(x){return X`
      ${this.pageHead("Shared runner","Runs and capacity","See what is running, what is next, and whether the machine has room to do more.")}
      ${this.resourceChart(x.resources,"Runner utilization")}
      ${this.resourceMetrics(x.resources)}
      <autback-runs-table></autback-runs-table>
    `}projectPage(x){let d=x.session.projects.find((h)=>h.slug===this.project);if(!d)return this.notFound("You do not have access to this project.");return X`
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
    `}runPage(x,d){let h=x.operation;if(!h)return this.notFound("You do not have access to this run.");let p=h.command||`${f1(h.kind)} ${R(h.id,18)}`;return X`
      ${this.pageHead(`${f1(h.kind)} run`,p,`${h.projectName} · ${R(h.id,26)}`)}
      ${this.resourceChart(x.resources,"Resource utilization")}
      <section class="metrics" aria-label="Run summary">
        ${F("Status",h.status,f1(h.kind),"pulse")}
        ${F("Queue wait",e(h.queueWaitMillis),"before starting","queue")}
        ${F("Duration",K1(h.startedAt,h.finishedAt,d),h.startedAt?"elapsed time":"not started","clock")}
        ${F("Exit code",h.exitCode==null?"—":String(h.exitCode),h.finishedAt?"result":"pending","terminal")}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel"><header class="panel-head"><div class="panel-title">${D("terminal")}Command</div><span class="badge ${h.status}">${h.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${h.command||"docker buildx build"}</pre></div>
          </article>
          ${this.logPanel(x,h)}
        </div>
        <div class="detail-stack">${this.runSummaryPanel(h,d)}${this.provenancePanel(h)}
          <article class="panel"><header class="panel-head"><div class="panel-title">${D("terminal")}Continue in CLI</div><span class="panel-meta">CLI</span></header>
            <div class="panel-body"><p class="lede">View the full log or inspect this run from your terminal.</p><pre class="command"><span class="prompt">$</span> autback ${h.kind==="job"?"logs":"build status"} ${h.id}</pre></div>
          </article>
        </div>
      </section>
    `}resourceMetrics(x){return X`<section class="metrics" aria-label="Runner capacity summary">
      ${F("Busy",H(x.busyRatio),"of the selected hour","pulse")}
      ${F("CPU while active",H(x.cpuAverage),`${H(x.cpuPeak)} peak`,"cpu")}
      ${F("Memory while active",H(x.memoryAverage),`${o(x.memoryBytesPeak)} peak`,"memory")}
      ${F("Queue wait p95",e(x.queueWaitP95Millis),"recent runs","queue")}
    </section>`}resourceChart(x,d){let h=Md(x.samples,(y)=>y.cpuUtilization),p=Md(x.samples,(y)=>y.memoryUtilization),i=x.samples.at(0),v=x.samples.at(-1);return X`<article class="panel resource-panel">
      <header class="panel-head"><div class="panel-title">${D("activity")}${d}</div>
        <span class="panel-meta">${Kh(x)}</span></header>
      ${x.samples.length<2?Q1("activity","Collecting runner data","Utilization will appear after the next samples arrive."):X`
        <div class="chart-legend">
          <span class="legend cpu"><i></i>CPU <strong>${H(x.cpuAverage)} avg · ${H(x.cpuPeak)} peak</strong></span>
          <span class="legend memory"><i></i>Memory <strong>${H(x.memoryAverage)} avg · ${H(x.memoryPeak)} peak</strong></span>
        </div>
        <div class="resource-chart">
          <svg viewBox="0 0 900 230" preserveAspectRatio="none" role="img" aria-label="CPU and memory utilization over time">
            ${[0,0.25,0.5,0.75,1].map((y)=>M`<line class="grid-line" x1="42" y1=${V1(y)} x2="892" y2=${V1(y)}></line><text class="axis-label" x="4" y=${V1(y)+4}>${Math.round(y*100)}%</text>`)}
            <polyline class="series memory" points=${p}></polyline>
            <polyline class="series cpu" points=${h}></polyline>
          </svg>
          <div class="chart-times"><span>${gd(i?.observedAt)}</span><span>${gd(v?.observedAt)}</span></div>
        </div>
      `}
    </article>`}projectTrends(x){let d=x.filter((i)=>i.startedAt&&i.finishedAt).slice(0,20).reverse(),h=d.map((i)=>Date.parse(i.finishedAt)-Date.parse(i.startedAt)),p=Math.max(...h,1);return X`<section class="trend-grid">
      <article class="panel trend-panel"><header class="panel-head"><div class="panel-title">${D("clock")}Run duration</div><span class="panel-meta">Last ${d.length}</span></header>
        <div class="duration-bars">${h.length===0?Q1("clock","No completed runs","Duration history will appear here."):h.map((i)=>X`<i style=${`height:${Math.max(5,i/p*100)}%`} title=${e(i)}></i>`)}</div>
      </article>
      <article class="panel project-health"><div><span>Success rate</span><strong>${m0(x.map((i)=>i.status))}</strong></div><div><span>Queue wait p95</span><strong>${e(Bh(x.map((i)=>i.queueWaitMillis)))}</strong></div></article>
    </section>`}runSummaryPanel(x,d){return X`<article class="panel"><header class="panel-head"><div class="panel-title">${D("activity")}Run summary</div><span class="panel-meta">${x.resources.sampleCount} samples</span></header>
      <dl class="definition"><dt>Started</dt><dd>${x.startedAt?z1(x.startedAt,d):"—"}</dd><dt>CPU peak</dt><dd>${H(x.resources.cpuPeak)}</dd><dt>Memory peak</dt><dd>${o(x.resources.memoryBytesPeak)}</dd><dt>Queue wait</dt><dd>${e(x.queueWaitMillis)}</dd></dl>
    </article>`}logPanel(x,d){let h=!d.finishedAt&&!["succeeded","failed","cancelled","timed_out","lost"].includes(d.status),p=X`Older lines remain available with <span class="mono">autback logs ${d.id}</span>.`;return X`<article class="panel"><header class="panel-head"><div class="panel-title">${D("terminal")}Output</div><span class="panel-meta">${x.log.available?h?"Following":"Complete":"Unavailable"}</span></header>
      ${x.log.available?X`<pre class="log">${x.log.content||"Waiting for output…"}</pre>${x.log.truncated?X`<div class="log-note">${h?X`Following live output. ${p}`:p}</div>`:B}`:Q1("terminal","No output available",d.kind==="build"?"Build progress remains in the invoking terminal.":"The runner has not produced output yet.")}
    </article>`}provenancePanel(x){let d=x.caches?.length?x.caches.map((h)=>h.name).join(", "):"None declared";return X`<article class="panel"><header class="panel-head"><div class="panel-title">${D("fingerprint")}Provenance</div><span class="panel-meta">Inputs</span></header>
      <dl class="definition"><dt>Run</dt><dd>${x.id}</dd><dt>Project</dt><dd>${x.project}</dd><dt>Image</dt><dd title=${x.image}>${u1(x.image)}</dd><dt>Workdir</dt><dd>${x.workingDirectory||"—"}</dd><dt>Root</dt><dd>${x.rootDigest||"—"}</dd><dt>Caches</dt><dd>${d}</dd></dl>
    </article>`}auditPage(x,d){return X`${this.pageHead("Governance","Audit log","Project, access, image, job, and build activity across Autback.")}
      <article class="panel"><header class="panel-head"><div class="panel-title">${D("shield")}Recent events</div><span class="panel-meta">${x.audit.length} records</span></header>
      ${x.audit.length===0?Q1("shield","No audit events yet","Changes made with the Autback CLI will appear here."):this.auditTable(x.audit,d)}</article>`}auditTable(x,d){return X`<div class="table-wrap"><table><thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${x.map((h)=>X`<tr><td><span class="audit-action">${h.action}</span>${Uh(h)}</td><td>${h.actor}</td><td>${h.project||"Service"}</td><td class="mono">${R(h.target,18)}</td><td>${z1(h.createdAt,d)}</td></tr>`)}</tbody>
    </table></div>`}pageHead(x,d,h){return X`<header class="page-head"><div><p class="eyebrow">${x}</p><h1>${d}</h1><p class="lede">${h}</p></div><div class="read-only">${D("eye")}CLI-managed</div></header>`}notFound(x){return X`${this.pageHead("Not found","Unavailable",x)}<article class="panel">${Q1("shield","Nothing to show","Return to the console overview.")}</article>`}}function D(x){let d={activity:M`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`,clock:M`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,cpu:M`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,cube:M`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`,disk:M`<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/>`,eye:M`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,fingerprint:M`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,memory:M`<rect x="5" y="7" width="14" height="10" rx="2"/><path d="M8 3v4M12 3v4M16 3v4M8 17v4M12 17v4M16 17v4M9 11h6"/>`,pulse:M`<path d="M3 12h4l2-5 4 10 2-5h6"/>`,queue:M`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,shield:M`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`,terminal:M`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`,trend:M`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`};return M`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${d[x]}</svg>`}function F(x,d,h,p){return X`<article class="metric"><div class="metric-top"><span>${x}</span>${D(p)}</div><div class="metric-value">${f1(d)}</div><div class="metric-note">${h}</div></article>`}function Q1(x,d,h){return X`<div class="empty"><div>${D(x)}<strong>${d}</strong><span>${h}</span></div></div>`}function Kh(x){return x.cpuCores?`${x.cpuCores} vCPU · ${o(x.memoryTotalBytes)} · ${o(x.diskTotalBytes)} disk`:"Waiting for capacity data"}function V1(x){return 216-Math.max(0,Math.min(1,x))*196}function Md(x,d){if(x.length===0)return"";return x.map((h,p)=>`${42+p/Math.max(1,x.length-1)*850},${V1(d(h))}`).join(" ")}function gd(x){if(!x)return"—";return new Intl.DateTimeFormat(void 0,{hour:"2-digit",minute:"2-digit"}).format(new Date(x))}function Bh(x){let d=x.filter((h)=>h!=null&&Number.isFinite(h)).sort((h,p)=>h-p);return d.length?d[Math.ceil(d.length*0.95)-1]:void 0}function Uh(x){let d=Object.entries(x.metadata??{}).slice(0,3);return d.length===0?B:X`<div class="metadata">${d.map(([h,p])=>X`<span>${h}=${R(p,28)}</span>`)}</div>`}function f1(x){return x?x[0].toUpperCase()+x.slice(1):"—"}function jh(x){return x.split(/\s+/).filter(Boolean).slice(0,2).map((d)=>d[0]?.toUpperCase()).join("")||"A"}customElements.define("autback-console",Ld);
