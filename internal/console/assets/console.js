var N=globalThis,L=N.ShadowRoot&&(N.ShadyCSS===void 0||N.ShadyCSS.nativeShadow)&&"adoptedStyleSheets"in Document.prototype&&"replace"in CSSStyleSheet.prototype,E=Symbol(),te=new WeakMap;class U{constructor(e,a,t){if(this._$cssResult$=!0,t!==E)throw Error("CSSResult is not constructable. Use `unsafeCSS` or `css` instead.");this.cssText=e,this.t=a}get styleSheet(){let e=this.o,a=this.t;if(L&&e===void 0){let t=a!==void 0&&a.length===1;t&&(e=te.get(a)),e===void 0&&((this.o=e=new CSSStyleSheet).replaceSync(this.cssText),t&&te.set(a,e))}return e}toString(){return this.cssText}}var re=(e)=>new U(typeof e=="string"?e:e+"",void 0,E),G=(e,...a)=>{let t=e.length===1?e[0]:a.reduce((r,i,o)=>r+((n)=>{if(n._$cssResult$===!0)return n.cssText;if(typeof n=="number")return n;throw Error("Value passed to 'css' function must be a 'css' function result: "+n+". Use 'unsafeCSS' to pass non-literal values, but take care to ensure page security.")})(i)+e[o+1],e[0]);return new U(t,e,E)},ie=(e,a)=>{if(L)e.adoptedStyleSheets=a.map((t)=>t instanceof CSSStyleSheet?t:t.styleSheet);else for(let t of a){let r=document.createElement("style"),i=N.litNonce;i!==void 0&&r.setAttribute("nonce",i),r.textContent=t.cssText,e.appendChild(r)}},Q=L?(e)=>e:(e)=>e instanceof CSSStyleSheet?((a)=>{let t="";for(let r of a.cssRules)t+=r.cssText;return re(t)})(e):e;var{is:De,defineProperty:Ie,getOwnPropertyDescriptor:Se,getOwnPropertyNames:Oe,getOwnPropertySymbols:Ve,getPrototypeOf:Ae}=Object,K=globalThis,ne=K.trustedTypes,Fe=ne?ne.emptyScript:"",Ne=K.reactiveElementPolyfillSupport,z=(e,a)=>e,H={toAttribute(e,a){switch(a){case Boolean:e=e?Fe:null;break;case Object:case Array:e=e==null?e:JSON.stringify(e)}return e},fromAttribute(e,a){let t=e;switch(a){case Boolean:t=e!==null;break;case Number:t=e===null?null:Number(e);break;case Object:case Array:try{t=JSON.parse(e)}catch(r){t=null}}return t}},se=(e,a)=>!De(e,a),oe={attribute:!0,type:String,converter:H,reflect:!1,useDefault:!1,hasChanged:se};Symbol.metadata??=Symbol("metadata"),K.litPropertyMetadata??=new WeakMap;class x extends HTMLElement{static addInitializer(e){this._$Ei(),(this.l??=[]).push(e)}static get observedAttributes(){return this.finalize(),this._$Eh&&[...this._$Eh.keys()]}static createProperty(e,a=oe){if(a.state&&(a.attribute=!1),this._$Ei(),this.prototype.hasOwnProperty(e)&&((a=Object.create(a)).wrapped=!0),this.elementProperties.set(e,a),!a.noAccessor){let t=Symbol(),r=this.getPropertyDescriptor(e,t,a);r!==void 0&&Ie(this.prototype,e,r)}}static getPropertyDescriptor(e,a,t){let{get:r,set:i}=Se(this.prototype,e)??{get(){return this[a]},set(o){this[a]=o}};return{get:r,set(o){let n=r?.call(this);i?.call(this,o),this.requestUpdate(e,n,t)},configurable:!0,enumerable:!0}}static getPropertyOptions(e){return this.elementProperties.get(e)??oe}static _$Ei(){if(this.hasOwnProperty(z("elementProperties")))return;let e=Ae(this);e.finalize(),e.l!==void 0&&(this.l=[...e.l]),this.elementProperties=new Map(e.elementProperties)}static finalize(){if(this.hasOwnProperty(z("finalized")))return;if(this.finalized=!0,this._$Ei(),this.hasOwnProperty(z("properties"))){let a=this.properties,t=[...Oe(a),...Ve(a)];for(let r of t)this.createProperty(r,a[r])}let e=this[Symbol.metadata];if(e!==null){let a=litPropertyMetadata.get(e);if(a!==void 0)for(let[t,r]of a)this.elementProperties.set(t,r)}this._$Eh=new Map;for(let[a,t]of this.elementProperties){let r=this._$Eu(a,t);r!==void 0&&this._$Eh.set(r,a)}this.elementStyles=this.finalizeStyles(this.styles)}static finalizeStyles(e){let a=[];if(Array.isArray(e)){let t=new Set(e.flat(1/0).reverse());for(let r of t)a.unshift(Q(r))}else e!==void 0&&a.push(Q(e));return a}static _$Eu(e,a){let t=a.attribute;return t===!1?void 0:typeof t=="string"?t:typeof e=="string"?e.toLowerCase():void 0}constructor(){super(),this._$Ep=void 0,this.isUpdatePending=!1,this.hasUpdated=!1,this._$Em=null,this._$Ev()}_$Ev(){this._$ES=new Promise((e)=>this.enableUpdating=e),this._$AL=new Map,this._$E_(),this.requestUpdate(),this.constructor.l?.forEach((e)=>e(this))}addController(e){(this._$EO??=new Set).add(e),this.renderRoot!==void 0&&this.isConnected&&e.hostConnected?.()}removeController(e){this._$EO?.delete(e)}_$E_(){let e=new Map,a=this.constructor.elementProperties;for(let t of a.keys())this.hasOwnProperty(t)&&(e.set(t,this[t]),delete this[t]);e.size>0&&(this._$Ep=e)}createRenderRoot(){let e=this.shadowRoot??this.attachShadow(this.constructor.shadowRootOptions);return ie(e,this.constructor.elementStyles),e}connectedCallback(){this.renderRoot??=this.createRenderRoot(),this.enableUpdating(!0),this._$EO?.forEach((e)=>e.hostConnected?.())}enableUpdating(e){}disconnectedCallback(){this._$EO?.forEach((e)=>e.hostDisconnected?.())}attributeChangedCallback(e,a,t){this._$AK(e,t)}_$ET(e,a){let t=this.constructor.elementProperties.get(e),r=this.constructor._$Eu(e,t);if(r!==void 0&&t.reflect===!0){let i=(t.converter?.toAttribute!==void 0?t.converter:H).toAttribute(a,t.type);this._$Em=e,i==null?this.removeAttribute(r):this.setAttribute(r,i),this._$Em=null}}_$AK(e,a){let t=this.constructor,r=t._$Eh.get(e);if(r!==void 0&&this._$Em!==r){let i=t.getPropertyOptions(r),o=typeof i.converter=="function"?{fromAttribute:i.converter}:i.converter?.fromAttribute!==void 0?i.converter:H;this._$Em=r;let n=o.fromAttribute(a,i.type);this[r]=n??this._$Ej?.get(r)??n,this._$Em=null}}requestUpdate(e,a,t,r=!1,i){if(e!==void 0){let o=this.constructor;if(r===!1&&(i=this[e]),t??=o.getPropertyOptions(e),!((t.hasChanged??se)(i,a)||t.useDefault&&t.reflect&&i===this._$Ej?.get(e)&&!this.hasAttribute(o._$Eu(e,t))))return;this.C(e,a,t)}this.isUpdatePending===!1&&(this._$ES=this._$EP())}C(e,a,{useDefault:t,reflect:r,wrapped:i},o){t&&!(this._$Ej??=new Map).has(e)&&(this._$Ej.set(e,o??a??this[e]),i!==!0||o!==void 0)||(this._$AL.has(e)||(this.hasUpdated||t||(a=void 0),this._$AL.set(e,a)),r===!0&&this._$Em!==e&&(this._$Eq??=new Set).add(e))}async _$EP(){this.isUpdatePending=!0;try{await this._$ES}catch(a){Promise.reject(a)}let e=this.scheduleUpdate();return e!=null&&await e,!this.isUpdatePending}scheduleUpdate(){return this.performUpdate()}performUpdate(){if(!this.isUpdatePending)return;if(!this.hasUpdated){if(this.renderRoot??=this.createRenderRoot(),this._$Ep){for(let[r,i]of this._$Ep)this[r]=i;this._$Ep=void 0}let t=this.constructor.elementProperties;if(t.size>0)for(let[r,i]of t){let{wrapped:o}=i,n=this[r];o!==!0||this._$AL.has(r)||n===void 0||this.C(r,void 0,i,n)}}let e=!1,a=this._$AL;try{e=this.shouldUpdate(a),e?(this.willUpdate(a),this._$EO?.forEach((t)=>t.hostUpdate?.()),this.update(a)):this._$EM()}catch(t){throw e=!1,this._$EM(),t}e&&this._$AE(a)}willUpdate(e){}_$AE(e){this._$EO?.forEach((a)=>a.hostUpdated?.()),this.hasUpdated||(this.hasUpdated=!0,this.firstUpdated(e)),this.updated(e)}_$EM(){this._$AL=new Map,this.isUpdatePending=!1}get updateComplete(){return this.getUpdateComplete()}getUpdateComplete(){return this._$ES}shouldUpdate(e){return!0}update(e){this._$Eq&&=this._$Eq.forEach((a)=>this._$ET(a,this[a])),this._$EM()}updated(e){}firstUpdated(e){}}x.elementStyles=[],x.shadowRootOptions={mode:"open"},x[z("elementProperties")]=new Map,x[z("finalized")]=new Map,Ne?.({ReactiveElement:x}),(K.reactiveElementVersions??=[]).push("2.1.2");var Y=globalThis,le=(e)=>e,W=Y.trustedTypes,de=W?W.createPolicy("lit-html",{createHTML:(e)=>e}):void 0;var f=`lit$${Math.random().toFixed(9).slice(2)}$`,he="?"+f,Le=`<${he}>`,j=document,M=()=>j.createComment(""),D=(e)=>e===null||typeof e!="object"&&typeof e!="function",J=Array.isArray,Ue=(e)=>J(e)||typeof e?.[Symbol.iterator]=="function";var q=/<(?:(!--|\/[^a-zA-Z])|(\/?[a-zA-Z][^>\s]*)|(\/?$))/g,pe=/-->/g,ce=/>/g,y=RegExp(`>|[ 	
\f\r](?:([^\\s"'>=/]+)([ 	
\f\r]*=[ 	
\f\r]*(?:[^ 	
\f\r"'\`<>=]|("|')|))|$)`,"g"),ue=/'/g,ge=/"/g,ve=/^(?:script|style|textarea|title)$/i,X=(e)=>(a,...t)=>({_$litType$:e,strings:a,values:t}),d=X(1),g=X(2),_e=X(3),R=Symbol.for("lit-noChange"),c=Symbol.for("lit-nothing"),me=new WeakMap,k=j.createTreeWalker(j,129);function xe(e,a){if(!J(e)||!e.hasOwnProperty("raw"))throw Error("invalid template strings array");return de!==void 0?de.createHTML(a):a}var Ke=(e,a)=>{let t=e.length-1,r=[],i,o=a===2?"<svg>":a===3?"<math>":"",n=q;for(let s=0;s<t;s++){let p=e[s],F,l,m=-1,h=0;for(;h<p.length&&(n.lastIndex=h,l=n.exec(p),l!==null);)h=n.lastIndex,n===q?l[1]==="!--"?n=pe:l[1]!==void 0?n=ce:l[2]!==void 0?(ve.test(l[2])&&(i=RegExp("</"+l[2],"g")),n=y):l[3]!==void 0&&(n=y):n===y?l[0]===">"?(n=i??q,m=-1):l[1]===void 0?m=-2:(m=n.lastIndex-l[2].length,F=l[1],n=l[3]===void 0?y:l[3]==='"'?ge:ue):n===ge||n===ue?n=y:n===pe||n===ce?n=q:(n=y,i=void 0);let $=n===y&&e[s+1].startsWith("/>")?" ":"";o+=n===q?p+Le:m>=0?(r.push(F),p.slice(0,m)+"$lit$"+p.slice(m)+f+$):p+f+(m===-2?s:$)}return[xe(e,o+(e[t]||"<?>")+(a===2?"</svg>":a===3?"</math>":"")),r]};class I{constructor({strings:e,_$litType$:a},t){let r;this.parts=[];let i=0,o=0,n=e.length-1,s=this.parts,[p,F]=Ke(e,a);if(this.el=I.createElement(p,t),k.currentNode=this.el.content,a===2||a===3){let l=this.el.content.firstChild;l.replaceWith(...l.childNodes)}for(;(r=k.nextNode())!==null&&s.length<n;){if(r.nodeType===1){if(r.hasAttributes())for(let l of r.getAttributeNames())if(l.endsWith("$lit$")){let m=F[o++],h=r.getAttribute(l).split(f),$=/([.?@])?(.*)/.exec(m);s.push({type:1,index:i,name:$[2],strings:h,ctor:$[1]==="."?be:$[1]==="?"?we:$[1]==="@"?$e:O}),r.removeAttribute(l)}else l.startsWith(f)&&(s.push({type:6,index:i}),r.removeAttribute(l));if(ve.test(r.tagName)){let l=r.textContent.split(f),m=l.length-1;if(m>0){r.textContent=W?W.emptyScript:"";for(let h=0;h<m;h++)r.append(l[h],M()),k.nextNode(),s.push({type:2,index:++i});r.append(l[m],M())}}}else if(r.nodeType===8)if(r.data===he)s.push({type:2,index:i});else{let l=-1;for(;(l=r.data.indexOf(f,l+1))!==-1;)s.push({type:7,index:i}),l+=f.length-1}i++}}static createElement(e,a){let t=j.createElement("template");return t.innerHTML=e,t}}function T(e,a,t=e,r){if(a===R)return a;let i=r!==void 0?t._$Co?.[r]:t._$Cl,o=D(a)?void 0:a._$litDirective$;return i?.constructor!==o&&(i?._$AO?.(!1),o===void 0?i=void 0:(i=new o(e),i._$AT(e,t,r)),r!==void 0?(t._$Co??=[])[r]=i:t._$Cl=i),i!==void 0&&(a=T(e,i._$AS(e,a.values),i,r)),a}class fe{constructor(e,a){this._$AV=[],this._$AN=void 0,this._$AD=e,this._$AM=a}get parentNode(){return this._$AM.parentNode}get _$AU(){return this._$AM._$AU}u(e){let{el:{content:a},parts:t}=this._$AD,r=(e?.creationScope??j).importNode(a,!0);k.currentNode=r;let i=k.nextNode(),o=0,n=0,s=t[0];for(;s!==void 0;){if(o===s.index){let p;s.type===2?p=new S(i,i.nextSibling,this,e):s.type===1?p=new s.ctor(i,s.name,s.strings,this,e):s.type===6&&(p=new ye(i,this,e)),this._$AV.push(p),s=t[++n]}o!==s?.index&&(i=k.nextNode(),o++)}return k.currentNode=j,r}p(e){let a=0;for(let t of this._$AV)t!==void 0&&(t.strings!==void 0?(t._$AI(e,t,a),a+=t.strings.length-2):t._$AI(e[a])),a++}}class S{get _$AU(){return this._$AM?._$AU??this._$Cv}constructor(e,a,t,r){this.type=2,this._$AH=c,this._$AN=void 0,this._$AA=e,this._$AB=a,this._$AM=t,this.options=r,this._$Cv=r?.isConnected??!0}get parentNode(){let e=this._$AA.parentNode,a=this._$AM;return a!==void 0&&e?.nodeType===11&&(e=a.parentNode),e}get startNode(){return this._$AA}get endNode(){return this._$AB}_$AI(e,a=this){e=T(this,e,a),D(e)?e===c||e==null||e===""?(this._$AH!==c&&this._$AR(),this._$AH=c):e!==this._$AH&&e!==R&&this._(e):e._$litType$!==void 0?this.$(e):e.nodeType!==void 0?this.T(e):Ue(e)?this.k(e):this._(e)}O(e){return this._$AA.parentNode.insertBefore(e,this._$AB)}T(e){this._$AH!==e&&(this._$AR(),this._$AH=this.O(e))}_(e){this._$AH!==c&&D(this._$AH)?this._$AA.nextSibling.data=e:this.T(j.createTextNode(e)),this._$AH=e}$(e){let{values:a,_$litType$:t}=e,r=typeof t=="number"?this._$AC(e):(t.el===void 0&&(t.el=I.createElement(xe(t.h,t.h[0]),this.options)),t);if(this._$AH?._$AD===r)this._$AH.p(a);else{let i=new fe(r,this),o=i.u(this.options);i.p(a),this.T(o),this._$AH=i}}_$AC(e){let a=me.get(e.strings);return a===void 0&&me.set(e.strings,a=new I(e)),a}k(e){J(this._$AH)||(this._$AH=[],this._$AR());let a=this._$AH,t,r=0;for(let i of e)r===a.length?a.push(t=new S(this.O(M()),this.O(M()),this,this.options)):t=a[r],t._$AI(i),r++;r<a.length&&(this._$AR(t&&t._$AB.nextSibling,r),a.length=r)}_$AR(e=this._$AA.nextSibling,a){for(this._$AP?.(!1,!0,a);e!==this._$AB;){let t=le(e).nextSibling;le(e).remove(),e=t}}setConnected(e){this._$AM===void 0&&(this._$Cv=e,this._$AP?.(e))}}class O{get tagName(){return this.element.tagName}get _$AU(){return this._$AM._$AU}constructor(e,a,t,r,i){this.type=1,this._$AH=c,this._$AN=void 0,this.element=e,this.name=a,this._$AM=r,this.options=i,t.length>2||t[0]!==""||t[1]!==""?(this._$AH=Array(t.length-1).fill(new String),this.strings=t):this._$AH=c}_$AI(e,a=this,t,r){let i=this.strings,o=!1;if(i===void 0)e=T(this,e,a,0),o=!D(e)||e!==this._$AH&&e!==R,o&&(this._$AH=e);else{let n=e,s,p;for(e=i[0],s=0;s<i.length-1;s++)p=T(this,n[t+s],a,s),p===R&&(p=this._$AH[s]),o||=!D(p)||p!==this._$AH[s],p===c?e=c:e!==c&&(e+=(p??"")+i[s+1]),this._$AH[s]=p}o&&!r&&this.j(e)}j(e){e===c?this.element.removeAttribute(this.name):this.element.setAttribute(this.name,e??"")}}class be extends O{constructor(){super(...arguments),this.type=3}j(e){this.element[this.name]=e===c?void 0:e}}class we extends O{constructor(){super(...arguments),this.type=4}j(e){this.element.toggleAttribute(this.name,!!e&&e!==c)}}class $e extends O{constructor(e,a,t,r,i){super(e,a,t,r,i),this.type=5}_$AI(e,a=this){if((e=T(this,e,a,0)??c)===R)return;let t=this._$AH,r=e===c&&t!==c||e.capture!==t.capture||e.once!==t.once||e.passive!==t.passive,i=e!==c&&(t===c||r);r&&this.element.removeEventListener(this.name,this,t),i&&this.element.addEventListener(this.name,this,e),this._$AH=e}handleEvent(e){typeof this._$AH=="function"?this._$AH.call(this.options?.host??this.element,e):this._$AH.handleEvent(e)}}class ye{constructor(e,a,t){this.element=e,this.type=6,this._$AN=void 0,this._$AM=a,this.options=t}get _$AU(){return this._$AM._$AU}_$AI(e){T(this,e)}}var We=Y.litHtmlPolyfillSupport;We?.(I,S),(Y.litHtmlVersions??=[]).push("3.3.3");var ke=(e,a,t)=>{let r=t?.renderBefore??a,i=r._$litPart$;if(i===void 0){let o=t?.renderBefore??null;r._$litPart$=i=new S(a.insertBefore(M(),o),o,void 0,t??{})}return i._$AI(e),i};var Z=globalThis;class C extends x{constructor(){super(...arguments),this.renderOptions={host:this},this._$Do=void 0}createRenderRoot(){let e=super.createRenderRoot();return this.renderOptions.renderBefore??=e.firstChild,e}update(e){let a=this.render();this.hasUpdated||(this.renderOptions.isConnected=this.isConnected),super.update(e),this._$Do=ke(a,this.renderRoot,this.renderOptions)}connectedCallback(){super.connectedCallback(),this._$Do?.setConnected(!0)}disconnectedCallback(){super.disconnectedCallback(),this._$Do?.setConnected(!1)}render(){return R}}C._$litElement$=!0,C.finalized=!0,Z.litElementHydrateSupport?.({LitElement:C});var Be=Z.litElementPolyfillSupport;Be?.({LitElement:C});(Z.litElementVersions??=[]).push("4.2.2");var je=null;function Re(){let e=new URL("/app/assets/datastar.js",window.location.href).href;return je??=import(e),je}var P=null,Ce=null;function Te(e){class a extends e{#e=null;#a=!1;connectedCallback(){this.#a=!0,super.connectedCallback(),Ee().then(async()=>{if(!this.#a)return;if(this.requestUpdate(),await this.updateComplete,await Ge(),this.#a)this.requestUpdate()})}performUpdate(){if(!this.isUpdatePending)return;let t=P;if(!t){super.performUpdate();return}this.#e?.();let r=!0;this.#e=t.effect(()=>{if(Object.keys(t.root),r){r=!1,super.performUpdate();return}this.requestUpdate()})}disconnectedCallback(){this.#a=!1,this.#e?.(),this.#e=null,super.disconnectedCallback()}signal(t,r){let i=P?.getPath(t);return _(i===void 0?r:i)}}return a}async function Ee(){if(P)return P;return Ce??=Re(),P=await Ce,P}async function Ge(){await Promise.resolve(),await new Promise((e)=>requestAnimationFrame(()=>e()))}function _(e){if(Array.isArray(e))return e.map((a)=>_(a));if(e&&typeof e==="object")return Object.fromEntries(Object.entries(e).map(([a,t])=>[a,_(t)]));return e}var Pe=G`
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
    .grid, .detail-grid { grid-template-columns: 1fr; }
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
    th:nth-child(3), td:nth-child(3), th:nth-child(5), td:nth-child(5) { display: none; }
  }

  @media (max-width: 430px) {
    .metrics { grid-template-columns: 1fr; }
    .project-facts { gap: 12px; }
    .definition { grid-template-columns: 96px minmax(0,1fr); }
  }

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { scroll-behavior: auto !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; transition-duration: .01ms !important; }
  }
`;function b(e,a=12){if(e.length<=a)return e;return`${e.slice(0,a-1)}…`}function ee(e){if(!e)return"Not configured";let a=e.includes("@")?e.split("@").at(-1):e;return a.length>23?`${a.slice(0,16)}…${a.slice(-6)}`:a}function V(e,a=Date.now()){let t=Date.parse(e);if(!Number.isFinite(t))return"—";let r=Math.max(0,Math.round((a-t)/1000));if(r<5)return"now";if(r<60)return`${r}s ago`;let i=Math.floor(r/60);if(i<60)return`${i}m ago`;let o=Math.floor(i/60);if(o<24)return`${o}h ago`;return`${Math.floor(o/24)}d ago`}function ae(e,a,t=Date.now()){if(!e)return"—";let r=Date.parse(e),i=a?Date.parse(a):t;if(!Number.isFinite(r)||!Number.isFinite(i)||i<r)return"—";let o=i-r;if(o<1000)return`${o}ms`;let n=o/1000;if(n<60)return`${n.toFixed(n<10?1:0)}s`;return`${Math.floor(n/60)}m ${Math.floor(n%60)}s`}function ze(e){let a=e.filter((r)=>["succeeded","success","failed","cancelled"].includes(r));if(a.length===0)return"—";let t=a.filter((r)=>r==="succeeded"||r==="success").length;return`${Math.round(t/a.length*100)}%`}var v={session:{user:"",admin:!1,projects:[]},service:{name:"Autback",version:"",control:"CLI only",admission:"Strict FIFO",startedAt:""},worker:{status:"connecting",capacity:"1 operation",activeId:"",updatedAt:""},queue:[],operations:[],operation:null,log:{available:!1,truncated:!1,content:""},audit:[],status:{ready:!1,route:"",message:"Connecting to SQLite",updatedAt:""}};class Me extends Te(C){static styles=Pe;get routeKind(){return this.getAttribute("route-kind")||"overview"}get project(){return this.getAttribute("project")||""}get operationKind(){return this.getAttribute("operation-kind")||""}get operationID(){return this.getAttribute("operation-id")||""}render(){let e=this.signals();return d`
      <div class="shell">
        ${this.sidebar(e)}
        <section class="workspace">
          ${this.topbar(e)}
          ${e.status.ready?d`<main class="content" id="content">${this.page(e)}</main>`:d`<main class="loading" id="content"><div class="loader">Opening live console</div></main>`}
        </section>
      </div>
    `}signals(){return{session:this.signal("session",v.session),service:this.signal("service",v.service),worker:this.signal("worker",v.worker),queue:this.signal("queue",v.queue),operations:this.signal("operations",v.operations),operation:this.signal("operation",v.operation),log:this.signal("log",v.log),audit:this.signal("audit",v.audit),status:this.signal("status",v.status)}}sidebar(e){return d`
      <aside class="sidebar" aria-label="Console navigation">
        <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
        <nav class="nav-section" aria-label="Primary">
          <div class="nav-label">Console</div>
          ${this.navLink("/app","overview","Overview","activity")}
          ${this.navLink("/app/audit","audit","Audit log","shield")}
        </nav>
        <nav class="nav-section projects-nav" aria-label="Projects">
          <div class="nav-label">Projects</div>
          ${e.session.projects.map((a)=>d`
            <a class="nav-link ${this.routeKind==="project"&&this.project===a.slug?"active":""}" href=${`/app/projects/${encodeURIComponent(a.slug)}`}>
              ${u("cube")}<span>${a.name}</span><span class="count">${a.trusts}</span>
            </a>
          `)}
        </nav>
        <div class="sidebar-foot">
          <div class="identity">
            <span class="avatar">${He(e.session.user)}</span>
            <div><div class="identity-name">${e.session.user||"Connecting"}</div><div class="identity-role">${e.session.admin?"Administrator":"Member"}</div></div>
          </div>
        </div>
      </aside>
    `}navLink(e,a,t,r){return d`<a class="nav-link ${this.routeKind===a?"active":""}" href=${e}>${u(r)}<span>${t}</span></a>`}topbar(e){let a=this.routeKind==="project"?this.project:this.routeKind==="operation"?b(this.operationID,18):this.routeKind==="audit"?"Audit log":"Overview";return d`
      <header class="topbar">
        <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${a}</strong></div>
        <div class="live ${e.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${e.status.message}</span></div>
      </header>
    `}page(e){switch(this.routeKind){case"project":return this.projectPage(e);case"operation":return this.operationPage(e);case"audit":return this.auditPage(e);default:return this.overview(e)}}overview(e){let a=e.operations.filter((r)=>["running","active","preparing"].includes(r.status)).length,t=e.queue.filter((r)=>r.status==="queued").length;return d`
      ${this.pageHead("Shared runner","One trusted queue. Every heavy task gets the machine.","Live governance for jobs, builds, projects, and trust. All changes remain CLI-only.")}
      <section class="metrics" aria-label="Service metrics">
        ${w("Worker",e.worker.status,e.worker.capacity,"cpu")}
        ${w("Active",String(a),a===1?"operation using the VM":"operations using the VM","pulse")}
        ${w("Waiting",String(t),t===1?"operation in strict FIFO":"operations in strict FIFO","queue")}
        ${w("Success",ze(e.operations.map((r)=>r.status)),"recent terminal operations","trend")}
      </section>
      <section class="grid">
        ${this.queuePanel(e.queue)}
        <article class="panel">
          <header class="panel-head"><div class="panel-title">${u("cpu")}Worker lease</div><span class="badge ${e.worker.status}">${e.worker.status}</span></header>
          <div class="worker-orbit">
            <div class="worker-core">${u("cpu")}</div>
            <div class="worker-label"><strong>${e.worker.activeId?b(e.worker.activeId,16):"Available"}</strong><span>${e.worker.activeId?"holds the single lease":"next job gets the machine"}</span></div>
          </div>
        </article>
      </section>
      ${this.operationsPanel(e.operations,"Recent operations")}
    `}projectPage(e){let a=e.session.projects.find((t)=>t.slug===this.project);if(!a)return this.notFound("Project is not available to this device.");return d`
      ${this.pageHead("Project",a.name,"Runner image, trust posture, queue position, and recent remote work.")}
      <section class="project-banner">
        <div><div class="project-name">${a.name}</div><div class="project-slug">${a.slug}</div><p class="digest">${ee(a.activeImage)}</p></div>
        <div class="project-facts">
          <div class="fact"><strong>${a.members}</strong><span>Members</span></div>
          <div class="fact"><strong>${a.trusts}</strong><span>GitHub trusts</span></div>
          <div class="fact"><strong>${a.allowImageOverrides?"Allowed":"Pinned"}</strong><span>Image policy</span></div>
        </div>
      </section>
      <section class="grid">
        ${this.queuePanel(e.queue)}
        <article class="panel">
          <header class="panel-head"><div class="panel-title">${u("terminal")}CLI control</div><span class="panel-meta">read only</span></header>
          <div class="panel-body"><p class="lede">The console reflects durable state. Change this project from a trusted terminal.</p><pre class="command"><span class="prompt">$</span> autback image show --project ${a.slug}</pre></div>
        </article>
      </section>
      ${this.operationsPanel(e.operations,"Project operations")}
    `}operationPage(e){let a=e.operation;if(!a)return this.notFound("Operation is not available to this device.");let t=a.command||`${B(a.kind)} ${b(a.id,18)}`;return d`
      ${this.pageHead(`${B(a.kind)} operation`,t,`${a.projectName} · ${b(a.id,26)}`)}
      <section class="metrics" aria-label="Operation metrics">
        ${w("Status",a.status,a.kind,"pulse")}
        ${w("Duration",ae(a.startedAt,a.finishedAt),a.startedAt?"wall-clock execution":"not started","clock")}
        ${w("Exit code",a.exitCode==null?"—":String(a.exitCode),a.finishedAt?"process result":"pending","terminal")}
        ${w("Created",V(a.createdAt),a.projectName,"calendar")}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel">
            <header class="panel-head"><div class="panel-title">${u("terminal")}Command</div><span class="badge ${a.status}">${a.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${a.command||"docker buildx build"}</pre></div>
          </article>
          ${this.logPanel(e,a)}
        </div>
        <div class="detail-stack">
          ${this.provenancePanel(a)}
          <article class="panel">
            <header class="panel-head"><div class="panel-title">${u("terminal")}Continue in CLI</div><span class="panel-meta">authoritative</span></header>
            <div class="panel-body"><p class="lede">Stream the complete log or inspect this operation from any enrolled device.</p><pre class="command"><span class="prompt">$</span> autback ${a.kind==="job"?"logs":"build status"} ${a.id}</pre></div>
          </article>
        </div>
      </section>
    `}auditPage(e){return d`
      ${this.pageHead("Governance","Audit log","An append-only account of project, trust, token, image, job, and build lifecycle events.")}
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${u("shield")}Recent events</div><span class="panel-meta">${e.audit.length} records</span></header>
        ${e.audit.length===0?A("shield","No audit events yet","CLI mutations will appear here."):this.auditTable(e.audit)}
      </article>
    `}pageHead(e,a,t){return d`
      <header class="page-head">
        <div><p class="eyebrow">${e}</p><h1>${a}</h1><p class="lede">${t}</p></div>
        <div class="read-only">${u("eye")}Read-only console · use the CLI to make changes</div>
      </header>
    `}queuePanel(e){return d`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${u("queue")}Strict FIFO queue</div><span class="panel-meta">${e.length} operations</span></header>
        <div class="queue-list">
          ${e.length===0?A("queue","The queue is clear","The next submitted task receives the worker lease."):e.map((a)=>d`
            <div class="queue-row">
              <span class="position">${a.position}</span>
              <div class="queue-main"><a href=${qe(a.kind,a.id)}>${a.id}</a><div class="queue-sub">${a.projectName} · ${V(a.acceptedAt)}</div></div>
              <span class="badge ${a.status}">${a.status}</span>
            </div>
          `)}
        </div>
      </article>
    `}operationsPanel(e,a){return d`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${u("activity")}${a}</div><span class="panel-meta">${e.length} shown</span></header>
        ${e.length===0?A("activity","No operations yet","Submit a repository command with autback exec."):d`
          <div class="table-wrap"><table>
            <thead><tr><th>Operation</th><th>Status</th><th>Project</th><th>Duration</th><th>Created</th></tr></thead>
            <tbody>${e.map((t)=>d`<tr>
              <td class="primary"><a href=${qe(t.kind,t.id)}><span class="kind-icon">${u(t.kind==="build"?"cube":"terminal")}</span><span><span class="mono">${b(t.id,20)}</span><br><span class="muted">${t.command||B(t.kind)}</span></span></a></td>
              <td><span class="badge ${t.status}">${t.status}</span></td>
              <td>${t.projectName}</td>
              <td class="mono">${ae(t.startedAt,t.finishedAt)}</td>
              <td>${V(t.createdAt)}</td>
            </tr>`)}</tbody>
          </table></div>
        `}
      </article>
    `}logPanel(e,a){return d`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${u("terminal")}Log tail</div><span class="panel-meta">${e.log.available?"live projection":"not available"}</span></header>
        ${e.log.available?d`<pre class="log">${e.log.content||"Waiting for output…"}</pre>${e.log.truncated?d`<div class="log-note">Showing the newest 64 KiB. Use <span class="mono">autback logs ${a.id}</span> for the complete stream.</div>`:c}`:A("terminal","No log tail available",a.kind==="build"?"Build progress remains in the invoking terminal.":"The worker has not produced output yet.")}
      </article>
    `}provenancePanel(e){let a=e.caches?.length?e.caches.map((t)=>t.name).join(", "):"None declared";return d`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${u("fingerprint")}Provenance</div><span class="panel-meta">immutable inputs</span></header>
        <dl class="definition">
          <dt>Operation</dt><dd>${e.id}</dd>
          <dt>Project</dt><dd>${e.project}</dd>
          <dt>Image</dt><dd title=${e.image}>${ee(e.image)}</dd>
          <dt>Workdir</dt><dd>${e.workingDirectory||"—"}</dd>
          <dt>Root</dt><dd>${e.rootDigest||"—"}</dd>
          <dt>Caches</dt><dd>${a}</dd>
        </dl>
      </article>
    `}auditTable(e){return d`<div class="table-wrap"><table>
      <thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${e.map((a)=>d`<tr>
        <td><span class="audit-action">${a.action}</span>${Qe(a)}</td>
        <td>${a.actor}</td><td>${a.project||"Service"}</td><td class="mono">${b(a.target,18)}</td><td>${V(a.createdAt)}</td>
      </tr>`)}</tbody>
    </table></div>`}notFound(e){return d`${this.pageHead("Not found","Unavailable",e)}<article class="panel">${A("shield","Nothing to show","Return to the console overview.")}</article>`}}function u(e){let a={activity:g`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`,calendar:g`<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/>`,clock:g`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,cpu:g`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,cube:g`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`,eye:g`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,fingerprint:g`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,pulse:g`<path d="M3 12h4l2-5 4 10 2-5h6"/>`,queue:g`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,shield:g`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`,terminal:g`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`,trend:g`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`};return g`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${a[e]}</svg>`}function w(e,a,t,r){return d`<article class="metric"><div class="metric-top"><span>${e}</span>${u(r)}</div><div class="metric-value">${B(a)}</div><div class="metric-note">${t}</div></article>`}function A(e,a,t){return d`<div class="empty"><div>${u(e)}<strong>${a}</strong><span>${t}</span></div></div>`}function qe(e,a){return`/app/operations/${encodeURIComponent(e)}/${encodeURIComponent(a)}`}function Qe(e){let a=Object.entries(e.metadata??{}).slice(0,3);if(a.length===0)return c;return d`<div class="metadata">${a.map(([t,r])=>d`<span>${t}=${b(r,28)}</span>`)}</div>`}function B(e){return e?e[0].toUpperCase()+e.slice(1):"—"}function He(e){return e.split(/\s+/).filter(Boolean).slice(0,2).map((a)=>a[0]?.toUpperCase()).join("")||"A"}customElements.define("autback-console",Me);
