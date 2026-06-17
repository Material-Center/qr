(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const l of document.querySelectorAll('link[rel="modulepreload"]'))o(l);new MutationObserver(l=>{for(const r of l)if(r.type==="childList")for(const d of r.addedNodes)d.tagName==="LINK"&&d.rel==="modulepreload"&&o(d)}).observe(document,{childList:!0,subtree:!0});function a(l){const r={};return l.integrity&&(r.integrity=l.integrity),l.referrerpolicy&&(r.referrerPolicy=l.referrerpolicy),l.crossorigin==="use-credentials"?r.credentials="include":l.crossorigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function o(l){if(l.ep)return;l.ep=!0;const r=a(l);fetch(l.href,r)}})();const v=document.querySelector("#app"),n={page:"dashboard",dashboard:null,selectedJobId:0,selectedItems:[],message:"",error:""},D=[["dashboard","\u8FD0\u884C\u9762\u677F"],["settings","\u5168\u5C40\u914D\u7F6E"],["profiles","\u7528\u6237"],["apis","API \u6A21\u677F"],["tasks","\u4EFB\u52A1\u6A21\u677F"],["jobs","\u4EFB\u52A1\u5386\u53F2"]],i=()=>{var e,t;return(t=(e=window.go)==null?void 0:e.main)==null?void 0:t.App};function p(e){const t=Number(e||0);return!Number.isFinite(t)||t<=0?0:Math.round(t/1e6)}function T(e){const t=Number(e||0);return!Number.isFinite(t)||t<=0?0:Math.round(t*1e6)}function S(e){const t=p(e);return t?t%1e3===0?`${t/1e3}s`:`${t}ms`:"0s"}function $(e){if(!e)return"-";const t=new Date(e);return Number.isNaN(t.getTime())?"-":t.toLocaleString()}function s(e){return String(e!=null?e:"").replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;").replaceAll("'","&#39;")}function y(e,t){const a=String(e||"").trim();if(!a)return{};try{const o=JSON.parse(a);if(!o||Array.isArray(o)||typeof o!="object")throw new Error(`${t} \u5FC5\u987B\u662F JSON \u5BF9\u8C61`);return o}catch(o){throw new Error(`${t} \u89E3\u6790\u5931\u8D25: ${o.message}`)}}function m(e,t=0,a=""){const o=['<option value="0">\u4E0D\u4F7F\u7528</option>'];for(const l of e||[])a&&l.APIType!==a||o.push(`<option value="${l.ID}" ${Number(t)===Number(l.ID)?"selected":""}>${s(l.Name||`#${l.ID}`)}</option>`);return o.join("")}function g(e=0){var a;const t=['<option value="0">\u8BF7\u9009\u62E9</option>'];for(const o of((a=n.dashboard)==null?void 0:a.profiles)||[])t.push(`<option value="${o.ID}" ${Number(e)===Number(o.ID)?"selected":""}>${s(o.Name||`#${o.ID}`)}</option>`);return t.join("")}function N(e){return{running:"\u8FD0\u884C\u4E2D",paused:"\u5DF2\u6682\u505C",stopped:"\u5DF2\u505C\u6B62",finished:"\u5DF2\u5B8C\u6210",pending:"\u5F85\u5F00\u59CB",succeeded:"\u6210\u529F",failed:"\u5931\u8D25",created:"\u5DF2\u521B\u5EFA",waiting_code:"\u7B49\u9A8C\u8BC1\u7801",code_submitted:"\u5DF2\u63D0\u4EA4"}[e]||e||"-"}function P(e){return e==="send_code"?"\u53D1\u7801":"\u6536\u7801"}function A(e){return e==="api"?"API":e==="txt"?"TXT":"-"}function b(e,t=!1){n.message=t?"":e,n.error=t?e:"",f()}async function h(e={}){try{n.dashboard=await i().Dashboard(),n.error="",e.keepMessage!==!0&&(n.message="")}catch(t){n.error=String(t)}f()}function f(){const e=n.dashboard;if(!i()){v.innerHTML='<main class="offline">\u8BF7\u5728 Wails \u7A0B\u5E8F\u5185\u6253\u5F00\u3002</main>';return}if(!e){v.innerHTML='<main class="offline">\u6B63\u5728\u52A0\u8F7D...</main>';return}v.innerHTML=`
    <main class="shell">
      <aside class="nav">
        <div class="brand">Phone Task Client</div>
        ${D.map(([t,a])=>`<button class="nav-item ${n.page===t?"active":""}" data-page="${t}">${a}</button>`).join("")}
      </aside>
      <section class="content">
        <header class="topbar">
          <div>
            <h1>${s(R())}</h1>
            <p>${s(e.status.description)}</p>
          </div>
          <button class="secondary" data-action="refresh">\u5237\u65B0</button>
        </header>
        ${n.message?`<div class="notice">${s(n.message)}</div>`:""}
        ${n.error?`<div class="notice error">${s(n.error)}</div>`:""}
        ${w()}
      </section>
    </main>
  `}function R(){var e;return((e=D.find(([t])=>t===n.page))==null?void 0:e[1])||"\u8FD0\u884C\u9762\u677F"}function w(){switch(n.page){case"settings":return j();case"profiles":return x();case"apis":return O();case"tasks":return M();case"jobs":return k(!0);default:return L()}}function L(){const e=n.dashboard,t=e.jobs.reduce((a,o)=>(a.total+=o.total,a.pending+=o.pending,a.active+=o.active,a.succeeded+=o.succeeded,a.failed+=o.failed,a),{total:0,pending:0,active:0,succeeded:0,failed:0});return`
    <section class="metrics">
      <div class="metric"><span>\u4EFB\u52A1\u6570</span><strong>${e.jobs.length}</strong></div>
      <div class="metric"><span>\u5904\u7406\u4E2D</span><strong>${t.pending+t.active}</strong></div>
      <div class="metric"><span>\u6210\u529F</span><strong>${t.succeeded}</strong></div>
      <div class="metric"><span>\u5931\u8D25</span><strong>${t.failed}</strong></div>
    </section>
    ${J()}
    ${k(!1)}
  `}function j(){const e=n.dashboard.settings||{};return`
    <section class="panel">
      <form id="settings-form" class="form-grid">
        <label>\u670D\u52A1\u5668\u5730\u5740<input name="BaseURL" value="${s(e.BaseURL)}" placeholder="https://server.example"></label>
        <label>\u4FDD\u7559\u8BBE\u5907<input name="ReserveDevices" type="number" min="0" value="${s(e.ReserveDevices||0)}"></label>
        <label>\u8F6E\u8BE2\u95F4\u9694 ms<input name="IntervalMS" type="number" min="200" value="${s(p(e.Interval))}"></label>
        <label>\u8BF7\u6C42\u8D85\u65F6 ms<input name="TimeoutMS" type="number" min="1000" value="${s(p(e.Timeout))}"></label>
        <label class="wide">\u65E5\u5FD7\u76EE\u5F55<input name="LogDir" value="${s(e.LogDir)}"></label>
        <div class="form-actions"><button type="submit">\u4FDD\u5B58\u914D\u7F6E</button></div>
      </form>
    </section>
  `}function x(){return`
    <section class="panel">
      <form id="profile-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>\u540D\u79F0<input name="Name" placeholder="sales-1"></label>
        <label>Token<input name="TokenRef" placeholder="openapi token"></label>
        <label>create_delay ms<input name="CreateDelayMS" type="number" min="0" value="0"></label>
        <label>\u8986\u76D6\u670D\u52A1\u5668<input name="BaseURLOverride" placeholder="\u4E3A\u7A7A\u65F6\u4F7F\u7528\u5168\u5C40\u914D\u7F6E"></label>
        <label class="wide">\u5907\u6CE8<input name="Remark"></label>
        <div class="form-actions">
          <button type="submit">\u4FDD\u5B58\u7528\u6237</button>
          <button type="button" class="secondary" data-action="clear-profile">\u6E05\u7A7A</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>Token</th><th>\u5EF6\u8FDF</th><th>\u670D\u52A1\u5668</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${n.dashboard.profiles.map(t=>`
    <tr>
      <td>${t.ID}</td>
      <td>${s(t.Name)}</td>
      <td>${s(t.TokenMask||"****")}</td>
      <td>${S(t.CreateDelay)}</td>
      <td>${s(t.BaseURLOverride||"-")}</td>
      <td><button class="secondary small" data-action="edit-profile" data-id="${t.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("")||u(6)}</tbody></table>
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
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u7C7B\u578B</th><th>\u65B9\u6CD5</th><th>URL</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${n.dashboard.apiTemplates.map(t=>`
    <tr>
      <td>${t.ID}</td>
      <td>${s(t.Name)}</td>
      <td>${s(t.APIType)}</td>
      <td>${s(t.Method||"GET")}</td>
      <td class="mono">${s(t.URL)}</td>
      <td><button class="secondary small" data-action="edit-api" data-id="${t.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("")||u(6)}</tbody></table>
    </section>
  `}function M(){const e=n.dashboard.apiTemplates||[],t=n.dashboard.taskTemplates.map(a=>`
    <tr>
      <td>${a.ID}</td>
      <td>${s(a.Name)}</td>
      <td>${P(a.TaskType)}</td>
      <td>${A(a.PhoneSourceType)}</td>
      <td>${s(_(a.ProfileID))}</td>
      <td><button class="secondary small" data-action="edit-task-template" data-id="${a.ID}">\u7F16\u8F91</button></td>
    </tr>
  `).join("");return`
    <section class="panel">
      <form id="task-template-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>\u540D\u79F0<input name="Name" placeholder="receive-txt"></label>
        <label>\u9ED8\u8BA4\u7528\u6237<select name="ProfileID">${g()}</select></label>
        <label>\u6A21\u5F0F<select name="TaskType"><option value="receive_code">\u6536\u7801</option><option value="send_code">\u53D1\u7801</option></select></label>
        <label>\u624B\u673A\u53F7\u6765\u6E90<select name="PhoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>\u624B\u673A\u53F7 API<select name="PhoneAPITemplateID">${m(e,0,"phone_source")}</select></label>
        <label>\u9A8C\u8BC1\u7801 API<select name="CodeAPITemplateID">${m(e,0,"code_source")}</select></label>
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
      <table><thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u6A21\u5F0F</th><th>\u6765\u6E90</th><th>\u7528\u6237</th><th>\u64CD\u4F5C</th></tr></thead><tbody>${t||u(6)}</tbody></table>
    </section>
  `}function J(){const e=n.dashboard.apiTemplates||[];return`
    <section class="panel">
      <h2>\u521B\u5EFA\u4EFB\u52A1</h2>
      <form id="job-form" class="form-grid">
        <label>\u540D\u79F0<input name="name" placeholder="\u4E3A\u7A7A\u81EA\u52A8\u751F\u6210"></label>
        <label>\u7528\u6237<select name="profileId">${g()}</select></label>
        <label>\u4EFB\u52A1\u6A21\u677F<select name="taskTemplateId">${E()}</select></label>
        <label>\u6A21\u5F0F<select name="taskType"><option value="receive_code">\u6536\u7801</option><option value="send_code">\u53D1\u7801</option></select></label>
        <label>\u624B\u673A\u53F7\u6765\u6E90<select name="phoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>\u624B\u673A\u53F7 API<select name="phoneApiTemplateId">${m(e,0,"phone_source")}</select></label>
        <label>\u9A8C\u8BC1\u7801 API<select name="codeApiTemplateId">${m(e,0,"code_source")}</select></label>
        <label class="wide">TXT \u6587\u4EF6\u8DEF\u5F84<input name="inputPath" placeholder="C:\\path\\phones.txt"></label>
        <div class="form-actions"><button type="submit">\u521B\u5EFA\u5E76\u8FD0\u884C</button></div>
      </form>
    </section>
  `}function k(e){return`
    <section class="panel table-panel">
      <h2>\u4EFB\u52A1\u5217\u8868</h2>
      <table>
        <thead><tr><th>ID</th><th>\u540D\u79F0</th><th>\u6A21\u5F0F</th><th>\u72B6\u6001</th><th>\u603B\u6570</th><th>\u5F85\u5904\u7406</th><th>\u5904\u7406\u4E2D</th><th>\u6210\u529F</th><th>\u5931\u8D25</th><th>\u66F4\u65B0\u65F6\u95F4</th><th>\u64CD\u4F5C</th></tr></thead>
        <tbody>${n.dashboard.jobs.map(a=>{const o=a.job;return`
      <tr>
        <td>${o.ID}</td>
        <td>${s(o.Name)}</td>
        <td>${P(o.TaskType)}</td>
        <td>${N(o.Status)}</td>
        <td>${a.total}</td>
        <td>${a.pending}</td>
        <td>${a.active}</td>
        <td>${a.succeeded}</td>
        <td>${a.failed}</td>
        <td>${$(o.UpdatedAt)}</td>
        <td class="actions">
          <button class="secondary small" data-action="run-job" data-id="${o.ID}">\u8FD0\u884C</button>
          <button class="secondary small" data-action="pause-job" data-id="${o.ID}">\u6682\u505C</button>
          <button class="secondary small" data-action="resume-job" data-id="${o.ID}">\u7EE7\u7EED</button>
          <button class="secondary small danger" data-action="stop-job" data-id="${o.ID}">\u505C\u6B62</button>
          <button class="secondary small" data-action="show-items" data-id="${o.ID}">\u660E\u7EC6</button>
          <button class="secondary small" data-action="export-failed" data-id="${o.ID}">\u5BFC\u51FA\u5931\u8D25</button>
        </td>
      </tr>
    `}).join("")||u(11)}</tbody>
      </table>
    </section>
    ${e?U():""}
  `}function U(){const e=n.selectedItems.map(t=>`
    <tr>
      <td>${t.ID}</td>
      <td>${s(t.Phone)}</td>
      <td>${N(t.Status)}</td>
      <td>${t.RemoteTaskID||"-"}</td>
      <td>${s(t.RemoteStatus||"-")}</td>
      <td>${s(t.VerifyCode||"-")}</td>
      <td class="error-text">${s(t.LastError||"")}</td>
      <td>${$(t.UpdatedAt)}</td>
    </tr>
  `).join("");return`
    <section class="panel table-panel">
      <h2>\u4EFB\u52A1\u660E\u7EC6 ${n.selectedJobId?`#${n.selectedJobId}`:""}</h2>
      <table><thead><tr><th>ID</th><th>\u624B\u673A\u53F7</th><th>\u72B6\u6001</th><th>\u670D\u52A1\u7AEF\u4EFB\u52A1</th><th>\u8FDC\u7AEF\u72B6\u6001</th><th>\u9A8C\u8BC1\u7801</th><th>\u9519\u8BEF</th><th>\u66F4\u65B0\u65F6\u95F4</th></tr></thead><tbody>${e||u(8)}</tbody></table>
    </section>
  `}function E(e=0){var a;const t=['<option value="0">\u4E0D\u4F7F\u7528</option>'];for(const o of((a=n.dashboard)==null?void 0:a.taskTemplates)||[])t.push(`<option value="${o.ID}" ${Number(e)===Number(o.ID)?"selected":""}>${s(o.Name||`#${o.ID}`)}</option>`);return t.join("")}function _(e){var t;return((t=n.dashboard.profiles.find(a=>Number(a.ID)===Number(e)))==null?void 0:t.Name)||"-"}function u(e){return`<tr><td colspan="${e}" class="empty">\u6682\u65E0\u6570\u636E</td></tr>`}async function C(e){await i().SaveSettings({BaseURL:e.BaseURL.value.trim(),ReserveDevices:Number(e.ReserveDevices.value||0),Interval:T(e.IntervalMS.value),Timeout:T(e.TimeoutMS.value),LogDir:e.LogDir.value.trim()})}async function q(e){const t=e.TokenRef.value.trim();await i().SaveProfile({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),TokenRef:t,TokenMask:t?"":void 0,BaseURLOverride:e.BaseURLOverride.value.trim(),CreateDelay:T(e.CreateDelayMS.value),Remark:e.Remark.value.trim(),Enabled:!0})}async function B(e){await i().SaveAPITemplate({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),APIType:e.APIType.value,Method:e.Method.value,URL:e.URL.value.trim(),Query:y(e.Query.value,"Query JSON"),ExtractRule:y(e.ExtractRule.value,"\u63D0\u53D6\u89C4\u5219 JSON"),ResponseType:e.ResponseType.value,Enabled:!0})}async function F(e){await i().SaveTaskTemplate({ID:Number(e.ID.value||0),Name:e.Name.value.trim(),ProfileID:Number(e.ProfileID.value||0),TaskType:e.TaskType.value,PhoneSourceType:e.PhoneSourceType.value,PhoneAPITemplateID:Number(e.PhoneAPITemplateID.value||0),DefaultTXTDir:e.DefaultTXTDir.value.trim(),CodeSourceType:e.TaskType.value==="receive_code"?"api":"none",CodeAPITemplateID:Number(e.CodeAPITemplateID.value||0),FailedOutputDir:e.FailedOutputDir.value.trim(),Remark:e.Remark.value.trim(),Enabled:!0})}async function X(e){await i().StartJob({name:e.name.value.trim(),profileId:Number(e.profileId.value||0),taskTemplateId:Number(e.taskTemplateId.value||0),taskType:e.taskType.value,phoneSourceType:e.phoneSourceType.value,inputPath:e.inputPath.value.trim(),phoneApiTemplateId:Number(e.phoneApiTemplateId.value||0),codeApiTemplateId:Number(e.codeApiTemplateId.value||0)})}function Q(e){const t=n.dashboard.profiles.find(o=>Number(o.ID)===Number(e)),a=document.querySelector("#profile-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.TokenRef.value=t.TokenRef||"",a.BaseURLOverride.value=t.BaseURLOverride||"",a.CreateDelayMS.value=p(t.CreateDelay),a.Remark.value=t.Remark||"")}function G(e){const t=n.dashboard.apiTemplates.find(o=>Number(o.ID)===Number(e)),a=document.querySelector("#api-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.APIType.value=t.APIType||"phone_source",a.Method.value=t.Method||"GET",a.ResponseType.value=t.ResponseType||"auto",a.URL.value=t.URL||"",a.Query.value=JSON.stringify(t.Query||{},null,2),a.ExtractRule.value=JSON.stringify(t.ExtractRule||{},null,2))}function H(e){const t=n.dashboard.taskTemplates.find(o=>Number(o.ID)===Number(e)),a=document.querySelector("#task-template-form");!t||!a||(a.ID.value=t.ID,a.Name.value=t.Name||"",a.ProfileID.value=t.ProfileID||0,a.TaskType.value=t.TaskType||"receive_code",a.PhoneSourceType.value=t.PhoneSourceType||"txt",a.PhoneAPITemplateID.value=t.PhoneAPITemplateID||0,a.CodeAPITemplateID.value=t.CodeAPITemplateID||0,a.DefaultTXTDir.value=t.DefaultTXTDir||"",a.FailedOutputDir.value=t.FailedOutputDir||"",a.Remark.value=t.Remark||"")}function K(e){const t=n.dashboard.taskTemplates.find(o=>Number(o.ID)===Number(e)),a=document.querySelector("#job-form");!t||!a||(t.ProfileID&&(a.profileId.value=t.ProfileID),a.taskType.value=t.TaskType||"receive_code",a.phoneSourceType.value=t.PhoneSourceType||"txt",a.phoneApiTemplateId.value=t.PhoneAPITemplateID||0,a.codeApiTemplateId.value=t.CodeAPITemplateID||0)}document.addEventListener("click",async e=>{var r,d,I;const t=e.target.closest("button");if(!t)return;const a=t.dataset.action,o=t.dataset.page;if(o){n.page=o,f();return}if(!a)return;const l=Number(t.dataset.id||0);try{if(a==="refresh"&&await h(),a==="edit-profile"&&Q(l),a==="clear-profile"&&((r=document.querySelector("#profile-form"))==null||r.reset()),a==="edit-api"&&G(l),a==="clear-api"&&((d=document.querySelector("#api-form"))==null||d.reset()),a==="edit-task-template"&&H(l),a==="clear-task-template"&&((I=document.querySelector("#task-template-form"))==null||I.reset()),a==="run-job"&&await i().RunJob(l),a==="pause-job"&&await i().PauseJob(l),a==="resume-job"&&await i().ResumeJob(l),a==="stop-job"&&await i().StopJob(l),a==="show-items"&&(n.selectedJobId=l,n.selectedItems=await i().ListJobItems(l),n.page="jobs"),a==="export-failed"){const c=prompt("\u5931\u8D25\u6587\u4EF6\u8F93\u51FA\u8DEF\u5F84",`failed-${l}.txt`);c&&await i().ExportFailed(l,c)}["run-job","pause-job","resume-job","stop-job","export-failed"].includes(a)?(await h({keepMessage:!0}),b("\u64CD\u4F5C\u5DF2\u63D0\u4EA4")):f()}catch(c){b(String(c),!0)}});document.addEventListener("change",e=>{var t;((t=e.target)==null?void 0:t.name)==="taskTemplateId"&&e.target.closest("#job-form")&&K(e.target.value)});document.addEventListener("submit",async e=>{e.preventDefault();try{e.target.id==="settings-form"&&await C(e.target),e.target.id==="profile-form"&&await q(e.target),e.target.id==="api-form"&&await B(e.target),e.target.id==="task-template-form"&&await F(e.target),e.target.id==="job-form"&&await X(e.target),await h({keepMessage:!0}),b("\u4FDD\u5B58\u6210\u529F")}catch(t){b(String(t),!0)}});h();
