import './style.css';

const app = document.querySelector('#app');

const state = {
  page: 'dashboard',
  dashboard: null,
  selectedJobId: 0,
  selectedItems: [],
  message: '',
  error: '',
};

const pages = [
  ['dashboard', '运行面板'],
  ['settings', '全局配置'],
  ['profiles', '用户'],
  ['apis', 'API 模板'],
  ['tasks', '任务模板'],
  ['jobs', '任务历史'],
];

const api = () => window.go?.main?.App;

function getDurationMs(value) {
  const n = Number(value || 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.round(n / 1000000);
}

function toDurationNs(ms) {
  const n = Number(ms || 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.round(n * 1000000);
}

function fmtDuration(value) {
  const ms = getDurationMs(value);
  if (!ms) return '0s';
  if (ms % 1000 === 0) return `${ms / 1000}s`;
  return `${ms}ms`;
}

function fmtTime(value) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}

function h(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function parseJSONMap(raw, field) {
  const text = String(raw || '').trim();
  if (!text) return {};
  try {
    const parsed = JSON.parse(text);
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error(`${field} 必须是 JSON 对象`);
    }
    return parsed;
  } catch (error) {
    throw new Error(`${field} 解析失败: ${error.message}`);
  }
}

function templateOptions(items, selected = 0, filterType = '') {
  const rows = ['<option value="0">不使用</option>'];
  for (const item of items || []) {
    if (filterType && item.APIType !== filterType) continue;
    rows.push(`<option value="${item.ID}" ${Number(selected) === Number(item.ID) ? 'selected' : ''}>${h(item.Name || `#${item.ID}`)}</option>`);
  }
  return rows.join('');
}

function profileOptions(selected = 0) {
  const rows = ['<option value="0">请选择</option>'];
  for (const item of state.dashboard?.profiles || []) {
    rows.push(`<option value="${item.ID}" ${Number(selected) === Number(item.ID) ? 'selected' : ''}>${h(item.Name || `#${item.ID}`)}</option>`);
  }
  return rows.join('');
}

function statusText(status) {
  const map = {
    running: '运行中',
    paused: '已暂停',
    stopped: '已停止',
    finished: '已完成',
    pending: '待开始',
    succeeded: '成功',
    failed: '失败',
    created: '已创建',
    waiting_code: '等验证码',
    code_submitted: '已提交',
  };
  return map[status] || status || '-';
}

function modeText(value) {
  return value === 'send_code' ? '发码' : '收码';
}

function sourceText(value) {
  return value === 'api' ? 'API' : value === 'txt' ? 'TXT' : '-';
}

function showMessage(message, isError = false) {
  state.message = isError ? '' : message;
  state.error = isError ? message : '';
  render();
}

async function refresh(options = {}) {
  try {
    state.dashboard = await api().Dashboard();
    state.error = '';
    if (options.keepMessage !== true) state.message = '';
  } catch (error) {
    state.error = String(error);
  }
  render();
}

function render() {
  const dash = state.dashboard;
  if (!api()) {
    app.innerHTML = '<main class="offline">请在 Wails 程序内打开。</main>';
    return;
  }
  if (!dash) {
    app.innerHTML = '<main class="offline">正在加载...</main>';
    return;
  }
  app.innerHTML = `
    <main class="shell">
      <aside class="nav">
        <div class="brand">Phone Task Client</div>
        ${pages.map(([key, label]) => `<button class="nav-item ${state.page === key ? 'active' : ''}" data-page="${key}">${label}</button>`).join('')}
      </aside>
      <section class="content">
        <header class="topbar">
          <div>
            <h1>${h(pageTitle())}</h1>
            <p>${h(dash.status.description)}</p>
            <p class="version-line">版本 ${h(dash.status.version || '-')} / commit ${h(dash.status.gitCommit || '-')}</p>
          </div>
          <button class="secondary" data-action="refresh">刷新</button>
        </header>
        ${state.message ? `<div class="notice">${h(state.message)}</div>` : ''}
        ${state.error ? `<div class="notice error">${h(state.error)}</div>` : ''}
        ${renderPage()}
      </section>
    </main>
  `;
}

function pageTitle() {
  return pages.find(([key]) => key === state.page)?.[1] || '运行面板';
}

function renderPage() {
  switch (state.page) {
    case 'settings':
      return renderSettings();
    case 'profiles':
      return renderProfiles();
    case 'apis':
      return renderAPIs();
    case 'tasks':
      return renderTaskTemplates();
    case 'jobs':
      return renderJobs(true);
    default:
      return renderDashboard();
  }
}

function renderDashboard() {
  const dash = state.dashboard;
  const totals = dash.jobs.reduce((acc, row) => {
    acc.total += row.total;
    acc.pending += row.pending;
    acc.active += row.active;
    acc.succeeded += row.succeeded;
    acc.failed += row.failed;
    return acc;
  }, { total: 0, pending: 0, active: 0, succeeded: 0, failed: 0 });
  return `
    <section class="metrics">
      <div class="metric"><span>任务数</span><strong>${dash.jobs.length}</strong></div>
      <div class="metric"><span>处理中</span><strong>${totals.pending + totals.active}</strong></div>
      <div class="metric"><span>成功</span><strong>${totals.succeeded}</strong></div>
      <div class="metric"><span>失败</span><strong>${totals.failed}</strong></div>
    </section>
    ${renderCreateJob()}
    ${renderJobs(false)}
  `;
}

function renderSettings() {
  const s = state.dashboard.settings || {};
  return `
    <section class="panel">
      <form id="settings-form" class="form-grid">
        <label>服务器地址<input name="BaseURL" value="${h(s.BaseURL)}" placeholder="https://server.example"></label>
        <label>额外保留设备<input name="ReserveDevices" type="number" min="0" value="${h(s.ReserveDevices || 0)}"></label>
        <label>轮询间隔 ms<input name="IntervalMS" type="number" min="200" value="${h(getDurationMs(s.Interval))}"></label>
        <label>请求超时 ms<input name="TimeoutMS" type="number" min="1000" value="${h(getDurationMs(s.Timeout))}"></label>
        <label class="wide">日志目录<input name="LogDir" value="${h(s.LogDir)}"></label>
        <div class="form-actions"><button type="submit">保存配置</button></div>
      </form>
    </section>
  `;
}

function renderProfiles() {
  const rows = state.dashboard.profiles.map((p) => `
    <tr>
      <td>${p.ID}</td>
      <td>${h(p.Name)}</td>
      <td>${h(p.TokenMask || '****')}</td>
      <td>${fmtDuration(p.CreateDelay)}</td>
      <td>${h(p.BaseURLOverride || '-')}</td>
      <td><button class="secondary small" data-action="edit-profile" data-id="${p.ID}">编辑</button></td>
    </tr>
  `).join('');
  return `
    <section class="panel">
      <form id="profile-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>名称<input name="Name" placeholder="sales-1"></label>
        <label>Token<input name="TokenRef" placeholder="openapi token"></label>
        <label>create_delay ms<input name="CreateDelayMS" type="number" min="0" value="0"></label>
        <label>覆盖服务器<input name="BaseURLOverride" placeholder="为空时使用全局配置"></label>
        <label class="wide">备注<input name="Remark"></label>
        <div class="form-actions">
          <button type="submit">保存用户</button>
          <button type="button" class="secondary" data-action="clear-profile">清空</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>名称</th><th>Token</th><th>延迟</th><th>服务器</th><th>操作</th></tr></thead><tbody>${rows || emptyRow(6)}</tbody></table>
    </section>
  `;
}

function renderAPIs() {
  const rows = state.dashboard.apiTemplates.map((t) => `
    <tr>
      <td>${t.ID}</td>
      <td>${h(t.Name)}</td>
      <td>${h(t.APIType)}</td>
      <td>${h(t.Method || 'GET')}</td>
      <td class="mono">${h(t.URL)}</td>
      <td><button class="secondary small" data-action="edit-api" data-id="${t.ID}">编辑</button></td>
    </tr>
  `).join('');
  return `
    <section class="panel">
      <form id="api-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>名称<input name="Name" placeholder="code-api"></label>
        <label>类型<select name="APIType"><option value="phone_source">手机号 API</option><option value="code_source">验证码 API</option></select></label>
        <label>方法<select name="Method"><option value="GET">GET</option></select></label>
        <label>响应<select name="ResponseType"><option value="auto">自动</option><option value="text">文本</option><option value="json">JSON</option></select></label>
        <label class="wide">URL<input name="URL" placeholder="https://example.com/code?phone={phone}"></label>
        <label class="wide">Query JSON<textarea name="Query" rows="3" placeholder='{"phone":"{phone}"}'></textarea></label>
        <label class="wide">提取规则 JSON<textarea name="ExtractRule" rows="3" placeholder='{"code":"data.code"}'></textarea></label>
        <div class="form-actions">
          <button type="submit">保存 API 模板</button>
          <button type="button" class="secondary" data-action="clear-api">清空</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>名称</th><th>类型</th><th>方法</th><th>URL</th><th>操作</th></tr></thead><tbody>${rows || emptyRow(6)}</tbody></table>
    </section>
  `;
}

function renderTaskTemplates() {
  const apiItems = state.dashboard.apiTemplates || [];
  const rows = state.dashboard.taskTemplates.map((t) => `
    <tr>
      <td>${t.ID}</td>
      <td>${h(t.Name)}</td>
      <td>${modeText(t.TaskType)}</td>
      <td>${sourceText(t.PhoneSourceType)}</td>
      <td>${h(profileName(t.ProfileID))}</td>
      <td><button class="secondary small" data-action="edit-task-template" data-id="${t.ID}">编辑</button></td>
    </tr>
  `).join('');
  return `
    <section class="panel">
      <form id="task-template-form" class="form-grid">
        <input type="hidden" name="ID" value="0">
        <label>名称<input name="Name" placeholder="receive-txt"></label>
        <label>默认用户<select name="ProfileID">${profileOptions()}</select></label>
        <label>模式<select name="TaskType"><option value="receive_code">收码</option><option value="send_code">发码</option></select></label>
        <label>手机号来源<select name="PhoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>手机号 API<select name="PhoneAPITemplateID">${templateOptions(apiItems, 0, 'phone_source')}</select></label>
        <label>验证码 API<select name="CodeAPITemplateID">${templateOptions(apiItems, 0, 'code_source')}</select></label>
        <label class="wide">默认 TXT 目录<input name="DefaultTXTDir"></label>
        <label class="wide">失败导出目录<input name="FailedOutputDir"></label>
        <label class="wide">备注<input name="Remark"></label>
        <div class="form-actions">
          <button type="submit">保存任务模板</button>
          <button type="button" class="secondary" data-action="clear-task-template">清空</button>
        </div>
      </form>
    </section>
    <section class="panel table-panel">
      <table><thead><tr><th>ID</th><th>名称</th><th>模式</th><th>来源</th><th>用户</th><th>操作</th></tr></thead><tbody>${rows || emptyRow(6)}</tbody></table>
    </section>
  `;
}

function renderCreateJob() {
  const apiItems = state.dashboard.apiTemplates || [];
  return `
    <section class="panel">
      <h2>创建任务</h2>
      <form id="job-form" class="form-grid">
        <label>名称<input name="name" placeholder="为空自动生成"></label>
        <label>用户<select name="profileId">${profileOptions()}</select></label>
        <label>任务模板<select name="taskTemplateId">${taskTemplateOptions()}</select></label>
        <label>模式<select name="taskType"><option value="receive_code">收码</option><option value="send_code">发码</option></select></label>
        <label>手机号来源<select name="phoneSourceType"><option value="txt">TXT</option><option value="api">API</option></select></label>
        <label>手机号 API<select name="phoneApiTemplateId">${templateOptions(apiItems, 0, 'phone_source')}</select></label>
        <label>验证码 API<select name="codeApiTemplateId">${templateOptions(apiItems, 0, 'code_source')}</select></label>
        <label class="wide">TXT 文件路径<input name="inputPath" placeholder="C:\\path\\phones.txt"></label>
        <div class="form-actions"><button type="submit">创建并运行</button></div>
      </form>
    </section>
  `;
}

function renderJobs(includeItems) {
  const rows = state.dashboard.jobs.map((row) => {
    const job = row.job;
    return `
      <tr>
        <td>${job.ID}</td>
        <td>${h(job.Name)}</td>
        <td>${modeText(job.TaskType)}</td>
        <td>${statusText(job.Status)}</td>
        <td>${row.total}</td>
        <td>${row.pending}</td>
        <td>${row.active}</td>
        <td>${row.succeeded}</td>
        <td>${row.failed}</td>
        <td>${fmtTime(job.UpdatedAt)}</td>
        <td class="actions">
          <button class="secondary small" data-action="run-job" data-id="${job.ID}">运行</button>
          <button class="secondary small" data-action="pause-job" data-id="${job.ID}">暂停</button>
          <button class="secondary small" data-action="resume-job" data-id="${job.ID}">继续</button>
          <button class="secondary small danger" data-action="stop-job" data-id="${job.ID}">停止</button>
          <button class="secondary small" data-action="show-items" data-id="${job.ID}">明细</button>
          <button class="secondary small" data-action="export-failed" data-id="${job.ID}">导出失败</button>
        </td>
      </tr>
    `;
  }).join('');
  return `
    <section class="panel table-panel">
      <h2>任务列表</h2>
      <table>
        <thead><tr><th>ID</th><th>名称</th><th>模式</th><th>状态</th><th>总数</th><th>待处理</th><th>处理中</th><th>成功</th><th>失败</th><th>更新时间</th><th>操作</th></tr></thead>
        <tbody>${rows || emptyRow(11)}</tbody>
      </table>
    </section>
    ${includeItems ? renderItems() : ''}
  `;
}

function renderItems() {
  const rows = state.selectedItems.map((item) => `
    <tr>
      <td>${item.ID}</td>
      <td>${h(item.Phone)}</td>
      <td>${statusText(item.Status)}</td>
      <td>${item.RemoteTaskID || '-'}</td>
      <td>${h(item.RemoteStatus || '-')}</td>
      <td>${h(item.VerifyCode || '-')}</td>
      <td class="error-text">${h(item.LastError || '')}</td>
      <td>${fmtTime(item.UpdatedAt)}</td>
    </tr>
  `).join('');
  return `
    <section class="panel table-panel">
      <h2>任务明细 ${state.selectedJobId ? `#${state.selectedJobId}` : ''}</h2>
      <table><thead><tr><th>ID</th><th>手机号</th><th>状态</th><th>服务端任务</th><th>远端状态</th><th>验证码</th><th>错误</th><th>更新时间</th></tr></thead><tbody>${rows || emptyRow(8)}</tbody></table>
    </section>
  `;
}

function taskTemplateOptions(selected = 0) {
  const rows = ['<option value="0">不使用</option>'];
  for (const item of state.dashboard?.taskTemplates || []) {
    rows.push(`<option value="${item.ID}" ${Number(selected) === Number(item.ID) ? 'selected' : ''}>${h(item.Name || `#${item.ID}`)}</option>`);
  }
  return rows.join('');
}

function profileName(id) {
  return state.dashboard.profiles.find((p) => Number(p.ID) === Number(id))?.Name || '-';
}

function emptyRow(cols) {
  return `<tr><td colspan="${cols}" class="empty">暂无数据</td></tr>`;
}

async function saveSettings(form) {
  await api().SaveSettings({
    BaseURL: form.BaseURL.value.trim(),
    ReserveDevices: Number(form.ReserveDevices.value || 0),
    Interval: toDurationNs(form.IntervalMS.value),
    Timeout: toDurationNs(form.TimeoutMS.value),
    LogDir: form.LogDir.value.trim(),
  });
}

async function saveProfile(form) {
  const token = form.TokenRef.value.trim();
  await api().SaveProfile({
    ID: Number(form.ID.value || 0),
    Name: form.Name.value.trim(),
    TokenRef: token,
    TokenMask: token ? '' : undefined,
    BaseURLOverride: form.BaseURLOverride.value.trim(),
    CreateDelay: toDurationNs(form.CreateDelayMS.value),
    Remark: form.Remark.value.trim(),
    Enabled: true,
  });
}

async function saveAPI(form) {
  await api().SaveAPITemplate({
    ID: Number(form.ID.value || 0),
    Name: form.Name.value.trim(),
    APIType: form.APIType.value,
    Method: form.Method.value,
    URL: form.URL.value.trim(),
    Query: parseJSONMap(form.Query.value, 'Query JSON'),
    ExtractRule: parseJSONMap(form.ExtractRule.value, '提取规则 JSON'),
    ResponseType: form.ResponseType.value,
    Enabled: true,
  });
}

async function saveTaskTemplate(form) {
  await api().SaveTaskTemplate({
    ID: Number(form.ID.value || 0),
    Name: form.Name.value.trim(),
    ProfileID: Number(form.ProfileID.value || 0),
    TaskType: form.TaskType.value,
    PhoneSourceType: form.PhoneSourceType.value,
    PhoneAPITemplateID: Number(form.PhoneAPITemplateID.value || 0),
    DefaultTXTDir: form.DefaultTXTDir.value.trim(),
    CodeSourceType: form.TaskType.value === 'receive_code' ? 'api' : 'none',
    CodeAPITemplateID: Number(form.CodeAPITemplateID.value || 0),
    FailedOutputDir: form.FailedOutputDir.value.trim(),
    Remark: form.Remark.value.trim(),
    Enabled: true,
  });
}

async function startJob(form) {
  await api().StartJob({
    name: form.name.value.trim(),
    profileId: Number(form.profileId.value || 0),
    taskTemplateId: Number(form.taskTemplateId.value || 0),
    taskType: form.taskType.value,
    phoneSourceType: form.phoneSourceType.value,
    inputPath: form.inputPath.value.trim(),
    phoneApiTemplateId: Number(form.phoneApiTemplateId.value || 0),
    codeApiTemplateId: Number(form.codeApiTemplateId.value || 0),
  });
}

function fillProfile(id) {
  const item = state.dashboard.profiles.find((p) => Number(p.ID) === Number(id));
  const form = document.querySelector('#profile-form');
  if (!item || !form) return;
  form.ID.value = item.ID;
  form.Name.value = item.Name || '';
  form.TokenRef.value = item.TokenRef || '';
  form.BaseURLOverride.value = item.BaseURLOverride || '';
  form.CreateDelayMS.value = getDurationMs(item.CreateDelay);
  form.Remark.value = item.Remark || '';
}

function fillAPI(id) {
  const item = state.dashboard.apiTemplates.find((t) => Number(t.ID) === Number(id));
  const form = document.querySelector('#api-form');
  if (!item || !form) return;
  form.ID.value = item.ID;
  form.Name.value = item.Name || '';
  form.APIType.value = item.APIType || 'phone_source';
  form.Method.value = item.Method || 'GET';
  form.ResponseType.value = item.ResponseType || 'auto';
  form.URL.value = item.URL || '';
  form.Query.value = JSON.stringify(item.Query || {}, null, 2);
  form.ExtractRule.value = JSON.stringify(item.ExtractRule || {}, null, 2);
}

function fillTaskTemplate(id) {
  const item = state.dashboard.taskTemplates.find((t) => Number(t.ID) === Number(id));
  const form = document.querySelector('#task-template-form');
  if (!item || !form) return;
  form.ID.value = item.ID;
  form.Name.value = item.Name || '';
  form.ProfileID.value = item.ProfileID || 0;
  form.TaskType.value = item.TaskType || 'receive_code';
  form.PhoneSourceType.value = item.PhoneSourceType || 'txt';
  form.PhoneAPITemplateID.value = item.PhoneAPITemplateID || 0;
  form.CodeAPITemplateID.value = item.CodeAPITemplateID || 0;
  form.DefaultTXTDir.value = item.DefaultTXTDir || '';
  form.FailedOutputDir.value = item.FailedOutputDir || '';
  form.Remark.value = item.Remark || '';
}

function fillJobFromTaskTemplate(id) {
  const item = state.dashboard.taskTemplates.find((t) => Number(t.ID) === Number(id));
  const form = document.querySelector('#job-form');
  if (!item || !form) return;
  if (item.ProfileID) form.profileId.value = item.ProfileID;
  form.taskType.value = item.TaskType || 'receive_code';
  form.phoneSourceType.value = item.PhoneSourceType || 'txt';
  form.phoneApiTemplateId.value = item.PhoneAPITemplateID || 0;
  form.codeApiTemplateId.value = item.CodeAPITemplateID || 0;
}

document.addEventListener('click', async (event) => {
  const target = event.target.closest('button');
  if (!target) return;
  const action = target.dataset.action;
  const page = target.dataset.page;
  if (page) {
    state.page = page;
    render();
    return;
  }
  if (!action) return;
  const id = Number(target.dataset.id || 0);
  try {
    if (action === 'refresh') await refresh();
    if (action === 'edit-profile') fillProfile(id);
    if (action === 'clear-profile') document.querySelector('#profile-form')?.reset();
    if (action === 'edit-api') fillAPI(id);
    if (action === 'clear-api') document.querySelector('#api-form')?.reset();
    if (action === 'edit-task-template') fillTaskTemplate(id);
    if (action === 'clear-task-template') document.querySelector('#task-template-form')?.reset();
    if (action === 'run-job') await api().RunJob(id);
    if (action === 'pause-job') await api().PauseJob(id);
    if (action === 'resume-job') await api().ResumeJob(id);
    if (action === 'stop-job') await api().StopJob(id);
    if (action === 'show-items') {
      state.selectedJobId = id;
      state.selectedItems = await api().ListJobItems(id);
      state.page = 'jobs';
    }
    if (action === 'export-failed') {
      const path = prompt('失败文件输出路径', `failed-${id}.txt`);
      if (path) await api().ExportFailed(id, path);
    }
    if (['run-job', 'pause-job', 'resume-job', 'stop-job', 'export-failed'].includes(action)) {
      await refresh({ keepMessage: true });
      showMessage('操作已提交');
    } else {
      render();
    }
  } catch (error) {
    showMessage(String(error), true);
  }
});

document.addEventListener('change', (event) => {
  if (event.target?.name === 'taskTemplateId' && event.target.closest('#job-form')) {
    fillJobFromTaskTemplate(event.target.value);
  }
});

document.addEventListener('submit', async (event) => {
  event.preventDefault();
  try {
    if (event.target.id === 'settings-form') await saveSettings(event.target);
    if (event.target.id === 'profile-form') await saveProfile(event.target);
    if (event.target.id === 'api-form') await saveAPI(event.target);
    if (event.target.id === 'task-template-form') await saveTaskTemplate(event.target);
    if (event.target.id === 'job-form') await startJob(event.target);
    await refresh({ keepMessage: true });
    showMessage('保存成功');
  } catch (error) {
    showMessage(String(error), true);
  }
});

refresh();
