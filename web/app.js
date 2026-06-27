const API = "/api/v1";

let questions = [];
let questionsCacheKey = "";
let creatorId = null;
let lastScriptId = null;
let voiceInputPath = "paste_scripts";
let lastPersona = null;
let lastPrompt = { system: "", user: "", combined: "" };
let igSession = null;
let igProfile = null;
let igReels = [];

const scoreLabels = {
  formality: "Formality", humor: "Humor", energy: "Energy", brevity: "Brevity",
  storytelling: "Storytelling", directness: "Directness", emotional_warmth: "Warmth", hinglish_mix: "Hinglish",
};

async function api(path, options = {}) {
  const res = await fetch(`${API}${path}`, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText || "Request failed");
  return data;
}

function showStep(n) {
  [1, 2, 3, 4].forEach((i) => {
    document.getElementById(`step-${i}`).classList.toggle("hidden", i !== n);
    const pill = document.getElementById(`pill-${i}`);
    pill.classList.toggle("active", i === n);
    pill.classList.toggle("done", i < n);
  });
}

function showError(elId, msg) {
  const el = document.getElementById(elId);
  el.textContent = msg;
  el.classList.toggle("hidden", !msg);
}

// Match services/voice_text.go tokenize + countWords so UI matches server validation.
function countWords(text) {
  text = text.trim().toLowerCase();
  if (!text) return 0;
  let n = 0;
  let current = "";
  for (const char of text) {
    const isLetter = /\p{L}/u.test(char);
    const isNumber = /\p{N}/u.test(char);
    if (isLetter || isNumber || char === "'") {
      current += char;
    } else if (current) {
      n++;
      current = "";
    }
  }
  if (current) n++;
  return n;
}

const guidedWriteLabels = {
  guided_hook: "Video opener",
  guided_hot_take: "Hot take",
  guided_mini_story: "Personal story",
};

function syncVoiceInputPath() {
  const active = document.querySelector(".path-card.active");
  if (active?.dataset.path) voiceInputPath = active.dataset.path;
}

function validateProfileBeforeSubmit() {
  syncVoiceInputPath();
  if (voiceInputPath === "import_instagram") {
    const selected = igReels.filter((r) => r.selected && (r.text || r.caption || r.transcript));
    if (selected.length < 2) {
      const manual = parseManualInstagramCaptions();
      if (manual.length < 2) {
        return "Select or prepare at least 2 reels with caption/transcript text, or paste 2+ captions manually.";
      }
    }
    return null;
  }
  if (voiceInputPath !== "guided_write") return null;

  const issues = [];
  for (const q of questions.filter((item) => item.type === "guided_write")) {
    const el = document.querySelector(`[data-guided-id="${q.id}"]`);
    const text = el?.value.trim() || "";
    const min = q.min_words || 25;
    const n = countWords(text);
    if (n < min) {
      const label = guidedWriteLabels[q.id] || q.text;
      issues.push(`"${label}" needs at least ${min} words (you have ${n})`);
    }
  }

  const answers = collectStyleAnswers();
  if (!answers.preferred_words?.trim()) {
    issues.push("Add words or phrases you use often (preferred words)");
  }
  if (!answers.avoid_words?.trim()) {
    issues.push("Add words or phrases you never use (avoid words)");
  }
  if (!issues.length) return null;
  return issues.join(". ") + ". Each guided exercise is checked separately.";
}

function setupPathPicker() {
  document.querySelectorAll(".path-card").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".path-card").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      voiceInputPath = btn.dataset.path;
      questionsCacheKey = "";
    });
  });
}

async function loadQuestions() {
  syncVoiceInputPath();
  const genre = document.getElementById("genre").value.trim();
  const cacheKey = `${voiceInputPath}:${genre}`;
  if (cacheKey === questionsCacheKey && questions.length > 0) {
    updateCapturePanels();
    return;
  }

  const previousGuided = collectGuidedWrites();
  const previousStyle = collectStyleAnswers();

  const data = await api(`/onboarding/questions?voice_input_path=${voiceInputPath}&genre=${encodeURIComponent(genre)}`);
  questions = data.questions || [];
  questionsCacheKey = cacheKey;
  renderQuestions(previousStyle);
  renderGuidedExercises(previousGuided);
  updateCapturePanels();
}

function updateCapturePanels() {
  document.getElementById("paste-panel").classList.toggle("hidden", voiceInputPath !== "paste_scripts");
  document.getElementById("guided-panel").classList.toggle("hidden", voiceInputPath !== "guided_write");
  document.getElementById("instagram-panel").classList.toggle("hidden", voiceInputPath !== "import_instagram");
  document.getElementById("skip-panel").classList.toggle("hidden", voiceInputPath !== "skip_calibrate");
  if (voiceInputPath === "import_instagram") refreshInstagramStatus();
}

function renderQuestions(savedAnswers = {}) {
  const container = document.getElementById("questions-container");
  container.innerHTML = "";
  questions.filter((q) => q.type !== "guided_write").forEach((q) => {
    const block = document.createElement("div");
    block.className = "question-block";
    const title = document.createElement("label");
    title.textContent = q.text + (q.required ? " *" : "");
    block.appendChild(title);

    if (q.type === "scale_1_5") {
      const row = document.createElement("div");
      row.className = "scale-row";
      for (let i = 1; i <= 5; i++) {
        const lbl = document.createElement("label");
        const input = document.createElement("input");
        input.type = "radio"; input.name = q.id; input.value = String(i);
        if (i === 3) input.checked = true;
        lbl.append(input, document.createTextNode(" " + i));
        row.appendChild(lbl);
      }
      block.appendChild(row);
    } else if (q.type === "single_choice" || q.type === "comparative_choice") {
      const list = document.createElement("div");
      list.className = "options-list";
      q.options.forEach((opt, idx) => {
        const lbl = document.createElement("label");
        const input = document.createElement("input");
        input.type = "radio"; input.name = q.id; input.value = opt.id;
        if (idx === 0) input.checked = true;
        lbl.append(input, document.createTextNode(" " + opt.label));
        list.appendChild(lbl);
      });
      block.appendChild(list);
    } else if (q.type === "free_text") {
      const input = document.createElement(q.id === "anti_voice" || q.id === "inspiration_creators" ? "textarea" : "input");
      input.dataset.questionId = q.id;
      if (savedAnswers[q.id] !== undefined) {
        input.value = savedAnswers[q.id];
      } else if (q.id === "preferred_words") {
        input.value = "yaar, matlab";
      } else if (q.id === "avoid_words") {
        input.value = "delve, folks, leverage";
      }
      if (q.id === "anti_voice") input.placeholder = "motivational speaker, news anchor…";
      block.appendChild(input);
    }
    container.appendChild(block);
  });
}

function renderGuidedExercises(savedWrites = {}) {
  const container = document.getElementById("guided-container");
  container.innerHTML = "";
  questions.filter((q) => q.type === "guided_write").forEach((q) => {
    const block = document.createElement("div");
    block.className = "question-block";
    const title = document.createElement("label");
    title.textContent = q.text;
    block.appendChild(title);
    if (q.starter_line) {
      const hint = document.createElement("p");
      hint.className = "hint";
      hint.textContent = "Starter: " + q.starter_line;
      block.appendChild(hint);
    }
    const ta = document.createElement("textarea");
    ta.dataset.guidedId = q.id;
    ta.dataset.minWords = q.min_words || 25;
    ta.rows = 5;
    ta.value = savedWrites[q.id] || q.starter_line || "";
    block.appendChild(ta);
    const counter = document.createElement("span");
    counter.className = "word-count under-min";
    const min = q.min_words || 25;
    const updateCounter = () => {
      const n = countWords(ta.value);
      counter.textContent = `${n} / ${min} words`;
      counter.classList.toggle("under-min", n < min);
      counter.classList.toggle("ready", n >= min);
    };
    ta.addEventListener("input", updateCounter);
    updateCounter();
    block.appendChild(counter);
    container.appendChild(block);
  });
}

function collectStyleAnswers() {
  const answers = {};
  questions.forEach((q) => {
    if (q.type === "guided_write") return;
    if (q.type === "free_text") {
      const el = document.querySelector(`[data-question-id="${q.id}"]`);
      if (el) answers[q.id] = el.value.trim();
      return;
    }
    const selected = document.querySelector(`input[name="${q.id}"]:checked`);
    if (selected) answers[q.id] = selected.value;
  });
  return answers;
}

function collectWritingSamples() {
  const raw = document.getElementById("writing_samples").value.trim();
  if (!raw) return [];
  return raw.split(/\n---\n/).map((s) => s.trim()).filter(Boolean);
}

function collectGuidedWrites() {
  const out = {};
  document.querySelectorAll("[data-guided-id]").forEach((el) => {
    out[el.dataset.guidedId] = el.value.trim();
  });
  return out;
}

function parseManualInstagramCaptions() {
  const raw = document.getElementById("instagram_manual_captions")?.value.trim() || "";
  if (!raw) return [];
  return raw.split(/\n---\n/).map((s) => s.trim()).filter(Boolean);
}

function collectInstagramReels() {
  const prepared = igReels.filter((r) => r.selected && (r.text || r.caption || r.transcript));
  if (prepared.length >= 2) {
    return prepared.map((r) => ({
      ...r,
      text: r.text || r.caption || r.transcript,
      selected: true,
    }));
  }
  return parseManualInstagramCaptions().map((caption, i) => ({
    id: `manual-${i}`,
    caption,
    text: caption,
    text_source: "caption",
    selected: true,
  }));
}

function buildProfilePayload() {
  syncVoiceInputPath();
  const payload = {
    name: document.getElementById("name").value.trim(),
    genre: document.getElementById("genre").value.trim(),
    language: document.getElementById("language").value.trim(),
    platform: document.getElementById("platform").value.trim(),
    region: document.getElementById("region").value.trim(),
    content_type: document.getElementById("content_type").value.trim(),
    bio: document.getElementById("bio").value.trim(),
    style_answers: collectStyleAnswers(),
    voice_input_path: voiceInputPath,
    writing_samples: voiceInputPath === "paste_scripts" ? collectWritingSamples() : [],
    guided_writes: voiceInputPath === "guided_write" ? collectGuidedWrites() : {},
  };
  if (voiceInputPath === "import_instagram") {
    payload.instagram = igProfile || undefined;
    payload.instagram_reels = collectInstagramReels();
    if (igProfile?.biography && !payload.bio) payload.bio = igProfile.biography;
    if (igProfile?.name && !payload.name) payload.name = igProfile.name;
    payload.platform = payload.platform || "Instagram";
    payload.content_type = payload.content_type || "Reels";
    if (igProfile?.username) payload.channel = "@" + igProfile.username;
  }
  return payload;
}

function renderPersona(persona) {
  lastPersona = persona;
  document.getElementById("creator-id").textContent = persona.creator_id;
  document.getElementById("voice-mode").textContent = "Mode: " + (persona.voice_mode || "declared");
  document.getElementById("voice-confidence").textContent = "Confidence: " + (persona.voice_confidence ?? 0) + "/100";
  document.getElementById("voice-path").textContent = "Path: " + (persona.voice_input_path || voiceInputPath);

  const fp = persona.voice_fingerprint || {};
  const stats = [];
  if (fp.total_words) stats.push(`Words analyzed: ${fp.total_words}`);
  if (fp.avg_sentence_length) stats.push(`Avg sentence: ${fp.avg_sentence_length.toFixed(1)}`);
  if (fp.hook_pattern) stats.push(`Hook: ${fp.hook_pattern}`);
  if (fp.hinglish_ratio) stats.push(`Hinglish ratio: ${(fp.hinglish_ratio * 100).toFixed(0)}%`);
  document.getElementById("fingerprint-stats").textContent = stats.length ? stats.join(" · ") : "No fingerprint yet — calibrate after your first edited script.";

  const scores = persona.current_scores || {};
  const scoreContainer = document.getElementById("scores-container");
  scoreContainer.innerHTML = "";
  Object.entries(scoreLabels).forEach(([key, label]) => {
    const val = scores[key] ?? 50;
    const div = document.createElement("div");
    div.className = "score-item";
    div.innerHTML = `<span>${label}</span><strong>${val}</strong>`;
    scoreContainer.appendChild(div);
  });

  const lex = persona.lexical_profile || {};
  const lexicalContainer = document.getElementById("lexical-container");
  lexicalContainer.innerHTML = [
    lex.preferred_words?.length ? "Preferred: " + lex.preferred_words.join(", ") : "",
    lex.avoid_words?.length ? "Avoid: " + lex.avoid_words.join(", ") : "",
    lex.filler_words?.length ? "Fillers: " + lex.filler_words.join(", ") : "",
    lex.sentence_starters?.length ? "Starters: " + lex.sentence_starters.join(", ") : "",
  ].filter(Boolean).join("<br>") || "No lexical data yet.";
}

async function submitProfile() {
  showError("step-2-error", "");
  const validationError = validateProfileBeforeSubmit();
  if (validationError) {
    showError("step-2-error", validationError);
    return;
  }
  const payload = buildProfilePayload();
  const data = await api("/submit-profile", { method: "POST", body: JSON.stringify(payload) });
  creatorId = data.creator_id;
  if (data.warning) showError("step-2-error", data.warning);
  renderPersona(data.persona);
  showStep(3);
}

async function generatePrompt() {
  showError("step-4-error", "");
  const topic = document.getElementById("topic").value.trim();
  const variant = document.getElementById("variant").value;
  const endpoint = variant === "base" ? "/generate-prompt" : "/generate-prompt-ab";
  const body = variant === "base"
    ? { creator_id: creatorId, topic }
    : { creator_id: creatorId, topic };

  const data = await api(endpoint, { method: "POST", body: JSON.stringify(body) });

  if (variant === "base") {
    lastPrompt = { system: data.system_prompt, user: data.user_prompt, combined: data.prompt };
  } else {
    lastPrompt = { system: "", user: "", combined: data.variants?.[variant] || data.variants?.A || "" };
  }

  document.getElementById("prompt-combined").textContent = lastPrompt.combined || "No prompt returned";
  document.getElementById("prompt-system").textContent = lastPrompt.system || "(see combined)";
  document.getElementById("prompt-user").textContent = lastPrompt.user || "(see combined)";

  const scriptData = await api("/generate-script", {
    method: "POST",
    body: JSON.stringify({ creator_id: creatorId, topic, variant: variant === "base" ? "base" : variant }),
  }).catch(() => null);
  if (scriptData?.script_id) lastScriptId = scriptData.script_id;
}

async function submitFeedback() {
  if (!lastScriptId) {
    document.getElementById("feedback-result").textContent = "Generate a prompt first (needs a saved script for feedback).";
    document.getElementById("feedback-result").classList.remove("hidden");
    return;
  }
  const toggles = [...document.querySelectorAll("#toggle-row input:checked")].map((el) => el.value);
  const data = await api(`/scripts/${lastScriptId}/feedback`, {
    method: "POST",
    body: JSON.stringify({
      rating: document.getElementById("feedback_rating").value,
      notes: document.getElementById("feedback_notes").value.trim(),
      toggles,
      edited_script: document.getElementById("edited_script").value.trim(),
      generated_script: document.getElementById("generated_script").value.trim(),
    }),
  });
  const msg = `Updated — mode: ${data.voice_mode}, confidence: ${data.voice_confidence}/100` +
    (data.shift_score ? `, shift score: ${data.shift_score}` : "");
  document.getElementById("feedback-result").textContent = msg;
  document.getElementById("feedback-result").classList.remove("hidden");
  if (creatorId) {
    const persona = await api(`/persona/${creatorId}`);
    renderPersona(persona);
  }
}

function copyText(text) {
  navigator.clipboard.writeText(text).then(() => {
    document.getElementById("copy-msg").classList.remove("hidden");
    setTimeout(() => document.getElementById("copy-msg").classList.add("hidden"), 2000);
  });
}

document.getElementById("btn-to-capture").addEventListener("click", async () => {
  showError("step-1-error", "");
  try {
    await loadQuestions();
    showStep(2);
  } catch (e) {
    showError("step-1-error", e.message);
  }
});

document.getElementById("btn-back-1").addEventListener("click", () => showStep(1));
document.getElementById("btn-back-2").addEventListener("click", () => showStep(2));
document.getElementById("btn-back-3").addEventListener("click", () => showStep(3));

document.getElementById("btn-submit").addEventListener("click", async () => {
  try { await submitProfile(); } catch (e) { showError("step-2-error", e.message); }
});

document.getElementById("btn-to-prompt").addEventListener("click", () => showStep(4));

document.getElementById("btn-generate").addEventListener("click", async () => {
  try { await generatePrompt(); } catch (e) { showError("step-4-error", e.message); }
});

document.getElementById("btn-copy-combined").addEventListener("click", () => copyText(lastPrompt.combined));
document.getElementById("btn-feedback").addEventListener("click", async () => {
  try { await submitFeedback(); } catch (e) {
    document.getElementById("feedback-result").textContent = e.message;
    document.getElementById("feedback-result").classList.remove("hidden");
  }
});

async function refreshInstagramStatus() {
  const el = document.getElementById("instagram-status");
  try {
    const status = await api("/instagram/status");
    if (!status.configured) {
      el.innerHTML = 'Instagram OAuth not configured on server. Paste captions below, or see planning/meta-instagram-setup.md in the repo.';
      return;
    }
    el.textContent = igSession
      ? "Connected — select reels and click Prepare."
      : "Click Connect Instagram (Business/Creator account linked to a Facebook Page).";
  } catch {
    el.textContent = "Could not reach Instagram API.";
  }
}

async function connectInstagram() {
  const data = await api("/instagram/auth-url");
  window.location.href = data.auth_url;
}

async function loadInstagramReels() {
  if (!igSession) return;
  const bundle = await api(`/instagram/reels?session=${encodeURIComponent(igSession)}`);
  igProfile = bundle.profile || null;
  igReels = (bundle.reels || []).map((r) => ({ ...r, selected: true }));
  if (igProfile) {
    const p = document.getElementById("instagram-profile");
    p.classList.remove("hidden");
    p.textContent = `@${igProfile.username || "—"} · ${igProfile.followers_count || 0} followers · ${igProfile.media_count || 0} posts`;
    if (igProfile.biography) document.getElementById("bio").value = igProfile.biography;
    if (igProfile.name) document.getElementById("name").value = igProfile.name;
    document.getElementById("platform").value = "Instagram";
    document.getElementById("content_type").value = "Reels";
  }
  renderInstagramReelsTable();
  document.getElementById("btn-instagram-refresh").classList.remove("hidden");
  document.getElementById("btn-instagram-prepare").classList.remove("hidden");
}

function renderInstagramReelsTable() {
  const container = document.getElementById("instagram-reels-table");
  if (!igReels.length) {
    container.innerHTML = "<p class=\"hint\">No reels found on this account.</p>";
    return;
  }
  const rows = igReels.map((r) => {
    const preview = (r.text || r.caption || "").slice(0, 80);
    const src = r.text_source ? ` · ${r.text_source}` : "";
    return `<tr>
      <td><input type="checkbox" data-reel-id="${r.id}" ${r.selected ? "checked" : ""} /></td>
      <td>${r.like_count ?? "—"}</td>
      <td>${countWords(r.text || r.caption || "")}w${src}</td>
      <td class="reel-preview">${preview || "—"}</td>
    </tr>`;
  }).join("");
  container.innerHTML = `<table class="reel-table"><thead><tr><th></th><th>Likes</th><th>Words</th><th>Preview</th></tr></thead><tbody>${rows}</tbody></table>`;
  container.querySelectorAll("input[data-reel-id]").forEach((cb) => {
    cb.addEventListener("change", () => {
      const reel = igReels.find((r) => r.id === cb.dataset.reelId);
      if (reel) reel.selected = cb.checked;
    });
  });
}

async function prepareInstagramReels() {
  const ids = igReels.filter((r) => r.selected).map((r) => r.id);
  if (!ids.length) throw new Error("Select at least one reel");
  const transcribe = document.getElementById("instagram-transcribe").checked;
  const data = await api("/instagram/prepare", {
    method: "POST",
    body: JSON.stringify({ session_id: igSession, reel_ids: ids, transcribe }),
  });
  igReels = data.reels || [];
  renderInstagramReelsTable();
}

function handleInstagramReturn() {
  const params = new URLSearchParams(window.location.search);
  const err = params.get("instagram_error");
  if (err) {
    voiceInputPath = "import_instagram";
    document.querySelectorAll(".path-card").forEach((b) => {
      b.classList.toggle("active", b.dataset.path === "import_instagram");
    });
    showStep(2);
    showError("step-2-error", decodeURIComponent(err.replace(/\+/g, " ")));
    window.history.replaceState({}, "", "/app/");
    return;
  }
  const session = params.get("ig_session");
  if (session) {
    igSession = session;
    voiceInputPath = "import_instagram";
    document.querySelectorAll(".path-card").forEach((b) => {
      b.classList.toggle("active", b.dataset.path === "import_instagram");
    });
    window.history.replaceState({}, "", "/app/");
    loadQuestions().then(() => {
      showStep(2);
      loadInstagramReels().catch((e) => showError("step-2-error", e.message));
    });
  }
}

document.getElementById("btn-instagram-connect").addEventListener("click", async () => {
  try { await connectInstagram(); } catch (e) { showError("step-2-error", e.message); }
});
document.getElementById("btn-instagram-refresh").addEventListener("click", async () => {
  try { await loadInstagramReels(); } catch (e) { showError("step-2-error", e.message); }
});
document.getElementById("btn-instagram-prepare").addEventListener("click", async () => {
  try { await prepareInstagramReels(); } catch (e) { showError("step-2-error", e.message); }
});

setupPathPicker();
handleInstagramReturn();
showStep(1);
