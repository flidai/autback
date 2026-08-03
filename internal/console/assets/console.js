var W=globalThis,K=W.ShadowRoot&&(W.ShadyCSS===void 0||W.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,H=Symbol(),oe=new WeakMap;class E{constructor(e,a,t){if(this._$cssResult$=!0,t!==H)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=e,this.t=a}get styleSheet(){let e=this.o,a=this.t;if(K&&e===void 0){let t=a!==void 0&&a.length===1;t&&(e=oe.get(a)),e===void 0&&((this.o=e=new CSSStyleSheet).replaceSync(this.cssText),t&&oe.set(a,e))}return e}toString(){return this.cssText}}var se=(e)=>new E(typeof e=="string"?e:e+"",void 0,H),J=(e,...a)=>{let t=e.length===1?e[0]:a.reduce((r,n,o)=>r+((i)=>{if(i._$cssResult$===!0)return i.cssText;if(typeof i=="number")return i;throw Error("Value passed to 'css' function must be a 'css' function result: "+i+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(n)+e[o+1],e[0]);return new E(t,e,H)},le=(e,a)=>{if(K)e.adoptedStyleSheets=a.map((t)=>t instanceof CSSStyleSheet?t:t.styleSheet);else for(let t of a){let r=document.createElement("style"),n=W.litNonce;n!==void 0&&r.setAttribute("nonce",n),r.textContent=t.cssText,e.appendChild(r)}},X=K?(e)=>e:(e)=>e instanceof CSSStyleSheet?((a)=>{let t="";for(let r of a.cssRules)t+=r.cssText;return se(t)})(e):e;var{is:Ne,defineProperty:Be,getOwnPropertyDescriptor:Fe,getOwnPropertyNames:Oe,getOwnPropertySymbols:Le,getPrototypeOf:We}=Object,Q=globalThis,de=Q.trustedTypes,Ke=de?de.emptyScript:"",Ee=Q.reactiveElementPolyfillSupport,A=(e,a)=>e,Z={toAttribute(e,a){switch(a){case Boolean:e=e?Ke:null;break;case Object:case Array:e=e==null?e:JSON.stringify(e)}return e},fromAttribute(e,a){let t=e;switch(a){case Boolean:t=e!==null;break;case Number:t=e===null?null:Number(e);break;case Object:case Array:try{t=JSON.parse(e)}catch(r){t=null}}return t}},ce=(e,a)=>!Ne(e,a),pe={attribute:!0,type:String,converter:Z,reflect:!1,useDefault:!1,hasChanged:ce};Symbol.metadata??=Symbol("metadata"),Q.litPropertyMetadata??=new WeakMap;class b extends HTMLElement{static addInitializer(e){this._$Ei(),(this.l??=[]).push(e)}static get observedAttributes(){return this.finalize(),this._$Eh&&[...this._$Eh.keys()]}static createProperty(e,a=pe){if(a.state&&(a.attribute=!1),this._$Ei(),this.prototype.hasOwnProperty(e)&&((a=Object.create(a)).wrapped=!0),this.elementProperties.set(e,a),!a.noAccessor){let t=Symbol(),r=this.getPropertyDescriptor(e,t,a);r!==void 0&&Be(this.prototype,e,r)}}static getPropertyDescriptor(e,a,t){let{get:r,set:n}=Fe(this.prototype,e)??{get(){return this[a]},set(o){this[a]=o}};return{get:r,set(o){let i=r?.call(this);n?.call(this,o),this.requestUpdate(e,i,t)},configurable:!0,enumerable:!0}}static getPropertyOptions(e){return this.elementProperties.get(e)??pe}static _$Ei(){if(this.hasOwnProperty(A("elementProperties")))return;let e=We(this);e.finalize(),e.l!==void 0&&(this.l=[...e.l]),this.elementProperties=new Map(e.elementProperties)}static finalize(){if(this.hasOwnProperty(A("finalized")))return;if(this.finalized=!0,this._$Ei(),this.hasOwnProperty(A("properties"))){let a=this.properties,t=[...Oe(a),...Le(a)];for(let r of t)this.createProperty(r,a[r])}let e=this[Symbol.metadata];if(e!==null){let a=litPropertyMetadata.get(e);if(a!==void 0)for(let[t,r]of a)this.elementProperties.set(t,r)}this._$Eh=new Map;for(let[a,t]of this.elementProperties){let r=this._$Eu(a,t);r!==void 0&&this._$Eh.set(r,a)}this.elementStyles=this.finalizeStyles(this.styles)}static finalizeStyles(e){let a=[];if(Array.isArray(e)){let t=new Set(e.flat(1/0).reverse());for(let r of t)a.unshift(X(r))}else e!==void 0&&a.push(X(e));return a}static _$Eu(e,a){let t=a.attribute;return t===!1?void 0:typeof t=="string"?t:typeof e=="string"?e.toLowerCase():void 0}constructor(){super(),this._$Ep=void 0,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Em=null,this._$Ev()}_$Ev(){this._$ES=new Promise((e)=>this.enableUpdating=e),this._$AL=new Map,this._$E_(),this.requestUpdate(),this.constructor.l?.forEach((e)=>e(this))}addController(e){(this._$EO??=new Set).add(e),this.renderRoot!==void 0&&this.isConnected&&e.hostConnected?.()}removeController(e){this._$EO?.delete(e)}_$E_(){let e=new Map,a=this.constructor.elementProperties;for(let t of a.keys())this.hasOwnProperty(t)&&(e.set(t,this[t]),delete this[t]);e.size>0&&(this._$Ep=e)}createRenderRoot(){let e=this.shadowRoot??this.attachShadow(this.constructor.shadowRootOptions);return le(e,this.constructor.elementStyles),e}connectedCallback(){this.renderRoot??=this.createRenderRoot(),this.enableUpdating(!0),this._$EO?.forEach((e)=>e.hostConnected?.())}enableUpdating(e){}disconnectedCallback(){this._$EO?.forEach((e)=>e.hostDisconnected?.())}attributeChangedCallback(e,a,t){this._$AK(e,t)}_$ET(e,a){let t=this.constructor.elementProperties.get(e),r=this.constructor._$Eu(e,t);if(r!==void 0&&t.reflect===!0){let n=(t.converter?.toAttribute!==void 0?t.converter:Z).toAttribute(a,t.type);this._$Em=e,n==null?this.removeAttribute(r):this.setAttribute(r,n),this._$Em=null}}_$AK(e,a){let t=this.constructor,r=t._$Eh.get(e);if(r!==void 0&&this._$Em!==r){let n=t.getPropertyOptions(r),o=typeof n.converter=="function"?{fromAttribute:n.converter}:n.converter?.fromAttribute!==void 0?n.converter:Z;this._$Em=r;let i=o.fromAttribute(a,n.type);this[r]=i??this._$Ej?.get(r)??i,this._$Em=null}}requestUpdate(e,a,t,r=!1,n){if(e!==void 0){let o=this.constructor;if(r===!1&&(n=this[e]),t??=o.getPropertyOptions(e),!((t.hasChanged??ce)(n,a)||t.useDefault&&t.reflect&&n===this._$Ej?.get(e)&&!this.hasAttribute(o._$Eu(e,t))))return;this.C(e,a,t)}this.isUpdatePending===!1&&(this._$ES=this._$EP())}C(e,a,{useDefault:t,reflect:r,wrapped:n},o){t&&!(this._$Ej??=new Map).has(e)&&(this._$Ej.set(e,o??a??this[e]),n!==!0||o!==void 0)||(this._$AL.has(e)||(this.hasUpdated||t||(a=void 0),this._$AL.set(e,a)),r===!0&&this._$Em!==e&&(this._$Eq??=new Set).add(e))}async _$EP(){this.isUpdatePending=!0;try{await this._$ES}catch(a){Promise.reject(a)}let e=this.scheduleUpdate();return e!=null&&await e,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){if(!this.isUpdatePending)return;if(!this.hasUpdated){if(this.renderRoot??=this.createRenderRoot(),this._$Ep){for(let[r,n]of this._$Ep)this[r]=n;this._$Ep=void 0}let t=this.constructor.elementProperties;if(t.size>0)for(let[r,n]of t){let{wrapped:o}=n,i=this[r];o!==!0||this._$AL.has(r)||i===void 0||this.C(r,void 0,n,i)}}let e=!1,a=this._$AL;try{e=this.shouldUpdate(a),e?(this.willUpdate(a),this._$EO?.forEach((t)=>t.hostUpdate?.()),this.update(a)):this._$EM()}catch(t){throw e=!1,this._$EM(),t}e&&this._$AE(a)}willUpdate(e){}_$AE(e){this._$EO?.forEach((a)=>a.hostUpdated?.()),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(e)),this.updated(e)}_$EM(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$ES}shouldUpdate(e){return!0}update(e){this._$Eq&&=this._$Eq.forEach((a)=>this._$ET(a,this[a])),this._$EM()}updated(e){}firstUpdated(e){}}b.elementStyles=[],b.shadowRootOptions={mode:"open"},b[A("elementProperties")]=new Map,b[A("finalized")]=new Map,Ee?.({ReactiveElement:b}),(Q.reactiveElementVersions??=[]).push("2.1.2");var _=globalThis,ue=(e)=>e,Y=_.trustedTypes,me=Y?Y.createPolicy("lit-html",{createHTML:(e)=>e}):void 0;var y=`lit$${Math.random().toFixed(9).slice(2)}$`,be="?"+y,Qe=`<${be}>`,M=document,V=()=>M.createComment(""),I=(e)=>e===null||typeof e!="object"&&typeof e!="function",ee=Array.isArray,Ye=(e)=>ee(e)||typeof e?.[Symbol.iterator]=="function";var S=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,ge=/-->/g,he=/>/g,R=RegExp(`>|[ 	
\f\r](?:([^\\s"'>=/]+)([ 	
\f\r]*=[ 	
\f\r]*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`,"g"),xe=/'/g,ve=/"/g,ye=/^(?:script|style|textarea|title)$/i,ae=(e)=>(a,...t)=>({_$litType$:e,strings:a,values:t}),s=ae(1),m=ae(2),la=ae(3),P=Symbol.for("lit-noChange"),c=Symbol.for("lit-nothing"),fe=new WeakMap,C=M.createTreeWalker(M,129);function we(e,a){if(!ee(e)||!e.hasOwnProperty("raw"))throw Error("invalid template strings array");return me!==void 0?me.createHTML(a):a}var Ge=(e,a)=>{let t=e.length-1,r=[],n,o=a===2?"<svg>":a===3?"<math>":"",i=S;for(let l=0;l<t;l++){let p=e[l],L,d,g=-1,v=0;for(;v<p.length&&(i.lastIndex=v,d=i.exec(p),d!==null);)v=i.lastIndex,i===S?d[1]==="!--"?i=ge:d[1]!==void 0?i=he:d[2]!==void 0?(ye.test(d[2])&&(n=RegExp("</"+d[2],"g")),i=R):d[3]!==void 0&&(i=R):i===R?d[0]===">"?(i=n??S,g=-1):d[1]===void 0?g=-2:(g=i.lastIndex-d[2].length,L=d[1],i=d[3]===void 0?R:d[3]==='"'?ve:xe):i===ve||i===xe?i=R:i===ge||i===he?i=S:(i=R,n=void 0);let k=i===R&&e[l+1].startsWith("/>")?" ":"";o+=i===S?p+Qe:g>=0?(r.push(L),p.slice(0,g)+"$lit$"+p.slice(g)+y+k):p+y+(g===-2?l:k)}return[we(e,o+(e[t]||"<?>")+(a===2?"</svg>":a===3?"</math>":"")),r]};class U{constructor({strings:e,_$litType$:a},t){let r;this.parts=[];let n=0,o=0,i=e.length-1,l=this.parts,[p,L]=Ge(e,a);if(this.el=U.createElement(p,t),C.currentNode=this.el.content,a===2||a===3){let d=this.el.content.firstChild;d.replaceWith(...d.childNodes)}for(;(r=C.nextNode())!==null&&l.length<i;){if(r.nodeType===1){if(r.hasAttributes())for(let d of r.getAttributeNames())if(d.endsWith("$lit$")){let g=L[o++],v=r.getAttribute(d).split(y),k=/([.?@])?(.*)/.exec(g);l.push({type:1,index:n,name:k[2],strings:v,ctor:k[1]==="."?ke:k[1]==="?"?Re:k[1]==="@"?Ce:B}),r.removeAttribute(d)}else d.startsWith(y)&&(l.push({type:6,index:n}),r.removeAttribute(d));if(ye.test(r.tagName)){let d=r.textContent.split(y),g=d.length-1;if(g>0){r.textContent=Y?Y.emptyScript:"";for(let v=0;v<g;v++)r.append(d[v],V()),C.nextNode(),l.push({type:2,index:++n});r.append(d[g],V())}}}else if(r.nodeType===8)if(r.data===be)l.push({type:2,index:n});else{let d=-1;for(;(d=r.data.indexOf(y,d+1))!==-1;)l.push({type:7,index:n}),d+=y.length-1}n++}}static createElement(e,a){let t=M.createElement("template");return t.innerHTML=e,t}}function z(e,a,t=e,r){if(a===P)return a;let n=r!==void 0?t._$Co?.[r]:t._$Cl,o=I(a)?void 0:a._$litDirective$;return n?.constructor!==o&&(n?._$AO?.(!1),o===void 0?n=void 0:(n=new o(e),n._$AT(e,t,r)),r!==void 0?(t._$Co??=[])[r]=n:t._$Cl=n),n!==void 0&&(a=z(e,n._$AS(e,a.values),n,r)),a}class $e{constructor(e,a){this._$AV=[],this._$AN=void 0,this._$AD=e,this._$AM=a}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}u(e){let{el:{content:a},parts:t}=this._$AD,r=(e?.creationScope??M).importNode(a,!0);C.currentNode=r;let n=C.nextNode(),o=0,i=0,l=t[0];for(;l!==void 0;){if(o===l.index){let p;l.type===2?p=new N(n,n.nextSibling,this,e):l.type===1?p=new l.ctor(n,l.name,l.strings,this,e):l.type===6&&(p=new Me(n,this,e)),this._$AV.push(p),l=t[++i]}o!==l?.index&&(n=C.nextNode(),o++)}return C.currentNode=M,r}p(e){let a=0;for(let t of this._$AV)t!==void 0&&(t.strings!==void 0?(t._$AI(e,t,a),a+=t.strings.length-2):t._$AI(e[a])),a++}}class N{get _$AU(){return this._$AM?._$AU??this._$Cv}constructor(e,a,t,r){this.type=2,this._$AH=c,this._$AN=void 0,this._$AA=e,this._$AB=a,this._$AM=t,this.options=r,this._$Cv=r?.isConnected??!0}get parentNode(){let e=this._$AA.parentNode,a=this._$AM;return a!==void 0&&e?.nodeType===11&&(e=a.parentNode),e}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(e,a=this){e=z(this,e,a),I(e)?e===c||e==null||e===""?(this._$AH!==c&&this._$AR(),this._$AH=c):e!==this._$AH&&e!==P&&this._(e):e._$litType$!==void 0?this.$(e):e.nodeType!==void 0?this.T(e):Ye(e)?this.k(e):this._(e)}O(e){return this._$AA.parentNode.insertBefore(e,this._$AB)}T(e){this._$AH!==e&&(this._$AR(),this._$AH=this.O(e))}_(e){this._$AH!==c&&I(this._$AH)?this._$AA.nextSibling.data=e:this.T(M.createTextNode(e)),this._$AH=e}$(e){let{values:a,_$litType$:t}=e,r=typeof t=="number"?this._$AC(e):(t.el===void 0&&(t.el=U.createElement(we(t.h,t.h[0]),this.options)),t);if(this._$AH?._$AD===r)this._$AH.p(a);else{let n=new $e(r,this),o=n.u(this.options);n.p(a),this.T(o),this._$AH=n}}_$AC(e){let a=fe.get(e.strings);return a===void 0&&fe.set(e.strings,a=new U(e)),a}k(e){ee(this._$AH)||(this._$AH=[],this._$AR());let a=this._$AH,t,r=0;for(let n of e)r===a.length?a.push(t=new N(this.O(V()),this.O(V()),this,this.options)):t=a[r],t._$AI(n),r++;r<a.length&&(this._$AR(t&&t._$AB.nextSibling,r),a.length=r)}_$AR(e=this._$AA.nextSibling,a){for(this._$AP?.(!1,!0,a);e!==this._$AB;){let t=ue(e).nextSibling;ue(e).remove(),e=t}}setConnected(e){this._$AM===void 0&&(this._$Cv=e,this._$AP?.(e))}}class B{get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}constructor(e,a,t,r,n){this.type=1,this._$AH=c,this._$AN=void 0,this.element=e,this.name=a,this._$AM=r,this.options=n,t.length>2||t[0]!==""||t[1]!==""?(this._$AH=Array(t.length-1).fill(new String),this.strings=t):this._$AH=c}_$AI(e,a=this,t,r){let n=this.strings,o=!1;if(n===void 0)e=z(this,e,a,0),o=!I(e)||e!==this._$AH&&e!==P,o&&(this._$AH=e);else{let i=e,l,p;for(e=n[0],l=0;l<n.length-1;l++)p=z(this,i[t+l],a,l),p===P&&(p=this._$AH[l]),o||=!I(p)||p!==this._$AH[l],p===c?e=c:e!==c&&(e+=(p??"")+n[l+1]),this._$AH[l]=p}o&&!r&&this.j(e)}j(e){e===c?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,e??"")}}class ke extends B{constructor(){super(...arguments),this.type=3}j(e){this.element[this.name]=e===c?void 0:e}}class Re extends B{constructor(){super(...arguments),this.type=4}j(e){this.element.toggleAttribute(this.name,!!e&&e!==c)}}class Ce extends B{constructor(e,a,t,r,n){super(e,a,t,r,n),this.type=5}_$AI(e,a=this){if((e=z(this,e,a,0)??c)===P)return;let t=this._$AH,r=e===c&&t!==c||e.capture!==t.capture||e.once!==t.once||e.passive!==t.passive,n=e!==c&&(t===c||r);r&&this.element.removeEventListener(this.name,this,t),n&&this.element.addEventListener(this.name,this,e),this._$AH=e}handleEvent(e){typeof this._$AH=="function"?this._$AH.call(this.options?.host??this.element,e):this._$AH.handleEvent(e)}}class Me{constructor(e,a,t){this.element=e,this.type=6,this._$AN=void 0,this._$AM=a,this.options=t}get _$AU(){return this._$AM._$AU}_$AI(e){z(this,e)}}var He=_.litHtmlPolyfillSupport;He?.(U,N),(_.litHtmlVersions??=[]).push("3.3.3");var Pe=(e,a,t)=>{let r=t?.renderBefore??a,n=r._$litPart$;if(n===void 0){let o=t?.renderBefore??null;r._$litPart$=n=new N(a.insertBefore(V(),o),o,void 0,t??{})}return n._$AI(e),n};var te=globalThis;class j extends b{constructor(){super(...arguments),this.renderOptions={host:this},this._$Do=void 0}createRenderRoot(){let e=super.createRenderRoot();return this.renderOptions.renderBefore??=e.firstChild,e}update(e){let a=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(e),this._$Do=Pe(a,this.renderRoot,this.renderOptions)}connectedCallback(){super.connectedCallback(),this._$Do?.setConnected(!0)}disconnectedCallback(){super.disconnectedCallback(),this._$Do?.setConnected(!1)}render(){return P}}j._$litElement$=!0,j.finalized=!0,te.litElementHydrateSupport?.({LitElement:j});var Je=te.litElementPolyfillSupport;Je?.({LitElement:j});(te.litElementVersions??=[]).push("4.2.2");var je=null;function Te(){let e=new URL("/app/assets/datastar.js",window.location.href).href;return je??=import(e),je}var D=null,ze=null;function De(e){class a extends e{#e=null;#a=!1;connectedCallback(){this.#a=!0,super.connectedCallback(),Xe().then(async()=>{if(!this.#a)return;if(this.requestUpdate(),await this.updateComplete,await Ze(),this.#a)this.requestUpdate()})}performUpdate(){if(!this.isUpdatePending)return;let t=D;if(!t){super.performUpdate();return}this.#e?.();let r=!0;this.#e=t.effect(()=>{if(Object.keys(t.root),r){r=!1,super.performUpdate();return}this.requestUpdate()})}disconnectedCallback(){this.#a=!1,this.#e?.(),this.#e=null,super.disconnectedCallback()}signal(t,r){let n=D?.getPath(t);return re(n===void 0?r:n)}}return a}async function Xe(){if(D)return D;return ze??=Te(),D=await ze,D}async function Ze(){await Promise.resolve(),await new Promise((e)=>requestAnimationFrame(()=>e()))}function re(e){if(Array.isArray(e))return e.map((a)=>re(a));if(e&&typeof e==="object")return Object.fromEntries(Object.entries(e).map(([a,t])=>[a,re(t)]));return e}var qe=J`
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

  .grid { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(290px, 0.75fr); gap: 14px; margin-bottom: 18px; }
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

  .runner-panel { min-height: 271px; }
  .runner-capacity { display: grid; grid-template-columns: repeat(3, 1fr); border-bottom: 1px solid var(--line); }
  .runner-capacity div { padding: 22px 14px; border-right: 1px solid var(--line); }
  .runner-capacity div:last-child { border-right: 0; }
  .runner-capacity strong { display: block; color: var(--text); font-size: 17px; font-weight: 570; letter-spacing: -.03em; }
  .runner-capacity span { display: block; margin-top: 4px; color: var(--text-faint); font-size: 9px; text-transform: uppercase; letter-spacing: .07em; }
  .runner-now { display: flex; align-items: center; gap: 12px; padding: 22px 18px; }
  .runner-now .live-dot { flex: 0 0 auto; }
  .runner-now strong { display: block; font: 11px var(--mono); }
  .runner-now span:not(.live-dot) { display: block; margin-top: 3px; color: var(--text-faint); font-size: 10px; }

  .worker-orbit { position: relative; display: grid; min-height: 218px; place-items: center; overflow: hidden; }
  .worker-orbit::before, .worker-orbit::after { content: ""; position: absolute; border: 1px solid var(--line); border-radius: 50%; }
  .worker-orbit::before { width: 185px; height: 185px; }
  .worker-orbit::after { width: 128px; height: 128px; border-color: rgba(227,130,66,.2); }
  .worker-core { position: relative; z-index: 1; display: grid; width: 86px; height: 86px; place-items: center; border: 1px solid rgba(227,130,66,.3); border-radius: 50%; background: radial-gradient(circle at 35% 30%, #33251e, #17171a 68%); box-shadow: 0 0 44px rgba(227,130,66,.08); }
  .worker-core svg { width: 25px; color: var(--ember); }
  .worker-label { position: absolute; z-index: 1; bottom: 21px; text-align: center; }
  .worker-label strong { display: block; font-size: 12px; font-weight: 600; }
  .worker-label span { color: var(--text-faint); font-size: 10px; }

  .queue-list { min-height: 218px; }
  .queue-row { display: grid; grid-template-columns: 28px 1fr auto; gap: 11px; align-items: center; min-height: 58px; padding: 8px 18px; border-bottom: 1px solid var(--line); }
  .queue-row:last-child { border-bottom: 0; }
  .position { display: grid; width: 23px; height: 23px; place-items: center; border: 1px solid var(--line); border-radius: 6px; color: var(--text-faint); font: 10px var(--mono); }
  .queue-main { min-width: 0; }
  .queue-main a { display: block; overflow: hidden; font: 11px var(--mono); text-decoration: none; text-overflow: ellipsis; white-space: nowrap; }
  .queue-main a:hover { color: var(--ember); }
  .queue-sub { margin-top: 3px; color: var(--text-faint); font-size: 10px; }

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
  .kind-icon { display: grid; flex: 0 0 auto; width: 26px; height: 26px; place-items: center; border: 1px solid var(--line); border-radius: 6px; background: var(--canvas-soft); }
  .kind-icon svg { width: 13px; color: var(--text-faint); }
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
    .grid, .detail-grid, .trend-grid { grid-template-columns: 1fr; }
    .worker-orbit { min-height: 190px; }
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
`;function f(e,a=12){if(e.length<=a)return e;return`${e.slice(0,a-1)}…`}function ne(e){if(!e)return"Not configured";let a=e.includes("@")?e.split("@").at(-1):e;return a.length>23?`${a.slice(0,16)}…${a.slice(-6)}`:a}function F(e,a=Date.now()){let t=Date.parse(e);if(!Number.isFinite(t))return"—";let r=Math.max(0,Math.round((a-t)/1000));if(r<5)return"now";if(r<60)return`${r}s ago`;let n=Math.floor(r/60);if(n<60)return`${n}m ago`;let o=Math.floor(n/60);if(o<24)return`${o}h ago`;return`${Math.floor(o/24)}d ago`}function ie(e,a,t=Date.now()){if(!e)return"—";let r=Date.parse(e),n=a?Date.parse(a):t;if(!Number.isFinite(r)||!Number.isFinite(n)||n<r)return"—";let o=n-r;if(o<1000)return`${o}ms`;let i=o/1000;if(i<60)return`${i.toFixed(i<10?1:0)}s`;return`${Math.floor(i/60)}m ${Math.floor(i%60)}s`}function Ae(e){let a=e.filter((r)=>["succeeded","success","failed","cancelled"].includes(r));if(a.length===0)return"—";let t=a.filter((r)=>r==="succeeded"||r==="success").length;return`${Math.round(t/a.length*100)}%`}function w(e){if(!Number.isFinite(e)||e<=0)return"—";let a=["B","KB","MB","GB","TB"],t=Math.min(Math.floor(Math.log(e)/Math.log(1024)),a.length-1),r=e/1024**t;return`${r>=10||Number.isInteger(r)?r.toFixed(0):r.toFixed(1)} ${a[t]}`}function h(e){if(!Number.isFinite(e))return"—";return`${Math.round(Math.max(0,Math.min(1,e))*100)}%`}function q(e){if(e==null||!Number.isFinite(e)||e<0)return"—";if(e<1000)return`${Math.round(e)}ms`;let a=e/1000;if(a<60)return`${a.toFixed(a<10?1:0)}s`;return`${Math.floor(a/60)}m ${Math.floor(a%60)}s`}var _e={samples:[],sampleCount:0,activeSampleCount:0,cpuCores:0,memoryTotalBytes:0,diskUsageBytes:0,diskTotalBytes:0,busyRatio:0,cpuAverage:0,cpuPeak:0,memoryAverage:0,memoryPeak:0,memoryBytesPeak:0,queueWaitP95Millis:0},x={session:{user:"",admin:!1,projects:[]},service:{name:"Autback",version:"",control:"CLI only",admission:"One at a time",startedAt:""},worker:{status:"connecting",capacity:"1 operation",activeId:"",updatedAt:""},clock:{now:""},resources:_e,queue:[],operations:[],operation:null,log:{available:!1,truncated:!1,content:""},audit:[],status:{ready:!1,route:"",message:"Connecting",updatedAt:""}};class Ue extends De(j){static styles=qe;get routeKind(){return this.getAttribute("route-kind")||"overview"}get project(){return this.getAttribute("project")||""}get operationID(){return this.getAttribute("operation-id")||""}render(){let e=this.signals();return s`<div class="shell">
      ${this.sidebar(e)}
      <section class="workspace">
        ${this.topbar(e)}
        ${e.status.ready?s`<main class="content" id="content">${this.page(e)}</main>`:s`<main class="loading" id="content"><div class="loader">Opening console</div></main>`}
      </section>
    </div>`}signals(){return{session:this.signal("session",x.session),service:this.signal("service",x.service),worker:this.signal("worker",x.worker),resources:this.signal("resources",x.resources),clock:this.signal("clock",x.clock),queue:this.signal("queue",x.queue),operations:this.signal("operations",x.operations),operation:this.signal("operation",x.operation),log:this.signal("log",x.log),audit:this.signal("audit",x.audit),status:this.signal("status",x.status)}}sidebar(e){return s`<aside class="sidebar" aria-label="Console navigation">
      <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
      <nav class="nav-section" aria-label="Primary">
        <div class="nav-label">Console</div>
        ${this.navLink("/app","overview","Runs","activity")}
        ${this.navLink("/app/audit","audit","Audit log","shield")}
      </nav>
      <nav class="nav-section projects-nav" aria-label="Projects">
        <div class="nav-label">Projects</div>
        ${e.session.projects.map((a)=>s`<a class="nav-link ${this.routeKind==="project"&&this.project===a.slug?"active":""}" href=${`/app/projects/${encodeURIComponent(a.slug)}`}>
          ${u("cube")}<span>${a.name}</span><span class="count">${a.trusts}</span>
        </a>`)}
      </nav>
      <div class="sidebar-foot"><div class="identity"><span class="avatar">${ra(e.session.user)}</span><div>
        <div class="identity-name">${e.session.user||"Connecting"}</div><div class="identity-role">${e.session.admin?"Administrator":"Member"}</div>
      </div></div></div>
    </aside>`}navLink(e,a,t,r){return s`<a class="nav-link ${this.routeKind===a?"active":""}" href=${e}>${u(r)}<span>${t}</span></a>`}topbar(e){let a=this.routeKind==="project"?this.project:this.routeKind==="operation"?f(this.operationID,18):this.routeKind==="audit"?"Audit log":"Runs";return s`<header class="topbar">
      <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${a}</strong></div>
      <div class="live ${e.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${e.status.message}</span></div>
    </header>`}page(e){let a=Date.parse(e.clock.now);switch(this.routeKind){case"project":return this.projectPage(e,a);case"operation":return this.runPage(e,a);case"audit":return this.auditPage(e,a);default:return this.overview(e,a)}}overview(e,a){return s`
      ${this.pageHead("Shared runner","Runs and capacity","See what is running, what is next, and whether the machine has room to do more.")}
      ${this.resourceChart(e.resources,"Runner utilization")}
      ${this.resourceMetrics(e.resources)}
      <section class="grid">
        ${this.jobsPanel(e.queue,a)}
        ${this.runnerPanel(e)}
      </section>
      ${this.runsPanel(e.operations,"Recent runs",a)}
    `}projectPage(e,a){let t=e.session.projects.find((r)=>r.slug===this.project);if(!t)return this.notFound("You do not have access to this project.");return s`
      ${this.pageHead("Project",t.name,"Runs, demand, and runner use for this project.")}
      <section class="project-banner">
        <div><div class="project-name">${t.name}</div><div class="project-slug">${t.slug}</div><p class="digest">${ne(t.activeImage)}</p></div>
        <div class="project-facts">
          <div class="fact"><strong>${t.members}</strong><span>Members</span></div>
          <div class="fact"><strong>${t.trusts}</strong><span>GitHub trusts</span></div>
          <div class="fact"><strong>${t.allowImageOverrides?"Flexible":"Pinned"}</strong><span>Runner image</span></div>
        </div>
      </section>
      ${this.projectTrends(e.operations)}
      ${this.resourceChart(e.resources,"Resource utilization")}
      <section class="grid">${this.jobsPanel(e.queue,a)}${this.runnerPanel(e)}</section>
      ${this.runsPanel(e.operations,"Project runs",a)}
    `}runPage(e,a){let t=e.operation;if(!t)return this.notFound("You do not have access to this run.");let r=t.command||`${O(t.kind)} ${f(t.id,18)}`;return s`
      ${this.pageHead(`${O(t.kind)} run`,r,`${t.projectName} · ${f(t.id,26)}`)}
      ${this.resourceChart(e.resources,"Resource utilization")}
      <section class="metrics" aria-label="Run summary">
        ${$("Status",t.status,O(t.kind),"pulse")}
        ${$("Queue wait",q(t.queueWaitMillis),"before starting","queue")}
        ${$("Duration",ie(t.startedAt,t.finishedAt,a),t.startedAt?"elapsed time":"not started","clock")}
        ${$("Exit code",t.exitCode==null?"—":String(t.exitCode),t.finishedAt?"result":"pending","terminal")}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel"><header class="panel-head"><div class="panel-title">${u("terminal")}Command</div><span class="badge ${t.status}">${t.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${t.command||"docker buildx build"}</pre></div>
          </article>
          ${this.logPanel(e,t)}
        </div>
        <div class="detail-stack">${this.runSummaryPanel(t,a)}${this.provenancePanel(t)}
          <article class="panel"><header class="panel-head"><div class="panel-title">${u("terminal")}Continue in CLI</div><span class="panel-meta">CLI</span></header>
            <div class="panel-body"><p class="lede">View the full log or inspect this run from your terminal.</p><pre class="command"><span class="prompt">$</span> autback ${t.kind==="job"?"logs":"build status"} ${t.id}</pre></div>
          </article>
        </div>
      </section>
    `}resourceMetrics(e){return s`<section class="metrics" aria-label="Runner capacity summary">
      ${$("Busy",h(e.busyRatio),"of the selected hour","pulse")}
      ${$("CPU while active",h(e.cpuAverage),`${h(e.cpuPeak)} peak`,"cpu")}
      ${$("Memory while active",h(e.memoryAverage),`${w(e.memoryBytesPeak)} peak`,"memory")}
		${$("Queue wait p95",q(e.queueWaitP95Millis),"recent runs","queue")}
    </section>`}resourceChart(e,a){let t=Ve(e.samples,(i)=>i.cpuUtilization),r=Ve(e.samples,(i)=>i.memoryUtilization),n=e.samples.at(0),o=e.samples.at(-1);return s`<article class="panel resource-panel">
      <header class="panel-head"><div class="panel-title">${u("activity")}${a}</div>
        <span class="panel-meta">${ea(e)}</span></header>
      ${e.samples.length<2?T("activity","Collecting runner data","Utilization will appear after the next samples arrive."):s`
        <div class="chart-legend">
          <span class="legend cpu"><i></i>CPU <strong>${h(e.cpuAverage)} avg · ${h(e.cpuPeak)} peak</strong></span>
          <span class="legend memory"><i></i>Memory <strong>${h(e.memoryAverage)} avg · ${h(e.memoryPeak)} peak</strong></span>
        </div>
        <div class="resource-chart">
          <svg viewBox="0 0 900 230" preserveAspectRatio="none" role="img" aria-label="CPU and memory utilization over time">
            ${[0,0.25,0.5,0.75,1].map((i)=>m`<line class="grid-line" x1="42" y1=${G(i)} x2="892" y2=${G(i)}></line><text class="axis-label" x="4" y=${G(i)+4}>${Math.round(i*100)}%</text>`)}
            <polyline class="series memory" points=${r}></polyline>
            <polyline class="series cpu" points=${t}></polyline>
          </svg>
          <div class="chart-times"><span>${Ie(n?.observedAt)}</span><span>${Ie(o?.observedAt)}</span></div>
        </div>
      `}
    </article>`}projectTrends(e){let a=e.filter((n)=>n.startedAt&&n.finishedAt).slice(0,20).reverse(),t=a.map((n)=>Date.parse(n.finishedAt)-Date.parse(n.startedAt)),r=Math.max(...t,1);return s`<section class="trend-grid">
      <article class="panel trend-panel"><header class="panel-head"><div class="panel-title">${u("clock")}Run duration</div><span class="panel-meta">Last ${a.length}</span></header>
        <div class="duration-bars">${t.length===0?T("clock","No completed runs","Duration history will appear here."):t.map((n)=>s`<i style=${`height:${Math.max(5,n/r*100)}%`} title=${q(n)}></i>`)}</div>
      </article>
      <article class="panel project-health"><div><span>Success rate</span><strong>${Ae(e.map((n)=>n.status))}</strong></div><div><span>Queue wait p95</span><strong>${q(aa(e.map((n)=>n.queueWaitMillis)))}</strong></div></article>
    </section>`}jobsPanel(e,a){return s`<article class="panel">
      <header class="panel-head"><div class="panel-title">${u("queue")}Jobs</div><span class="panel-meta">${e.length}</span></header>
      <div class="queue-list">${e.length===0?T("queue","No jobs queued or active","The next submitted job can start immediately."):e.map((t)=>s`
        <div class="queue-row"><span class="position">${t.position}</span>
          <div class="queue-main"><a href=${Se(t.kind,t.id)}>${f(t.id,24)}</a><div class="queue-sub">${t.projectName} · ${F(t.acceptedAt,a)}</div></div>
          <span class="badge ${t.status}">${t.status}</span>
        </div>`)}
      </div>
    </article>`}runnerPanel(e){let a=e.resources;return s`<article class="panel runner-panel"><header class="panel-head"><div class="panel-title">${u("cpu")}Runner</div><span class="badge ${e.worker.status}">${e.worker.status}</span></header>
      <div class="runner-capacity"><div><strong>${a.cpuCores||"—"}</strong><span>vCPU</span></div><div><strong>${w(a.memoryTotalBytes)}</strong><span>Memory</span></div><div><strong>${w(a.diskTotalBytes)}</strong><span>Disk</span></div></div>
      <div class="runner-now"><span class="live-dot"></span><div><strong>${e.worker.activeId?f(e.worker.activeId,22):"Ready"}</strong><span>${e.worker.activeId?"active now":"waiting for work"}</span></div></div>
    </article>`}runsPanel(e,a,t){return s`<article class="panel"><header class="panel-head"><div class="panel-title">${u("activity")}${a}</div><span class="panel-meta">${e.length} shown</span></header>
      ${e.length===0?T("activity","No runs yet","Submit a repository command with autback exec."):s`<div class="table-wrap"><table>
        <thead><tr><th>Run</th><th>Status</th><th>Project</th><th>Duration</th><th>CPU peak</th><th>Memory peak</th><th>Created</th></tr></thead>
        <tbody>${e.map((r)=>s`<tr>
          <td class="primary"><a href=${Se(r.kind,r.id)}><span class="kind-icon">${u(r.kind==="build"?"cube":"terminal")}</span><span><span class="mono">${f(r.id,20)}</span><br><span class="muted">${r.command||O(r.kind)}</span></span></a></td>
          <td><span class="badge ${r.status}">${r.status}</span></td><td>${r.projectName}</td>
          <td class="mono">${ie(r.startedAt,r.finishedAt,t)}</td>
          <td class="mono">${r.resources?.sampleCount?h(r.resources.cpuPeak):"—"}</td>
          <td class="mono">${r.resources?.sampleCount?w(r.resources.memoryBytesPeak):"—"}</td>
          <td>${F(r.createdAt,t)}</td>
        </tr>`)}</tbody>
      </table></div>`}
    </article>`}runSummaryPanel(e,a){return s`<article class="panel"><header class="panel-head"><div class="panel-title">${u("activity")}Run summary</div><span class="panel-meta">${e.resources.sampleCount} samples</span></header>
      <dl class="definition"><dt>Started</dt><dd>${e.startedAt?F(e.startedAt,a):"—"}</dd><dt>CPU peak</dt><dd>${h(e.resources.cpuPeak)}</dd><dt>Memory peak</dt><dd>${w(e.resources.memoryBytesPeak)}</dd><dt>Queue wait</dt><dd>${q(e.queueWaitMillis)}</dd></dl>
    </article>`}logPanel(e,a){return s`<article class="panel"><header class="panel-head"><div class="panel-title">${u("terminal")}Output</div><span class="panel-meta">${e.log.available?"Following":"Unavailable"}</span></header>
      ${e.log.available?s`<pre class="log">${e.log.content||"Waiting for output…"}</pre>${e.log.truncated?s`<div class="log-note">Showing the latest output. Use <span class="mono">autback logs ${a.id}</span> for the full log.</div>`:c}`:T("terminal","No output available",a.kind==="build"?"Build progress remains in the invoking terminal.":"The runner has not produced output yet.")}
    </article>`}provenancePanel(e){let a=e.caches?.length?e.caches.map((t)=>t.name).join(", "):"None declared";return s`<article class="panel"><header class="panel-head"><div class="panel-title">${u("fingerprint")}Provenance</div><span class="panel-meta">Inputs</span></header>
      <dl class="definition"><dt>Run</dt><dd>${e.id}</dd><dt>Project</dt><dd>${e.project}</dd><dt>Image</dt><dd title=${e.image}>${ne(e.image)}</dd><dt>Workdir</dt><dd>${e.workingDirectory||"—"}</dd><dt>Root</dt><dd>${e.rootDigest||"—"}</dd><dt>Caches</dt><dd>${a}</dd></dl>
    </article>`}auditPage(e,a){return s`${this.pageHead("Governance","Audit log","Project, access, image, job, and build activity across Autback.")}
      <article class="panel"><header class="panel-head"><div class="panel-title">${u("shield")}Recent events</div><span class="panel-meta">${e.audit.length} records</span></header>
      ${e.audit.length===0?T("shield","No audit events yet","Changes made with the Autback CLI will appear here."):this.auditTable(e.audit,a)}</article>`}auditTable(e,a){return s`<div class="table-wrap"><table><thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${e.map((t)=>s`<tr><td><span class="audit-action">${t.action}</span>${ta(t)}</td><td>${t.actor}</td><td>${t.project||"Service"}</td><td class="mono">${f(t.target,18)}</td><td>${F(t.createdAt,a)}</td></tr>`)}</tbody>
    </table></div>`}pageHead(e,a,t){return s`<header class="page-head"><div><p class="eyebrow">${e}</p><h1>${a}</h1><p class="lede">${t}</p></div><div class="read-only">${u("eye")}CLI-managed</div></header>`}notFound(e){return s`${this.pageHead("Not found","Unavailable",e)}<article class="panel">${T("shield","Nothing to show","Return to the console overview.")}</article>`}}function u(e){let a={activity:m`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`,clock:m`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,cpu:m`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,cube:m`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`,disk:m`<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/>`,eye:m`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,fingerprint:m`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,memory:m`<rect x="5" y="7" width="14" height="10" rx="2"/><path d="M8 3v4M12 3v4M16 3v4M8 17v4M12 17v4M16 17v4M9 11h6"/>`,pulse:m`<path d="M3 12h4l2-5 4 10 2-5h6"/>`,queue:m`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,shield:m`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`,terminal:m`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`,trend:m`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`};return m`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${a[e]}</svg>`}function $(e,a,t,r){return s`<article class="metric"><div class="metric-top"><span>${e}</span>${u(r)}</div><div class="metric-value">${O(a)}</div><div class="metric-note">${t}</div></article>`}function T(e,a,t){return s`<div class="empty"><div>${u(e)}<strong>${a}</strong><span>${t}</span></div></div>`}function Se(e,a){return`/app/runs/${encodeURIComponent(e)}/${encodeURIComponent(a)}`}function ea(e){return e.cpuCores?`${e.cpuCores} vCPU · ${w(e.memoryTotalBytes)} · ${w(e.diskTotalBytes)} disk`:"Waiting for capacity data"}function G(e){return 216-Math.max(0,Math.min(1,e))*196}function Ve(e,a){if(e.length===0)return"";return e.map((t,r)=>`${42+r/Math.max(1,e.length-1)*850},${G(a(t))}`).join(" ")}function Ie(e){if(!e)return"—";return new Intl.DateTimeFormat(void 0,{hour:"2-digit",minute:"2-digit"}).format(new Date(e))}function aa(e){let a=e.filter((t)=>t!=null&&Number.isFinite(t)).sort((t,r)=>t-r);return a.length?a[Math.ceil(a.length*0.95)-1]:void 0}function ta(e){let a=Object.entries(e.metadata??{}).slice(0,3);return a.length===0?c:s`<div class="metadata">${a.map(([t,r])=>s`<span>${t}=${f(r,28)}</span>`)}</div>`}function O(e){return e?e[0].toUpperCase()+e.slice(1):"—"}function ra(e){return e.split(/\s+/).filter(Boolean).slice(0,2).map((a)=>a[0]?.toUpperCase()).join("")||"A"}customElements.define("autback-console",Ue);
