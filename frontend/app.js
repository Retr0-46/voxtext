const $ = (id) => document.getElementById(id);

const state = {
  token: null,
  user: null,
  file: null,
  objectUrl: null,
  utterances: [],
  controller: null,
};

const elements = {
  authView: $("authView"), studioView: $("studioView"), globalMessage: $("globalMessage"),
  loginTab: $("loginTab"), registerTab: $("registerTab"), loginForm: $("loginForm"),
  registerForm: $("registerForm"), authError: $("authError"), logoutButton: $("logoutButton"),
  userName: $("userName"), userInitial: $("userInitial"), dropZone: $("dropZone"),
  audioFile: $("audioFile"), fileDetails: $("fileDetails"), fileName: $("fileName"),
  fileSize: $("fileSize"), clearFileButton: $("clearFileButton"), audioPreview: $("audioPreview"),
  transcribeButton: $("transcribeButton"), cancelButton: $("cancelButton"),
  processingStatus: $("processingStatus"), resultCount: $("resultCount"),
  emptyResults: $("emptyResults"), utteranceList: $("utteranceList"),
  copyButton: $("copyButton"), downloadButton: $("downloadButton"),
};

function showMessage(message, error = false) {
  elements.globalMessage.textContent = message;
  elements.globalMessage.classList.toggle("error", error);
  elements.globalMessage.hidden = false;
}

function clearMessage() {
  elements.globalMessage.hidden = true;
  elements.globalMessage.textContent = "";
}

function showAuthError(message) {
  elements.authError.textContent = message;
  elements.authError.hidden = false;
}

function selectTab(tab) {
  const login = tab === "login";
  elements.loginForm.hidden = !login;
  elements.registerForm.hidden = login;
  elements.loginTab.classList.toggle("active", login);
  elements.registerTab.classList.toggle("active", !login);
  elements.loginTab.setAttribute("aria-selected", String(login));
  elements.registerTab.setAttribute("aria-selected", String(!login));
  elements.authError.hidden = true;
  clearMessage();
  (login ? $("loginEmail") : $("registerName")).focus();
}

async function request(path, options = {}) {
  let response;
  try {
    response = await fetch(path, options);
  } catch (error) {
    if (error.name === "AbortError") throw error;
    throw new Error("Не удалось связаться с сервером. Проверьте, что backend запущен.");
  }
  const raw = await response.text();
  let data;
  try {
    data = raw ? JSON.parse(raw) : null;
  } catch {
    data = null;
  }
  if (!response.ok) {
    if (response.status === 401) throw Object.assign(new Error(path === "/auth/login" ? "Неверная почта или пароль." : "Сессия истекла. Войдите снова."), { status: 401 });
    if (response.status === 409) throw new Error("Пользователь с такой почтой или именем уже существует.");
    if (response.status === 413) throw new Error("Файл слишком большой для загрузки.");
    if (response.status === 400) throw new Error("Проверьте введённые данные или выбранный файл.");
    throw new Error(data?.error || "Не удалось выполнить запрос. Попробуйте ещё раз.");
  }
  return data;
}

function setBusy(form, busy) {
  const button = form.querySelector('button[type="submit"]');
  button.disabled = busy;
  button.dataset.label ??= button.innerHTML;
  button.innerHTML = busy ? "Подождите…" : button.dataset.label;
}

function showStudio(user) {
  state.user = user;
  elements.userName.textContent = user.username || "Пользователь";
  elements.userInitial.textContent = (user.username || "В").charAt(0).toUpperCase();
  elements.authView.hidden = true;
  elements.studioView.hidden = false;
  elements.logoutButton.hidden = false;
  clearMessage();
}

function logout(message = "") {
  state.controller?.abort();
  state.controller = null;
  state.token = null;
  state.user = null;
  state.utterances = [];
  setProcessing(false);
  clearFile();
  renderResults();
  elements.studioView.hidden = true;
  elements.authView.hidden = false;
  elements.logoutButton.hidden = true;
  elements.loginForm.reset();
  elements.registerForm.reset();
  elements.authError.hidden = true;
  selectTab("login");
  if (message) showMessage(message, true);
}

async function login(email, password) {
  const result = await request("/auth/login", {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  state.token = result.access_token;
  const user = await request("/auth/me", { headers: { Authorization: `Bearer ${state.token}` } });
  showStudio(user);
}

elements.loginTab.addEventListener("click", () => selectTab("login"));
elements.registerTab.addEventListener("click", () => selectTab("register"));
elements.logoutButton.addEventListener("click", () => logout());

elements.loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  elements.authError.hidden = true;
  if (!elements.loginForm.reportValidity()) return;
  setBusy(elements.loginForm, true);
  const data = new FormData(elements.loginForm);
  try {
    await login(String(data.get("email")).trim(), String(data.get("password")));
  } catch (error) {
    state.token = null;
    showAuthError(error.message);
  } finally {
    setBusy(elements.loginForm, false);
  }
});

elements.registerForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  elements.authError.hidden = true;
  if (!elements.registerForm.reportValidity()) return;
  setBusy(elements.registerForm, true);
  const data = new FormData(elements.registerForm);
  const payload = {
    username: String(data.get("username")).trim(),
    usersurname: String(data.get("usersurname")).trim(),
    email: String(data.get("email")).trim(),
    password: String(data.get("password")),
  };
  try {
    await request("/auth/register", {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload),
    });
    await login(payload.email, payload.password);
  } catch (error) {
    showAuthError(error.message);
  } finally {
    setBusy(elements.registerForm, false);
  }
});

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`;
}

function clearFile() {
  if (state.objectUrl) URL.revokeObjectURL(state.objectUrl);
  state.objectUrl = null;
  state.file = null;
  elements.audioFile.value = "";
  elements.fileDetails.hidden = true;
  elements.audioPreview.hidden = true;
  elements.audioPreview.removeAttribute("src");
  elements.transcribeButton.disabled = true;
}

function selectFile(file) {
  if (!file) return;
  if (state.objectUrl) URL.revokeObjectURL(state.objectUrl);
  state.file = file;
  if (file.type.startsWith("audio/")) {
    state.objectUrl = URL.createObjectURL(file);
    elements.audioPreview.src = state.objectUrl;
    elements.audioPreview.hidden = false;
  } else {
    state.objectUrl = null;
    elements.audioPreview.hidden = true;
    elements.audioPreview.removeAttribute("src");
  }
  elements.fileName.textContent = file.name;
  elements.fileSize.textContent = formatBytes(file.size);
  elements.fileDetails.hidden = false;
  elements.transcribeButton.disabled = false;
  clearMessage();
}

elements.dropZone.addEventListener("click", () => { if (!elements.audioFile.disabled) elements.audioFile.click(); });
elements.dropZone.addEventListener("keydown", (event) => {
  if ((event.key === "Enter" || event.key === " ") && !elements.audioFile.disabled) { event.preventDefault(); elements.audioFile.click(); }
});
elements.audioFile.addEventListener("change", () => selectFile(elements.audioFile.files[0]));
elements.clearFileButton.addEventListener("click", clearFile);
for (const eventName of ["dragenter", "dragover"]) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault(); elements.dropZone.classList.add("dragover");
  });
}
for (const eventName of ["dragleave", "drop"]) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault(); elements.dropZone.classList.remove("dragover");
  });
}
elements.dropZone.addEventListener("drop", (event) => {
  if (!elements.audioFile.disabled) selectFile(event.dataTransfer.files[0]);
});

function formatTime(seconds) {
  const value = Number.isFinite(Number(seconds)) ? Math.max(0, Number(seconds)) : 0;
  const minutes = Math.floor(value / 60);
  const rest = Math.floor(value % 60);
  return `${String(minutes).padStart(2, "0")}:${String(rest).padStart(2, "0")}`;
}

function transcriptText() {
  return state.utterances.map((item) => `[${formatTime(item.start)}] Спикер ${item.speaker + 1}: ${item.text}`).join("\n");
}

function renderResults() {
  const items = state.utterances;
  elements.resultCount.textContent = items.length ? `${items.length} реплик` : "Пока пусто";
  elements.emptyResults.hidden = items.length > 0;
  elements.utteranceList.hidden = items.length === 0;
  elements.copyButton.disabled = items.length === 0;
  elements.downloadButton.disabled = items.length === 0;
  elements.utteranceList.replaceChildren();
  for (const item of items) {
    const card = document.createElement("article");
    card.className = "utterance";
    const head = document.createElement("div");
    head.className = "utterance-head";
    const dot = document.createElement("span");
    dot.className = "speaker-dot";
    const speaker = document.createElement("strong");
    speaker.textContent = `Спикер ${Number(item.speaker) + 1}`;
    const time = document.createElement("time");
    time.textContent = `${formatTime(item.start)} – ${formatTime(item.end)}`;
    const text = document.createElement("p");
    text.textContent = item.text || "";
    head.append(dot, speaker, time);
    card.append(head, text);
    elements.utteranceList.append(card);
  }
}

function setProcessing(active) {
  elements.processingStatus.hidden = !active;
  elements.cancelButton.hidden = !active;
  elements.transcribeButton.disabled = active || !state.file;
  elements.audioFile.disabled = active;
  elements.dropZone.setAttribute("aria-disabled", String(active));
}

elements.cancelButton.addEventListener("click", () => state.controller?.abort());
elements.transcribeButton.addEventListener("click", async () => {
  if (!state.file || !state.token) return;
  clearMessage();
  const controller = new AbortController();
  state.controller = controller;
  const body = new FormData();
  body.append("file", state.file);
  setProcessing(true);
  try {
    const result = await request("/v1/transcriptions", {
      method: "POST", headers: { Authorization: `Bearer ${state.token}` }, body, signal: controller.signal,
    });
    state.utterances = Array.isArray(result.utterances) ? result.utterances : [];
    renderResults();
    if (!state.utterances.length) showMessage("Речь в записи не найдена.");
  } catch (error) {
    if (controller.signal.aborted) showMessage("Распознавание отменено.");
    else if (error.status === 401) logout(error.message);
    else showMessage(error.message, true);
  } finally {
    if (state.controller === controller) state.controller = null;
    if (state.token) setProcessing(false);
  }
});

elements.copyButton.addEventListener("click", async () => {
  try {
    await navigator.clipboard.writeText(transcriptText());
    showMessage("Текст скопирован.");
  } catch {
    showMessage("Не удалось скопировать текст. Используйте скачивание файла.", true);
  }
});

elements.downloadButton.addEventListener("click", () => {
  const blob = new Blob([transcriptText()], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${(state.file?.name || "voxtext").replace(/\.[^.]+$/, "")}-transcript.txt`;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});
