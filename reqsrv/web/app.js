async function api(path, body) {
  const res = await fetch(path, {
    method: body ? "POST" : "GET",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(await res.text());
  return await res.json();
}

function addMsg(role, text) {
  const el = document.createElement("div");
  el.className = `msg ${role}`;
  el.textContent = (role === "user" ? "You: " : "Assistant: ") + text;
  document.getElementById("chatlog").appendChild(el);
  el.scrollIntoView({ block: "end" });
}

async function refreshDocs() {
  const st = await api("/api/state");
  const sel = document.getElementById("docSelect");
  sel.innerHTML = "";
  st.docs.forEach(d => {
    const opt = document.createElement("option");
    opt.value = d.id;
    opt.textContent = `${d.title} (${d.id})`;
    sel.appendChild(opt);
  });
  await loadSelectedDoc();
}

async function loadSelectedDoc() {
  const sel = document.getElementById("docSelect");
  const id = sel.value;
  if (!id) {
    document.getElementById("docView").textContent = "(no documents yet)";
    return;
  }
  const res = await fetch(`/api/docs/${id}`);
  document.getElementById("docView").textContent = await res.text();
}

document.getElementById("send").onclick = async () => {
  const msgEl = document.getElementById("msg");
  const text = msgEl.value.trim();
  if (!text) return;
  msgEl.value = "";
  addMsg("user", text);

  try {
    const out = await api("/api/chat", { message: text });
    addMsg("assistant", out.assistant);
    await refreshDocs();
  } catch (e) {
    addMsg("assistant", "ERROR: " + e.message);
  }
};

document.getElementById("refresh").onclick = refreshDocs;
document.getElementById("docSelect").onchange = loadSelectedDoc;

refreshDocs();
