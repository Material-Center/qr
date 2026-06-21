(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const n of document.querySelectorAll('link[rel="modulepreload"]'))s(n);new MutationObserver(n=>{for(const r of n)if(r.type==="childList")for(const d of r.addedNodes)d.tagName==="LINK"&&d.rel==="modulepreload"&&s(d)}).observe(document,{childList:!0,subtree:!0});function a(n){const r={};return n.integrity&&(r.integrity=n.integrity),n.referrerpolicy&&(r.referrerPolicy=n.referrerpolicy),n.crossorigin==="use-credentials"?r.credentials="include":n.crossorigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function s(n){if(n.ep)return;n.ep=!0;const r=a(n);fetch(n.href,r)}})();const I=document.querySelector("#app"),o={page:"dashboard",dashboard:null,jobsPage:null,jobsPageNo:1,jobsPageSize:20,recentJobStatus:"",selectedJobId:0,selectedItems:[],message:"",error:""},$=[["dashboard","\u8FD0\u884C\u9762\u677F"],["settings","\u5168\u5C40\u914D\u7F6E"],["profiles","\u7528\u6237"],["apis","API \u6A21\u677F"],["tasks","\u4EFB\u52A1\u6A21\u677F"],["jobs","\u4EFB\u52A1\u5386\u53F2"]],i=()=>{var e,t;return(t=(e=window.go)==null?void 0:e.main)==null?void 0:t.App};function f(e){const t=Number(e||0);return!Number.isFinite(t)||t<=0?0:Math.round(t/1e6)}function g(e){const t=Number(e||0);return!Number.isFinite(t)||t<=0?0:Math.round(t*1e6)}function k(e){const t=f(e);return t?t%1e3===0?`${t/1e3}s`:`${t}ms`:"0s"}function D(e){if(!e)return"-";const t=new Date(e);return Number.isNaN(t.getTime())?"-":t.toLocaleString()}function l(e){return String(e!=null?e:"").replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;").replaceAll("'","&#39;")}function y(e,t){const a=String(e||"").trim();if(!a)return{};try{const s=JSON.parse(a);if(!s||Array.isArray(s)||typeof s!="object")throw new Error(`${t} \u5FC5\u987B\u662F JSON \u5BF9\u8C61`);return s}catch(s){throw new Error(`${t} \u89E3\u6790\u5931\u8D25: ${s.message}`)}}function h(e,t=0,a=""){const s=['<option value="0">\u4E0D\u4F7F\u7528</option>'];for(const n of e||[])a&&n.APIType!==a||s.push(`<option value="${n.ID}" ${Number(t)===Number(n.ID)?"selected":""}>${l(n.Name||`#${n.ID}`)}</option>`);return s.join("")}function P(e=0){var a;const t=['<option value="0">\u8BF7\u9009\u62E9</option>'];for(const s of((a=o.dashboard)==null?void 0:a.profiles)||[])t.push(`<option value="${s.ID}" ${Number(e)===Number(s.ID)?"selected":""}>${l(s.Name||`#${s.ID}`)}</option>`);return t.join("")}function N(e){return{running:"\u8FD0\u884C\u4E2D",paused:"\u5DF2\u6682\u505C",stopped:"\u5DF2\u505C\u6B62",finished:"\u5DF2\u5B8C\u6210",pending:"\u5F85\u5F00\u59CB",succeeded:"\u6210\u529F",failed:"\u5931\u8D25",created:"\u5DF2\u521B\u5EFA",waiting_code:"\u7B49\u9A8C\u8BC1\u7801",code_submitted:"\u5DF2\u63D0\u4EA4"}[e]||e||"-"}function S(e){return e==="send_code"?"\u53D1\u7801":"\u6536\u7801"}function w(e){return e==="api"?"API":e==="txt"?"TXT":"-"}function b(e,t=!1){o.message=t?"":e,o.error=t?e:"",c()}async function v(e={}){try{o.dashboard=await i().Dashboard(),o.page==="jobs"&&await p(o.jobsPageNo),o.selectedJobId&&((o.page==="jobs"&&o.jobsPage?o.jobsPage.jobs:o.dashboard.jobs).some(s=>Number(s.job.ID)===Number(o.selectedJobId))?o.selectedItems=await i().ListJobItems(o.selectedJobId):(o.selectedJobId=0,o.selectedItems=[])),o.error="",e.keepMessage!==!0&&(o.message="")}catch(t){o.error=String(t)}c()}async function p(e=1){const t=Math.max(1,Number(e||1));o.jobsPage=await i().ListJobsPage(t,o.jobsPageSize),o.jobsPageNo=o.jobsPage.page||t}function c(){const e=o.dashboard;if(!i()){I.innerHTML='<main class="offline">\u8BF7\u5728 Wails \u7A0B\u5E8F\u5185\u6253\u5F00\u3002</main>';return}if(!e){I.innerHTML='<main class="offline">\u6B63\u5728\u52A0\u8F7D...</main>';return}I.innerHTML=`
    <main class="shell">
      <aside class="nav">
        <div class="brand"><span class="brand-icon" title="Phone Task Client">PT</span><span class="version-badge">${l(e.status.version||"-")}</span></div>
        ${$.map(([t,a])=>`<button class="nav-item ${o.page===t?"active":""}" data-page="${t}">${a}</button>`).join("")}
      </aside>
      <section class="content">
        <header class="topbar">
          <div>
            <h1>${l(R())}</h1>
          </div>
          <button class="secondary" data-action="refresh">\u5237\u65B0</button>
        </header>
        ${o.message?`<div class="notice">${l(o.message)}</div>`:""}
        ${o.error?`<div class="notice error">${l(o.error)}</div>`:""}
        ${A()}
      </section>
    </main>
  `}function R(){var e;return((e=$.find(([t])=>t===o.page))==null?void 0:e[1])||"\u8FD0\u884C\u9762\u677F"}function A(){switch(o.page){case"settings":return L();case"profiles":return M();case"apis":return O();case"tasks":return E();case"jobs":return F();default:return J()}}function J(){const e=o.dashboard,t=e.jobs.reduce((a,s)=>(a.total+=s.total,a.pending+=s.pending,a.active+=s.active,a.succeeded+=s.succeeded,a.failed+=s.failed,a),{total:0,pending:0,active:0,succeeded:0,failed:0});return`
    ${x()}
    <section class="metrics">
      <div class="metric"><span>\u4EFB\u52A1\u6570</span><strong>${e.jobs.length}</strong></div>
      <div class="metric"><span>\u5904\u7406\u4E2D</span><strong>${t.pending+t.active}</strong></div>
      <div class="metric"><span>\u6210\u529F</span><strong>${t.succeeded}</strong></div>
      <div class="metric"><span>\u5931\u8D25</span><strong>${t.failed}</strong></div>
    </section>
    ${U()}
    ${C()}
  `}function x(){var a,s,n,r,d;const e=((a=o.dashboard)==null?void 0:a.deviceStats)||{},t=e.lastError?`<div class="device-error">${l(e.lastError)}</div>`:"";return`
    <section class="device-band">
      <div class="device-inline">
        <span>\u5728\u7EBF <strong>${l((s=e.online)!=null?s:0)}</strong></span>
        <span>\u7A7A\u95F2 <strong>${l((n=e.idle)!=null?n:0)}</strong></span>
        <span>\u53EF\u521B\u5EFA <strong>${l((r=e.capacity)!=null?r:0)}</strong></span>
        <span>\u672C\u5730\u4FDD\u7559 <strong>${l((d=e.reserve)!=null?d:0)}</strong></span>
      </div>
      ${t}
    </section>
  `}function L(){const e=o.dashboard.settings||{};return`
    <section class="panel">
      <form id="settings-form" class="form-grid">
        <label>\u670D\u52A1\u5668\u5730\u5740<input name="BaseURL" value="${l(e.BaseURL)}" placeholder="https://server.example"></label>
        <label>\u989D\u5916\u4FDD\u7559\u8BBE\u5907<input name="ReserveDevices" type="number" min="0" value="${l(e.ReserveDevices||0)}"></label>
        <label>\u8F6E\u8BE2\u95F4\u9694 ms<input name="IntervalMS" type="number" min="200" value="${l(f(e.Interval))}"></label>
        <label>\u8BF7\u6C42\u8D85\u65F6 ms<input name="TimeoutMS" type="number" min="1000" value="${l(f(e.Timeout))}"></label>
        <label class="wide">\u65E5\u5FD7\u76EE\u5F55<input name="LogDir" value="${l(e.LogDir)}"></label>
        <div class="form-actions"><button type="submit">\u4FDD\u5B58\u914D\u7F6E</button></div>
      </form>
    </section>
  `}function M(){return`
    <section class="panel">
      <form id="profile-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>\u540D\u79F0<input name="Name" placeholder="sales-1"></label>
        <label>Token<input name="TokenRef" placeholder="openapi token"></label>
        <label>\u521B\u5EFA\u5EF6\u8FDF ms<input name="CreateDelayMS" type="number" min="0" value="0"></label>
        <label>\u8986\u76D6\u670D\u52A1\u5668<input name="BaseURLOverride" placeholder="\u4E3A\u7A7A\u65F6\u4F7F\u7528\u5168\u5C40\u914D\u7F6E"></label>
        <label class="wide">\u5907\u6CE8<input name="Remark"></label>
        <div class="form-actions">
          <button type="submit">\u4FDD\u5B58\u7528\u6237</button>
          <button type="button" class="secondary" data-action="clear-profile">\u6E05\u7A7A</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>Token</th><th>\u5EF6\u8FDF</th><th>\u670D\u52A1\u5668</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${o.dashboard.profiles.map(t=>`
    <tr>
      <td>${t.ID}</td>
      <td>${l(t.Name)}</td>
      <td>${l(t.TokenMask||"****")}</td>
      <td>${k(t.CreateDelay)}</td>
      <td>${l(t.BaseURLOverride||"-")}</td>
      <td><button class="secondary small" data-action="edit-profile" data-id="${t.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("")||m(6)}</tbody></table>
    </section>
  `}function O(){return`
    <section class="panel">
      <form id="api-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>\u540D\u79F0<input name="Name" placeholder="code-api"></label>
        <label>\u7C7B\u578B<select name="APIType"><option value="phone_source">\u624B\u673A\u53F7 API</option><option value="code_source">\u9A8C\u8BC1\u7801 API</option></select></label>
        <label>\u65B9\u6CD5<select name="Method"><option value="GET">GET</option></select></label>
        <label>\u54CD\u5E94<select name="ResponseType"><option value="auto">\u81EA\u52A8</option><option value="text">\u6587\u672C</option><option value="json">JSON</option></select></label>
        <label class="wide">URL<input name="URL" placeholder="https://example.com/code?phone={phone}"></label>
        <label class="wide">Query JSON<textarea name="Query" rows="3" placeholder='{"phone":"{phone}"}'></textarea></label>
        <label class="wide">\u63D0\u53D6\u89C4\u5219 JSON<textarea name="ExtractRule" rows="3" placeholder='{"code":"data.code"}'></textarea></label>
        <div class="form-actions">
          <button type="submit">\u4FDD\u5B58 API \u6A21\u677F</button>
          <button type="button" class="secondary" data-action="clear-api">\u6E05\u7A7A</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u7C7B\u578B</th><th>\u65B9\u6CD5</th><th>URL</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${o.dashboard.apiTemplates.map(t=>`
    <tr>
      <td>${t.ID}</td>
      <td>${l(t.Name)}</td>
      <td>${l(t.APIType)}</td>
      <td>${l(t.Method||"GET")}</td>
      <td class="mono">${l(t.URL)}</td>
      <td><button class="secondary small" data-action="edit-api" data-id="${t.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("")||m(6)}</tbody></table>
    </section>
  `}function E(){const e=o.dashboard.apiTemplates||[],t=o.dashboard.taskTemplates.map(a=>`
    <tr>
      <td>${a.ID}</td>
      <td>${l(a.Name)}</td>
      <td>${S(a.TaskType)}</td>
      <td>${w(a.PhoneSourceType)}</td>
      <td>${l(z(a.ProfileID))}</td>
      <td><button class="secondary small" data-action="edit-task-template" data-id="${a.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("");return`
    <section class="panel">
      <form id="task-template-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>\u540D\u79F0<input name="Name" placeholder="receive-txt"></label>
        <label>\u9ED8\u8BA4\u7528\u6237<select name="ProfileID">${P()}</select></label>
        <label>\u6A21\u5F0F<select name="TaskType"><option value="receive_code">\u6536\u7801</option><option value="send_code">\u53D1\u7801</option></select></label>
        <label>\u624B\u673A\u53F7\u6765\u6E90<select name="PhoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>\u624B\u673A\u53F7 API<select name="PhoneAPITemplateID">${h(e,0,"phone_source")}</select></label>
        <label>\u9A8C\u8BC1\u7801 API<select name="CodeAPITemplateID">${h(e,0,"code_source")}</select></label>
        <label class="wide">\u9ED8\u8BA4 TXT \u76EE\u5F55<input name="DefaultTXTDir"></label>
        <label class="wide">\u5931\u8D25\u5BFC\u51FA\u76EE\u5F55<input name="FailedOutputDir"></label>
        <label class="wide">\u5907\u6CE8<input name="Remark"></label>
        <div class="form-actions">
          <button type="submit">\u4FDD\u5B58\u4EFB\u52A1\u6A21\u677F</button>
          <button type="button" class="secondary" data-action="clear-task-template">\u6E05\u7A7A</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u6A21\u5F0F</th><th>\u6765\u6E90</th><th>\u7528\u6237</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${t||m(6)}</tbody></table>
    </section>
  `}function U(){const e=o.dashboard.apiTemplates||[];return`
    <section class="panel">
      <h2>\u521B\u5EFA\u4EFB\u52A1</h2>
      <form id="job-form" class="form-grid">
        <label>\u540D\u79F0<input name="name" placeholder="\u4E3A\u7A7A\u81EA\u52A8\u751F\u6210"></label>
        <label>\u7528\u6237<select name="profileId">${P()}</select></label>
        <label>\u4EFB\u52A1\u6A21\u677F<select name="taskTemplateId">${Q()}</select></label>
        <label>\u6A21\u5F0F<select name="taskType"><option value="receive_code">\u6536\u7801</option><option value="send_code">\u53D1\u7801</option></select></label>
        <label>\u624B\u673A\u53F7\u6765\u6E90<select name="phoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>\u624B\u673A\u53F7 API<select name="phoneApiTemplateId">${h(e,0,"phone_source")}</select></label>
        <label>\u9A8C\u8BC1\u7801 API<select name="codeApiTemplateId">${h(e,0,"code_source")}</select></label>
        <label class="wide">TXT \u6587\u4EF6\u8DEF\u5F84<input name="inputPath" placeholder="C:\\path\\phones.txt"></label>
        <div class="form-actions"><button type="submit">\u521B\u5EFA\u4EFB\u52A1</button></div>
      </form>
    </section>
  `}function C(){const e=_(o.dashboard.jobs,o.recentJobStatus);return`
    ${j(e,"\u6700\u8FD1\u4EFB\u52A1",B())}
  `}function _(e,t){return t?(e||[]).filter(a=>{var s;return((s=a.job)==null?void 0:s.Status)===t}):e||[]}function B(){return`
    <label class="inline-filter compact-filter">\u72B6\u6001
      <select name="recentJobStatus">
        <option value="" ${o.recentJobStatus===""?"selected":""}>\u5168\u90E8</option>
        <option value="pending" ${o.recentJobStatus==="pending"?"selected":""}>\u5F85\u5F00\u59CB</option>
        <option value="running" ${o.recentJobStatus==="running"?"selected":""}>\u8FD0\u884C\u4E2D</option>
        <option value="paused" ${o.recentJobStatus==="paused"?"selected":""}>\u5DF2\u6682\u505C</option>
        <option value="stopped" ${o.recentJobStatus==="stopped"?"selected":""}>\u5DF2\u505C\u6B62</option>
        <option value="finished" ${o.recentJobStatus==="finished"?"selected":""}>\u5DF2\u5B8C\u6210</option>
      </select>
    </label>
  `}function F(){const e=o.jobsPage;return e?`
    ${j(e.jobs||[],"\u4EFB\u52A1\u5386\u53F2")}
    ${q(e)}
    ${X(e.jobs||[])}
  `:'<section class="panel">\u6B63\u5728\u52A0\u8F7D\u4EFB\u52A1\u5386\u53F2...</section>'}function j(e,t,a=""){const s=e.map(n=>{const r=n.job,d=r.Status==="running"?'<button class="secondary small danger" disabled title="\u6267\u884C\u4E2D\u7684\u4EFB\u52A1\u9700\u8981\u5148\u505C\u6B62">\u5148\u505C\u6B62</button>':`<button class="secondary small danger" data-action="delete-job" data-id="${r.ID}">\u5220\u9664</button>`;return`
      <tr>
        <td>${r.ID}</td>
        <td>${l(r.Name)}</td>
        <td>${S(r.TaskType)}</td>
        <td>${N(r.Status)}</td>
        <td>${n.total}</td>
        <td>${n.pending}</td>
        <td>${n.active}</td>
        <td>${n.succeeded}</td>
        <td>${n.failed}</td>
        <td>${D(r.UpdatedAt)}</td>
        <td class="actions">
          <button class="secondary small" data-action="run-job" data-id="${r.ID}">\u8FD0\u884C</button>
          <button class="secondary small" data-action="pause-job" data-id="${r.ID}">\u6682\u505C</button>
          <button class="secondary small" data-action="resume-job" data-id="${r.ID}">\u7EE7\u7EED</button>
          <button class="secondary small danger" data-action="stop-job" data-id="${r.ID}">\u505C\u6B62</button>
          <button class="secondary small" data-action="show-items" data-id="${r.ID}">\u660E\u7EC6</button>
          <button class="secondary small" data-action="export-success" data-id="${r.ID}">\u5BFC\u51FA\u6210\u529F</button>
          <button class="secondary small" data-action="export-failed" data-id="${r.ID}">\u5BFC\u51FA\u5931\u8D25</button>
          ${d}
        </td>
      </tr>
    `}).join("");return`
    <section class="panel table-panel">
      <div class="panel-heading">
        <h2>${t}</h2>
        ${a}
      </div>
      <table>
        <thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u6A21\u5F0F</th><th>\u72B6\u6001</th><th>\u603B\u6570</th><th>\u5F85\u5904\u7406</th><th>\u5904\u7406\u4E2D</th><th>\u6210\u529F</th><th>\u5931\u8D25</th><th>\u66F4\u65B0\u65F6\u95F4</th><th>\u64CD\u4F5C</th></tr></thead>
        <tbody>${s||m(11)}</tbody>
      </table>
    </section>
  `}function q(e){const t=Number(e.total||0),a=Number(e.pageSize||o.jobsPageSize),s=Number(e.page||o.jobsPageNo),n=Math.max(1,Math.ceil(t/a));return`
    <section class="pager">
      <button class="secondary small" data-action="jobs-prev" ${s<=1?"disabled":""}>\u4E0A\u4E00\u9875</button>
      <span>\u7B2C ${s} / ${n} \u9875\uFF0C\u5171 ${t} \u4E2A\u4EFB\u52A1</span>
      <button class="secondary small" data-action="jobs-next" ${s>=n?"disabled":""}>\u4E0B\u4E00\u9875</button>
    </section>
  `}function X(e=(t=>(t=o.dashboard)==null?void 0:t.jobs)()||[]){const a=['<option value="0">\u8BF7\u9009\u62E9\u4EFB\u52A1</option>'];for(const n of e||[]){const r=n.job;a.push(`<option value="${r.ID}" ${Number(o.selectedJobId)===Number(r.ID)?"selected":""}>#${r.ID} ${l(r.Name||"")}</option>`)}const s=o.selectedItems.map(n=>`
    <tr>
      <td>${n.ID}</td>
      <td>${l(n.Phone)}</td>
      <td>${N(n.Status)}</td>
      <td>${n.RemoteTaskID||"-"}</td>
      <td>${l(n.RemoteStatus||"-")}</td>
      <td>${l(n.VerifyCode||"-")}</td>
      <td class="error-text">${l(n.LastError||"")}</td>
      <td>${D(n.UpdatedAt)}</td>
    </tr>
  `).join("");return`
    <section class="panel table-panel">
      <div class="panel-heading">
        <h2>\u4EFB\u52A1\u660E\u7EC6</h2>
        <label class="inline-filter">\u4EFB\u52A1<select name="jobItemJobId">${a.join("")}</select></label>
      </div>
      <table><thead><tr><th>ID</th><th>\u624B\u673A\u53F7</th><th>\u72B6\u6001</th><th>\u670D\u52A1\u7AEF\u4EFB\u52A1</th><th>\u8FDC\u7AEF\u72B6\u6001</th><th>\u9A8C\u8BC1\u7801</th><th>\u9519\u8BEF</th><th>\u66F4\u65B0\u65F6\u95F4</th></tr></thead><tbody>${s||m(8)}</tbody></table>
    </section>
  `}function Q(e=0){var a;const t=['<option value="0">\u4E0D\u4F7F\u7528</option>'];for(const s of((a=o.dashboard)==null?void 0:a.taskTemplates)||[])t.push(`<option value="${s.ID}" ${Number(e)===Number(s.ID)?"selected":""}>${l(s.Name||`#${s.ID}`)}</option>`);return t.join("")}function z(e){var t;return((t=o.dashboard.profiles.find(a=>Number(a.ID)===Number(e)))==null?void 0:t.Name)||"-"}function m(e){return`<tr><td colspan="${e}" class="empty">\u6682\u65E0\u6570\u636E</td></tr>`}async function G(e){await i().SaveSettings({BaseURL:e.BaseURL.value.trim(),ReserveDevices:Number(e.ReserveDevices.value||0),Interval:g(e.IntervalMS.value),Timeout:g(e.TimeoutMS.value),LogDir:e.LogDir.value.trim()})}async function H(e){const t=e.TokenRef.value.trim();await i().SaveProfile({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),TokenRef:t,TokenMask:t?"":void 0,BaseURLOverride:e.BaseURLOverride.value.trim(),CreateDelay:g(e.CreateDelayMS.value),Remark:e.Remark.value.trim(),Enabled:!0})}async function K(e){await i().SaveAPITemplate({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),APIType:e.APIType.value,Method:e.Method.value,URL:e.URL.value.trim(),Query:y(e.Query.value,"Query JSON"),ExtractRule:y(e.ExtractRule.value,"\u63D0\u53D6\u89C4\u5219 JSON"),ResponseType:e.ResponseType.value,Enabled:!0})}async function V(e){await i().SaveTaskTemplate({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),ProfileID:Number(e.ProfileID.value||0),TaskType:e.TaskType.value,PhoneSourceType:e.PhoneSourceType.value,PhoneAPITemplateID:Number(e.PhoneAPITemplateID.value||0),DefaultTXTDir:e.DefaultTXTDir.value.trim(),CodeSourceType:e.TaskType.value==="receive_code"?"api":"none",CodeAPITemplateID:Number(e.CodeAPITemplateID.value||0),FailedOutputDir:e.FailedOutputDir.value.trim(),Remark:e.Remark.value.trim(),Enabled:!0})}async function W(e){await i().StartJob({name:e.name.value.trim(),profileId:Number(e.profileId.value||0),taskTemplateId:Number(e.taskTemplateId.value||0),taskType:e.taskType.value,phoneSourceType:e.phoneSourceType.value,inputPath:e.inputPath.value.trim(),phoneApiTemplateId:Number(e.phoneApiTemplateId.value||0),codeApiTemplateId:Number(e.codeApiTemplateId.value||0)})}function Y(e){const t=o.dashboard.profiles.find(s=>Number(s.ID)===Number(e)),a=document.querySelector("#profile-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.TokenRef.value=t.TokenRef||"",a.BaseURLOverride.value=t.BaseURLOverride||"",a.CreateDelayMS.value=f(t.CreateDelay),a.Remark.value=t.Remark||"")}function Z(e){const t=o.dashboard.apiTemplates.find(s=>Number(s.ID)===Number(e)),a=document.querySelector("#api-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.APIType.value=t.APIType||"phone_source",a.Method.value=t.Method||"GET",a.ResponseType.value=t.ResponseType||"auto",a.URL.value=t.URL||"",a.Query.value=JSON.stringify(t.Query||{},null,2),a.ExtractRule.value=JSON.stringify(t.ExtractRule||{},null,2))}function ee(e){const t=o.dashboard.taskTemplates.find(s=>Number(s.ID)===Number(e)),a=document.querySelector("#task-template-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.ProfileID.value=t.ProfileID||0,a.TaskType.value=t.TaskType||"receive_code",a.PhoneSourceType.value=t.PhoneSourceType||"txt",a.PhoneAPITemplateID.value=t.PhoneAPITemplateID||0,a.CodeAPITemplateID.value=t.CodeAPITemplateID||0,a.DefaultTXTDir.value=t.DefaultTXTDir||"",a.FailedOutputDir.value=t.FailedOutputDir||"",a.Remark.value=t.Remark||"")}function te(e){const t=o.dashboard.taskTemplates.find(s=>Number(s.ID)===Number(e)),a=document.querySelector("#job-form");!t||!a||(t.ProfileID&&(a.profileId.value=t.ProfileID),a.taskType.value=t.TaskType||"receive_code",a.phoneSourceType.value=t.PhoneSourceType||"txt",a.phoneApiTemplateId.value=t.PhoneAPITemplateID||0,a.codeApiTemplateId.value=t.CodeAPITemplateID||0)}document.addEventListener("click",async e=>{var r,d,T;const t=e.target.closest("button");if(!t)return;const a=t.dataset.action,s=t.dataset.page;if(s){if(o.page=s,s==="jobs"){p(o.jobsPageNo).then(()=>c()).catch(u=>b(String(u),!0));return}c();return}if(!a)return;const n=Number(t.dataset.id||0);try{if(a==="refresh"&&await v(),a==="edit-profile"){Y(n);return}if(a==="clear-profile"){(r=document.querySelector("#profile-form"))==null||r.reset();return}if(a==="edit-api"){Z(n);return}if(a==="clear-api"){(d=document.querySelector("#api-form"))==null||d.reset();return}if(a==="edit-task-template"){ee(n);return}if(a==="clear-task-template"){(T=document.querySelector("#task-template-form"))==null||T.reset();return}if(a==="run-job"&&await i().RunJob(n),a==="pause-job"&&await i().PauseJob(n),a==="resume-job"&&await i().ResumeJob(n),a==="stop-job"&&await i().StopJob(n),a==="delete-job"){if(!confirm(`\u786E\u8BA4\u5220\u9664\u4EFB\u52A1 #${n}\uFF1F`))return;await i().DeleteJob(n),o.selectedJobId===n&&(o.selectedJobId=0,o.selectedItems=[]),o.page==="jobs"&&await p(o.jobsPageNo)}if(a==="jobs-prev"){await p(o.jobsPageNo-1),c();return}if(a==="jobs-next"){await p(o.jobsPageNo+1),c();return}if(a==="show-items"&&(o.selectedJobId=n,o.selectedItems=await i().ListJobItems(n),o.page="jobs",await p(o.jobsPageNo)),a==="export-failed"){const u=prompt("\u5931\u8D25\u6587\u4EF6\u8F93\u51FA\u8DEF\u5F84",`failed-${n}.txt`);u&&await i().ExportFailed(n,u)}if(a==="export-success"){const u=prompt("\u6210\u529F\u6587\u4EF6\u8F93\u51FA\u8DEF\u5F84",`success-${n}.txt`);u&&await i().ExportSucceeded(n,u)}["run-job","pause-job","resume-job","stop-job","delete-job","export-failed","export-success"].includes(a)?(await v({keepMessage:!0}),b("\u64CD\u4F5C\u5DF2\u63D0\u4EA4")):c()}catch(u){b(String(u),!0)}});document.addEventListener("change",e=>{var t,a,s;if(((t=e.target)==null?void 0:t.name)==="taskTemplateId"&&e.target.closest("#job-form")&&te(e.target.value),((a=e.target)==null?void 0:a.name)==="recentJobStatus"&&(o.recentJobStatus=e.target.value,c()),((s=e.target)==null?void 0:s.name)==="jobItemJobId"){const n=Number(e.target.value||0);if(o.selectedJobId=n,n<=0){o.selectedItems=[],c();return}i().ListJobItems(n).then(r=>{o.selectedItems=r,c()}).catch(r=>{b(String(r),!0)})}});document.addEventListener("submit",async e=>{e.preventDefault();try{e.target.id==="settings-form"&&await G(e.target),e.target.id==="profile-form"&&await H(e.target),e.target.id==="api-form"&&await K(e.target),e.target.id==="task-template-form"&&await V(e.target),e.target.id==="job-form"&&await W(e.target),await v({keepMessage:!0}),b("\u4FDD\u5B58\u6210\u529F")}catch(t){b(String(t),!0)}});v();
