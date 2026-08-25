(function () {
  "use strict";

  var state = { cases: [], selected: null, view: null };
  var listEl = document.getElementById("case-list");
  var detailEl = document.getElementById("detail");
  var panelEl = document.getElementById("action-panel");
  var countEl = document.getElementById("case-count");
  var noticeEl = document.getElementById("notice");
  var dialog = document.getElementById("create-dialog");

  function h(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function lines(value) {
    return String(value || "").split(/\n|；|;/).map(function (item) { return item.trim(); }).filter(Boolean);
  }

  function requestID(prefix) {
    if (window.crypto && window.crypto.randomUUID) return prefix + "-" + window.crypto.randomUUID();
    return prefix + "-" + Date.now() + "-" + Math.random().toString(16).slice(2);
  }

  function actor(name, role) {
    return { name: String(name || "").trim(), role: role };
  }

  async function api(path, options) {
    var response = await fetch(path, Object.assign({ headers: { "Content-Type": "application/json" } }, options || {}));
    var wrapped;
    try { wrapped = await response.json(); } catch (_) { throw new Error("服务返回了无法识别的响应"); }
    if (!response.ok || wrapped.error) {
      var error = wrapped.error || { message: "请求失败" };
      var detail = (error.issues || []).map(function (issue) { return issue.field + "：" + issue.message; }).join("；");
      throw new Error(error.message + (detail ? "。" + detail : ""));
    }
    return wrapped.data;
  }

  function notify(message, error) {
    noticeEl.textContent = message;
    noticeEl.className = "notice" + (error ? " error" : "");
    noticeEl.hidden = false;
    window.clearTimeout(notify.timer);
    notify.timer = window.setTimeout(function () { noticeEl.hidden = true; }, 5000);
  }

  async function loadCases() {
    var query = new URLSearchParams();
    var status = document.getElementById("status-filter").value;
    var search = document.getElementById("search").value.trim();
    if (status) query.set("status", status);
    if (search) query.set("q", search);
    try {
      state.cases = await api("/api/cases?" + query.toString());
      renderList();
    } catch (error) { notify(error.message, true); }
  }

  function renderList() {
    countEl.textContent = state.cases.length + " 件事项";
    if (!state.cases.length) {
      listEl.innerHTML = '<div class="panel-empty">暂无符合条件的事项</div>';
      return;
    }
    listEl.innerHTML = state.cases.map(function (item) {
      var active = item.id === state.selected ? " active" : "";
      return '<button class="case-item' + active + '" data-id="' + h(item.id) + '" type="button">' +
        "<strong>" + h(item.title) + "</strong><small>" + h(item.shelf_mark) + " · " + h(item.responsible_conservator) + "</small>" +
        '<span class="item-meta"><span class="badge" data-status="' + h(item.status) + '">' + h(item.status_label) +
        "</span><span>r" + h(item.revision) + "</span></span></button>";
    }).join("");
  }

  async function selectCase(id) {
    state.selected = id;
    renderList();
    try {
      state.view = await api("/api/cases/" + encodeURIComponent(id));
      renderDetail();
      renderAction();
    } catch (error) { notify(error.message, true); }
  }

  function valueList(values) {
    return (values || []).map(h).join("；") || "—";
  }

  function evidenceList(refs) {
    if (!refs || !refs.length) return "<p>无</p>";
    return '<ul class="evidence-list">' + refs.map(function (ref) {
      return "<li><strong>" + h(ref.filename) + "</strong><br><small>" + h(ref.media_type) + " · " + h(ref.id) +
        (ref.sha256 ? " · SHA-256 " + h(ref.sha256) : "") + "</small></li>";
    }).join("") + "</ul>";
  }

  function facts(items) {
    return '<dl class="facts">' + items.map(function (item) {
      return "<div><dt>" + h(item[0]) + "</dt><dd>" + h(item[1] == null || item[1] === "" ? "—" : item[1]) + "</dd></div>";
    }).join("") + "</dl>";
  }

  function renderDetail() {
    var view = state.view;
    var item = view.case;
    var html = '<header class="case-header"><div class="case-title-row"><div><h1>' + h(item.title) + "</h1><p>" +
      h(item.shelf_mark) + " · " + h(item.version_identifier) + '</p></div><span class="badge" data-status="' +
      h(item.status) + '">' + h(view.status_label) + '</span></div><div class="meta-strip"><span>责任修复师 <b>' +
      h(item.responsible_conservator) + "</b></span><span>修订 <b>r" + h(item.revision) + "</b></span><span>更新 <b>" +
      h(new Date(item.updated_at).toLocaleString("zh-CN")) + "</b></span></div></header>";

    html += '<section class="section"><h2>建档与损伤证据</h2>' + facts([
      ["载体材料", item.support_material], ["载体特征", item.carrier_characteristics],
      ["损伤部位", valueList(item.damage_locations)], ["版本标识", item.version_identifier]
    ]) + '<h2>图像证据</h2>' + evidenceList(item.initial_evidence) + "</section>";

    if (item.assessment) {
      var a = item.assessment;
      html += '<section class="section"><h2>专业损伤评估</h2>' + facts([
        ["损伤等级", a.severity], ["评估人", a.assessor], ["损伤表现", valueList(a.symptoms)],
        ["劣化原因", valueList(a.probable_causes)], ["保护目标", valueList(a.treatment_goals)],
        ["不可干预边界", valueList(a.non_intervention_limits)]
      ]) + "</section>";
	  if (a.partition_assessments && a.partition_assessments.length) {
		html += '<section class="section"><h2>分区损伤与保护边界</h2><ul class="review-list">' + a.partition_assessments.map(function (part) {
		  return '<li><strong>' + h(part.location) + ' · ' + h(part.severity) + '</strong><br>表现：' + h(valueList(part.symptoms)) + '<br>范围：' + h(part.impact_scope) + '<br>目标：' + h(valueList(part.treatment_goals)) + '<br>边界：' + h(valueList(part.boundaries)) + '</li>';
		}).join("") + '</ul></section>';
	  }
    }

    if (item.proposal) {
      var p = item.proposal;
      html += '<section class="section"><h2>修复方案 ' + h(p.proposal_version) + "</h2>" + facts([
        ["材料", p.materials.map(function (m) { return m.name + " / " + m.specification + " / " + m.purpose; }).join("\n")],
        ["环境要求", valueList(p.environment_requirements)], ["可逆性说明", p.reversibility_notes],
        ["风险控制", valueList(p.risk_controls)], ["验收条件", valueList(p.acceptance_criteria)]
      ]) + '<h2>分步工序</h2><ol class="step-list">' + p.procedure_steps.map(function (step) {
        return "<li><strong>" + h(step.instruction) + "</strong><br><small>检查点：" + h(step.checkpoint) + "</small></li>";
      }).join("") + "</ol></section>";
    }

	if (item.proposal_history && item.proposal_history.length) {
	  html += '<section class="section"><h2>方案版本历史</h2><ul class="review-list">' + item.proposal_history.map(function (record) {
		return '<li><strong>' + h(record.proposal.proposal_version) + '</strong><br>审议意见 ' + h((record.review_comments || []).length) + ' 条；差异 ' + h((record.diff || []).map(function (d) { return d.field + ' ' + d.change; }).join('；') || '无') + '</li>';
	  }).join("") + '</ul></section>';
	}

    if (item.reviews && item.reviews.length) {
      html += '<section class="section"><h2>同行审议 · ' + h(view.review_conclusion) + '</h2><ul class="review-list">' +
        item.reviews.map(function (review) {
          return "<li><strong>" + h(review.reviewer) + "</strong> · " + h(review.decision) +
            (review.reservations ? "<br>保留意见：" + h(review.reservations) : "") +
            (review.return_reason ? "<br>退回理由：" + h(review.return_reason) : "") + "</li>";
        }).join("") + "</ul></section>";
    }

    if (item.trial) {
      var t = item.trial;
      html += '<section class="section"><h2>小样核验 · ' + h(t.verdict) + "</h2>" + facts([
        ["批次", t.batch_code], ["方法", t.method], ["观察值", valueList(t.observations)], ["偏差", valueList(t.deviations)]
	  ]) + ((t.deviation_records || []).length ? '<h2>偏差处置</h2><ul class="review-list">' + t.deviation_records.map(function (d) { return '<li><strong>' + h(d.id) + ' · ' + h(d.impact) + ' · ' + h(d.closed ? '已关闭' : '未关闭') + '</strong><br>' + h(d.action) + '</li>'; }).join('') + '</ul>' : '') + evidenceList(t.evidence_refs) + "</section>";
    }
	if (item.trial_batches && item.trial_batches.length) {
	  html += '<section class="section"><h2>小样批次历史</h2><ul class="review-list">' + item.trial_batches.map(function (batch) {
		return '<li><strong>' + h(batch.batch_code) + ' · ' + h(batch.verdict) + '</strong><br>' + h(valueList(batch.observations)) + (batch.failed_criteria && batch.failed_criteria.length ? '<br>未通过条件：' + h(valueList(batch.failed_criteria)) : '') + (batch.previous_batch_id ? '<br>复测来源：' + h(batch.previous_batch_id) : '') + '</li>';
	  }).join("") + '</ul></section>';
	}

    html += '<section class="section"><h2>状态时间线</h2><ol class="timeline">' + view.timeline.map(function (entry) {
      return "<li><strong>" + h(entry.label) + "</strong><small>" + h(entry.actor) + " · " +
        h(new Date(entry.occurred_at).toLocaleString("zh-CN")) + " · " + h(entry.to_status) + "</small></li>";
    }).join("") + "</ol></section>";
    detailEl.innerHTML = html;
  }

  function metaFields(role, label) {
    return '<label>' + h(label) + '<input name="actor_name" required></label><input type="hidden" name="role" value="' + h(role) + '">';
  }

  function renderAction() {
    var view = state.view;
    var item = view.case;
    var head = "<h2>当前阶段操作</h2><p class=\"stage\">" + h(view.status_label) + " · r" + h(item.revision) + "</p>";
    var form = "";
	if (item.status === "draft" && view.allowed_actions.indexOf("revise_draft") !== -1) {
	  form = '<form class="action-form" data-action="draft">' + metaFields("conservator", "责任修复师") +
		'<label>题名<input name="title" value="' + h(item.title) + '" required></label><label>版本标识<input name="version_identifier" value="' + h(item.version_identifier) + '" required></label>' +
		'<label>载体特征<textarea name="carrier_characteristics" required>' + h(item.carrier_characteristics) + '</textarea></label><label>损伤部位<textarea name="damage_locations" required>' + h((item.damage_locations || []).join("\n")) + '</textarea></label>' +
		'<label>证据（标识 | 文件名 | 媒体类型 | SHA-256 | 备注）<textarea name="evidence_rows" required>' + h((item.initial_evidence || []).map(function (e) { return e.id + ' | ' + e.filename + ' | ' + e.media_type + ' | ' + (e.sha256 || '') + ' | ' + (e.note || ''); }).join("\n")) + '</textarea></label>' +
		'<button class="secondary" type="submit">保存草稿修订</button></form><form class="action-form" data-action="submit">' + metaFields("conservator", "执行修复师") + '<button class="primary" type="submit">提交专业评估</button></form>';
    } else if (item.status === "pending_assessment") {
	  var partitionRows = (item.damage_locations || []).map(function (location, index) { return '<fieldset><legend>' + h(location) + '</legend><input type="hidden" name="part_location_' + index + '" value="' + h(location) + '"><label>局部等级<select name="part_severity_' + index + '"><option value="minor">轻度</option><option value="moderate">中度</option><option value="severe">重度</option></select></label><label>损伤表现<input name="part_symptoms_' + index + '" required></label><label>影响范围<input name="part_scope_' + index + '" required></label><label>保护目标<input name="part_goals_' + index + '" required></label><label>不可干预边界<input name="part_boundaries_' + index + '" required></label></fieldset>'; }).join('');
      form = '<form class="action-form" data-action="assessment">' + metaFields("manager", "评估负责人") +
        '<label>损伤等级<select name="severity"><option value="minor">轻度</option><option value="moderate">中度</option><option value="severe">重度</option></select></label>' +
		partitionRows + '<label>总体损伤表现<textarea name="symptoms" required></textarea></label>' +
        '<label>劣化原因<textarea name="causes" required></textarea></label><label>保护目标<textarea name="goals" required></textarea></label>' +
        '<label>不可干预边界<textarea name="limits" required></textarea></label><label>证据文件名<input name="evidence" required></label>' +
        '<button class="primary" type="submit">确认评估</button></form>';
    } else if (item.status === "proposal_drafting") {
	  var lastHistory = item.proposal_history && item.proposal_history[item.proposal_history.length - 1];
	  var responseFields = lastHistory ? (lastHistory.review_comments || []).filter(function (r) { return r.decision !== 'approve'; }).map(function (r) { return '<label>回应 ' + h(r.id) + '<textarea name="response_' + h(r.id) + '" required></textarea></label>'; }).join('') : '';
      form = '<form class="action-form" data-action="proposal">' + metaFields("conservator", "方案编制人") +
        '<label>方案版本<input name="version" required></label><label>材料名称<input name="material" required></label>' +
        '<label>材料规格<input name="specification" required></label><label>材料用途<input name="purpose" required></label>' +
        '<label>分步工序<textarea name="steps" required></textarea></label><label>工序检查点<textarea name="checkpoints" required></textarea></label>' +
        '<label>环境要求<textarea name="environment" required></textarea></label><label>可逆性说明<textarea name="reversibility" required></textarea></label>' +
		'<label>风险缓解措施<textarea name="risks" required></textarea></label><label>小样验收条件<textarea name="criteria" required></textarea></label>' + responseFields +
        '<button class="primary" type="submit">提交同行审议</button></form>';
    } else if (item.status === "peer_review") {
	  var progress = view.review_progress || {};
	  var roster = item.review_roster;
	  var rosterMembers = roster ? (roster.members || roster.experts || []) : [];
	  form = '<p class="stage">已提交 ' + h(progress.submitted || 0) + ' · 待提交 ' + h(progress.pending || 0) + ' · 回避 ' + h(progress.recused || 0) + ' · 距法定人数 ' + h(progress.remaining || 0) + '</p><form class="action-form" data-action="roster">' + metaFields("manager", "名册负责人") + '<label>专家名册（每行：姓名；回避原因可选）<textarea name="members" required>' + h(rosterMembers.map(function (member) { return member.name + (member.recused ? '；' + (member.reason || member.recusal_reason || '') : ''); }).join("\n")) + '</textarea></label><label>法定人数<input name="quorum" type="number" min="2" value="' + h(roster ? roster.quorum : 2) + '" required></label><button class="secondary" type="submit">保存本轮名册</button></form><form class="action-form" data-action="review">' + metaFields("reviewer", "审议专家") +
        '<label>决定<select name="decision"><option value="approve">赞同</option><option value="reservation">保留意见</option><option value="return">退回</option></select></label>' +
        '<label>保留意见<textarea name="reservations"></textarea></label><label>退回理由<textarea name="return_reason"></textarea></label>' +
        '<button class="primary" type="submit">提交独立意见</button></form>';
    } else if (item.status === "pending_trial") {
	  var failed = (item.trial_batches || []).filter(function (batch) { return batch.verdict === 'failed'; });
      form = '<form class="action-form" data-action="trial">' + metaFields("conservator", "核验修复师") +
        '<label>小样批次<input name="batch" required></label><label>试验方法<textarea name="method" required></textarea></label>' +
        '<label>观察值<textarea name="observations" required></textarea></label><label>偏差<textarea name="deviations"></textarea></label>' +
		'<label>偏差处置（标识 | 影响等级 | 措施 | 已关闭）<textarea name="deviation_records"></textarea></label><label>前次失败批次<select name="previous_batch_id"><option value="">无</option>' + failed.map(function (batch) { return '<option value="' + h(batch.id) + '">' + h(batch.batch_code) + '</option>'; }).join('') + '</select></label>' +
        '<label>证据文件名<input name="evidence" required></label><div><strong>验收条件</strong>' +
        item.proposal.acceptance_criteria.map(function (criterion, index) {
          return '<label><input type="checkbox" name="criterion_' + index + '" checked> ' + h(criterion) + "</label>";
        }).join("") + '</div><button class="primary" type="submit">完成小样核验</button></form>';
    } else if (item.status === "pending_approval") {
	  var precheck = view.archive_precheck || { items: [] };
	  var blocked = precheck.items.some(function (check) { return check.blocking && !check.passed; });
	  form = '<form class="action-form" data-action="archive">' + metaFields("manager", "批准负责人") + '<ul class="review-list">' + precheck.items.map(function (check) { var hint = !check.blocking && !check.passed ? '<label><input type="checkbox" name="confirmed_hints" value="' + h(check.code) + '" required> 已确认此提示</label>' : ''; return '<li><strong>' + h(check.passed ? '通过' : (check.blocking ? '阻断' : '提示')) + ' · ' + h(check.stage) + '</strong><br>' + h(check.message) + hint + '</li>'; }).join('') + '</ul><button class="danger" type="submit"' + (blocked ? ' disabled' : '') + '>批准并封存</button></form>';
    } else {
      form = '<div class="archive-block"><strong>封存摘要已锁定</strong><p>该事项仅允许查看与导出，后续变更请求将被拒绝。</p>' +
        '<button class="secondary" id="download-audit" type="button">下载 JSON 审计包</button></div>';
    }
    panelEl.innerHTML = head + form;
  }

  function commonPayload(form, prefix) {
    var data = new FormData(form);
    return {
      request_id: requestID(prefix),
      expected_revision: state.view.case.revision,
      actor: actor(data.get("actor_name"), data.get("role"))
    };
  }

  async function submitAction(form) {
    var action = form.dataset.action;
    var data = new FormData(form);
    var payload = commonPayload(form, action);
    var path = "/api/cases/" + encodeURIComponent(state.selected);
    if (action === "submit") path += "/submit";
	if (action === "draft") {
	  path += "/draft";
	  payload.revision = { title: data.get("title"), version_identifier: data.get("version_identifier"), carrier_characteristics: data.get("carrier_characteristics"), damage_locations: lines(data.get("damage_locations")), initial_evidence: lines(data.get("evidence_rows")).map(function (row) { var parts = row.split('|').map(function (part) { return part.trim(); }); var old = (state.view.case.initial_evidence || []).find(function (e) { return e.id === parts[0]; }) || {}; return { id: parts[0] || '', filename: parts[1] || '', media_type: parts[2] || '', sha256: parts[3] || '', note: parts[4] || '', captured_at: old.captured_at }; }) };
	}
    if (action === "assessment") {
      path += "/assessment";
	  var partitions = (state.view.case.damage_locations || []).map(function (_, index) { return { location: data.get("part_location_" + index), severity: data.get("part_severity_" + index), symptoms: lines(data.get("part_symptoms_" + index)), impact_scope: data.get("part_scope_" + index), treatment_goals: lines(data.get("part_goals_" + index)), boundaries: lines(data.get("part_boundaries_" + index)) }; });
      payload.assessment = {
		severity: data.get("severity"), locations: state.view.case.damage_locations, symptoms: lines(data.get("symptoms")),
        probable_causes: lines(data.get("causes")), treatment_goals: lines(data.get("goals")),
        non_intervention_limits: lines(data.get("limits")), assessor: data.get("actor_name"),
		evidence_refs: [{ id: requestID("evidence"), filename: data.get("evidence"), media_type: "image/jpeg" }], partition_assessments: partitions
      };
    }
    if (action === "proposal") {
      path += "/proposal";
      var steps = lines(data.get("steps"));
      var checkpoints = lines(data.get("checkpoints"));
      payload.proposal = {
        proposal_version: data.get("version"),
        materials: [{ name: data.get("material"), specification: data.get("specification"), purpose: data.get("purpose") }],
        procedure_steps: steps.map(function (step, index) { return { order: index + 1, instruction: step, checkpoint: checkpoints[index] || checkpoints[0] || "" }; }),
        environment_requirements: lines(data.get("environment")), reversibility_notes: data.get("reversibility"),
        risk_controls: lines(data.get("risks")), acceptance_criteria: lines(data.get("criteria"))
      };
	  payload.response_notes = {};
	  var lastHistory = state.view.case.proposal_history && state.view.case.proposal_history[state.view.case.proposal_history.length - 1];
	  (lastHistory ? lastHistory.review_comments || [] : []).forEach(function (review) { if (review.decision !== 'approve') payload.response_notes[review.id] = data.get("response_" + review.id); });
    }
	if (action === "roster") {
	  path += "/reviews/roster";
	  payload.roster = { proposal_version: state.view.case.proposal.proposal_version, quorum: Number(data.get("quorum")), members: String(data.get("members") || '').split(/\n/).map(function (row) { return row.trim(); }).filter(Boolean).map(function (row) { var parts = row.split('；'); return { name: parts[0].trim(), recused: parts.length > 1 && parts[1].trim() !== '', reason: parts.length > 1 ? parts[1].trim() : '' }; }) };
	}
    if (action === "review") {
      path += "/reviews";
      payload.review = {
        id: requestID("review"), proposal_version: state.view.case.proposal.proposal_version,
        reviewer: data.get("actor_name"), decision: data.get("decision"),
        reservations: data.get("reservations"), return_reason: data.get("return_reason")
      };
    }
    if (action === "trial") {
      path += "/trial";
      payload.trial = {
        id: requestID("trial"), batch_code: data.get("batch"), method: data.get("method"),
		observations: lines(data.get("observations")), deviations: lines(data.get("deviations")), previous_batch_id: data.get("previous_batch_id"),
		deviation_records: lines(data.get("deviation_records")).map(function (row) { var parts = row.split('|').map(function (part) { return part.trim(); }); return { id: parts[0] || '', impact: parts[1] || '', action: parts[2] || '', closed: parts[3] === '已关闭' || parts[3] === 'true' }; }),
        evidence_refs: [{ id: requestID("trial-evidence"), filename: data.get("evidence"), media_type: "image/jpeg" }],
        criterion_results: state.view.case.proposal.acceptance_criteria.map(function (criterion, index) {
          return { criterion: criterion, passed: data.get("criterion_" + index) === "on", note: data.get("criterion_" + index) === "on" ? "" : "小样未达到条件" };
        })
      };
    }
	if (action === "archive") { path += "/archive"; payload.confirmed_hints = data.getAll("confirmed_hints"); }
    await api(path, { method: "POST", body: JSON.stringify(payload) });
    notify("操作已完成", false);
    await loadCases();
    await selectCase(state.selected);
  }

  async function createCase(form) {
    var data = new FormData(form);
    var name = data.get("responsible_conservator");
    var payload = {
      request_id: requestID("create"), actor: actor(name, "conservator"),
      shelf_mark: data.get("shelf_mark"), title: data.get("title"), version_identifier: data.get("version_identifier"),
      support_material: data.get("support_material"), carrier_characteristics: data.get("carrier_characteristics"),
      damage_locations: lines(data.get("damage_locations")), responsible_conservator: name,
      initial_evidence: [{ id: requestID("evidence"), filename: data.get("evidence_filename"), media_type: "image/jpeg" }]
    };
    var result = await api("/api/cases", { method: "POST", body: JSON.stringify(payload) });
    dialog.close();
    form.reset();
    await loadCases();
    await selectCase(result.case.id);
    notify("草稿已保存", false);
  }

  async function downloadAudit() {
    try {
      var data = await api("/api/cases/" + encodeURIComponent(state.selected) + "/audit");
      var blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      var url = URL.createObjectURL(blob);
      var link = document.createElement("a");
      link.href = url;
      link.download = "audit-" + state.selected + ".json";
      link.click();
      URL.revokeObjectURL(url);
    } catch (error) { notify(error.message, true); }
  }

  listEl.addEventListener("click", function (event) {
    var button = event.target.closest("[data-id]");
    if (button) selectCase(button.dataset.id);
  });
  panelEl.addEventListener("submit", function (event) {
    event.preventDefault();
    submitAction(event.target).catch(function (error) { notify(error.message, true); });
  });
  panelEl.addEventListener("click", function (event) {
    if (event.target.id === "download-audit") downloadAudit();
  });
  document.getElementById("new-case").addEventListener("click", function () { dialog.showModal(); });
  document.querySelectorAll("[data-close]").forEach(function (button) {
    button.addEventListener("click", function () { dialog.close(); });
  });
  document.getElementById("create-form").addEventListener("submit", function (event) {
    event.preventDefault();
    createCase(event.target).catch(function (error) { notify(error.message, true); });
  });
  document.getElementById("status-filter").addEventListener("change", loadCases);
  document.getElementById("search").addEventListener("input", function () {
    window.clearTimeout(loadCases.timer);
    loadCases.timer = window.setTimeout(loadCases, 250);
  });
  loadCases();
}());
