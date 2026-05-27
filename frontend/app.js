const API_BASE = "http://localhost:8080";

const state = {
  status: null,
  candidates: []
};

document.addEventListener("DOMContentLoaded", () => {
  const page = document.body.dataset.page;
  loadStatus().then(() => {
    if (page === "admin") initAdmin();
    if (page === "voter") initVoter();
    if (page === "results") initResults();
    if (page === "blockchain") initBlockchain();
  });
});

async function api(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok || data.success === false) {
    throw new Error(data.message || `HTTP ${response.status}`);
  }
  return data;
}

async function loadStatus() {
  try {
    state.status = await api("/api/status");
    renderStatus();
  } catch (error) {
    state.status = null;
    setText("votingStatus", "backend недоступен");
    setText("contractAddress", "неизвестно");
  }
}

async function loadCandidates() {
  state.candidates = await api("/api/candidates");
  return state.candidates;
}

function initAdmin() {
  byId("deployBtn")?.addEventListener("click", () => actionButton("deployBtn", deployContract));
  byId("startBtn")?.addEventListener("click", () => actionButton("startBtn", startVoting));
  byId("stopBtn")?.addEventListener("click", () => actionButton("stopBtn", stopVoting));
  byId("candidateForm")?.addEventListener("submit", submitCandidate);
  refreshCandidates("candidateList").catch(showAdminError);
}

function initVoter() {
  byId("voteForm")?.addEventListener("submit", submitVote);
  Promise.all([
    refreshCandidates("candidateList"),
    fillCandidateSelect(),
    renderAccounts()
  ]).catch(error => setMessage("voteMessage", error.message, false));
}

function initResults() {
  byId("refreshResultsBtn")?.addEventListener("click", () => actionButton("refreshResultsBtn", renderResults));
  renderResults().catch(error => setMessage("resultsMessage", error.message, false));
}

function initBlockchain() {
  byId("refreshBlockchainBtn")?.addEventListener("click", () => actionButton("refreshBlockchainBtn", renderBlockchain));
  byId("blockLimit")?.addEventListener("change", () => renderBlockchain().catch(error => setMessage("blockchainMessage", error.message, false)));
  renderBlockchain().catch(error => setMessage("blockchainMessage", error.message, false));
}

async function deployContract() {
  const result = await api("/api/deploy", { method: "POST" });
  setMessage("adminMessage", result.message || "Contract deployed", true);
  await loadStatus();
}

async function startVoting() {
  const result = await api("/api/start", { method: "POST" });
  setMessage("adminMessage", result.message, true);
  await loadStatus();
}

async function stopVoting() {
  const result = await api("/api/stop", { method: "POST" });
  setMessage("adminMessage", result.message, true);
  await loadStatus();
}

async function submitCandidate(event) {
  event.preventDefault();
  const input = byId("candidateName");
  try {
    const result = await api("/api/candidates", {
      method: "POST",
      body: JSON.stringify({ name: input.value.trim() })
    });
    input.value = "";
    setMessage("adminMessage", result.message, true);
    await refreshCandidates("candidateList");
    await loadStatus();
  } catch (error) {
    showAdminError(error);
  }
}

async function submitVote(event) {
  event.preventDefault();
  const voter = byId("voterSelect").value;
  const candidateId = Number(byId("candidateSelect").value);

  try {
    const result = await api("/api/vote", {
      method: "POST",
      body: JSON.stringify({ voter, candidateId })
    });
    setMessage("voteMessage", result.message, true);
    await refreshCandidates("candidateList");
  } catch (error) {
    setMessage("voteMessage", error.message, false);
  }
}

async function refreshCandidates(targetId) {
  const candidates = await loadCandidates();
  const target = byId(targetId);
  if (!target) return;

  if (candidates.length === 0) {
    target.innerHTML = `<div class="list-row"><span>Кандидаты не добавлены</span></div>`;
    return;
  }

  target.innerHTML = candidates.map(candidate => `
    <div class="list-row">
      <span>${escapeHTML(candidate.id)}. ${escapeHTML(candidate.name)}</span>
      <strong>${escapeHTML(candidate.voteCount)} голосов</strong>
    </div>
  `).join("");
}

async function fillCandidateSelect() {
  const candidates = state.candidates.length ? state.candidates : await loadCandidates();
  const select = byId("candidateSelect");
  if (!select) return;

  select.innerHTML = candidates.map(candidate => (
    `<option value="${candidate.id}">${escapeHTML(candidate.name)}</option>`
  )).join("");
}

async function renderResults() {
  const results = await api("/api/results");
  const target = byId("resultsList");

  if (results.length === 0) {
    target.innerHTML = `<div class="result-row"><span>Результаты пока пустые</span></div>`;
    setMessage("resultsMessage", "", true);
    return;
  }

  target.innerHTML = results.map(candidate => `
    <div class="result-row">
      <span>${escapeHTML(candidate.id)}. ${escapeHTML(candidate.name)}</span>
      <strong>${escapeHTML(candidate.voteCount)}</strong>
    </div>
  `).join("");
  setMessage("resultsMessage", "Результаты обновлены", true);
}

async function renderBlockchain() {
  const limit = byId("blockLimit")?.value || "20";
  const data = await api(`/api/blockchain?limit=${encodeURIComponent(limit)}`);
  const target = byId("blockList");

  setText("latestBlock", String(data.latestBlock));
  setText("explorerContract", data.contractAddress || "неизвестен после рестарта backend");

  const interestingBlocks = data.blocks.filter(block => block.transactionCount > 0);
  if (interestingBlocks.length === 0) {
    target.innerHTML = `<div class="list-row"><span>Транзакции не найдены. Пустые блоки майнятся каждую секунду, но здесь показываются только блоки с транзакциями.</span></div>`;
    setMessage("blockchainMessage", "Данные обновлены", true);
    return;
  }

  target.innerHTML = interestingBlocks.map(block => `
    <article class="block-card">
      <header class="block-head">
        <div>
          <strong>Блок #${escapeHTML(block.number)}</strong>
          <code>${escapeHTML(block.hash)}</code>
        </div>
        <span>${escapeHTML(block.transactionCount)} tx</span>
      </header>
      <div class="block-meta">
        <span>miner <code>${escapeHTML(block.miner)}</code></span>
        <span>time ${escapeHTML(new Date(block.time * 1000).toLocaleString())}</span>
      </div>
      <div class="tx-list">
        ${block.transactions.map(renderTx).join("")}
      </div>
    </article>
  `).join("");
  setMessage("blockchainMessage", "Данные обновлены", true);
}

function renderTx(tx) {
  const statusClass = tx.status === "success" ? "ok-pill" : tx.status === "reverted" ? "bad-pill" : "";
  const args = tx.arguments?.length ? `
    <div class="arg-list">
      ${tx.arguments.map(arg => `<span>${escapeHTML(arg.name)}=${escapeHTML(arg.value)}</span>`).join("")}
    </div>
  ` : "";
  const events = tx.events?.length ? `
    <div class="event-list">
      ${tx.events.map(renderEvent).join("")}
    </div>
  ` : "";

  return `
    <article class="tx-card">
      <div class="tx-title">
        <strong>${escapeHTML(tx.function || tx.type)}</strong>
        <span class="status-pill ${statusClass}">${escapeHTML(tx.status)}</span>
      </div>
      <div class="tx-grid">
        <span>hash</span><code>${escapeHTML(tx.hash)}</code>
        <span>from</span><code>${escapeHTML(roleAddress(tx.fromRole, tx.from))}</code>
        <span>to</span><code>${escapeHTML(tx.contractCreated ? `created ${tx.contractCreated}` : roleAddress(tx.toRole, tx.to || "-"))}</code>
        <span>gas used</span><code>${escapeHTML(tx.gasUsed)}</code>
      </div>
      ${args}
      ${events}
      ${tx.error ? `<p class="message error">${escapeHTML(tx.error)}</p>` : ""}
    </article>
  `;
}

function renderEvent(event) {
  const args = event.arguments?.length
    ? event.arguments.map(arg => `${arg.name}=${arg.value}`).join(", ")
    : "";
  return `<div class="event-row"><strong>${escapeHTML(event.name)}</strong><span>${escapeHTML(args)}</span></div>`;
}

function renderStatus() {
  const status = state.status;
  if (!status) return;

  setText("contractAddress", status.contractAddress || "не развернут");
  setText("votingStatus", status.deployed ? (status.isActive ? "активно" : "остановлено") : "контракт не развернут");
  setText("candidatesCount", String(status.candidatesCount || 0));

  const badge = byId("votingBadge");
  if (badge) {
    badge.textContent = status.deployed ? (status.isActive ? "активно" : "остановлено") : "нет контракта";
    badge.classList.toggle("active", Boolean(status.isActive));
    badge.classList.toggle("inactive", !status.isActive);
  }
}

function renderAccounts() {
  const accounts = state.status?.accounts || {};
  const target = byId("accountList");
  if (!target) return;

  target.innerHTML = Object.entries(accounts).map(([name, address]) => `
    <div class="list-row">
      <span>${escapeHTML(name)}</span>
      <code>${escapeHTML(address)}</code>
    </div>
  `).join("");
}

async function actionButton(buttonId, task) {
  const button = byId(buttonId);
  if (button) button.disabled = true;
  try {
    await task();
  } catch (error) {
    if (buttonId.includes("Results")) {
      setMessage("resultsMessage", error.message, false);
    } else {
      if (buttonId.includes("Blockchain")) {
        setMessage("blockchainMessage", error.message, false);
      } else {
        showAdminError(error);
      }
    }
  } finally {
    if (button) button.disabled = false;
  }
}

function showAdminError(error) {
  setMessage("adminMessage", error.message, false);
}

function setMessage(id, text, ok) {
  const el = byId(id);
  if (!el) return;
  el.textContent = text || "";
  el.classList.toggle("ok", Boolean(ok && text));
  el.classList.toggle("error", Boolean(!ok && text));
}

function setText(id, text) {
  const el = byId(id);
  if (el) el.textContent = text;
}

function byId(id) {
  return document.getElementById(id);
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function roleAddress(role, address) {
  if (!role) return address;
  return `${role} (${address})`;
}
