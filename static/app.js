const MEDIA_ROOT = "/";

const state = {
  path: MEDIA_ROOT,
  parent: MEDIA_ROOT,
  files: [],
};

const $ = (id) => document.getElementById(id);

const loginView = $("login-view");
const browserView = $("browser-view");
const playerView = $("player-view");
const fileList = $("file-list");
const breadcrumbs = $("breadcrumbs");
const browserError = $("browser-error");
const loginError = $("login-error");
const video = $("video");

async function api(url, options = {}) {
  const response = await fetch(url, {
    credentials: "same-origin",
    ...options,
  });

  if (response.status === 401) {
    showLogin();
    throw new Error("Unauthorized");
  }

  if (!response.ok) {
    throw new Error(await response.text() || "Request failed");
  }

  return response;
}

async function checkSession() {
  try {
    await api("/api/me");
    showBrowser();
    await loadDirectory(MEDIA_ROOT);
  } catch {
    showLogin();
  }
}

function showLogin() {
  loginView.classList.remove("hidden");
  browserView.classList.add("hidden");
  playerView.classList.add("hidden");
  $("logout").classList.add("hidden");
  video.pause();
  video.removeAttribute("src");
  video.querySelectorAll("track").forEach((track) => track.remove());
  $("subtitle").replaceChildren(new Option("Off", ""));
  $("subtitle").disabled = true;
}

function showBrowser() {
  loginView.classList.add("hidden");
  browserView.classList.remove("hidden");
  playerView.classList.add("hidden");
  $("logout").classList.remove("hidden");
}

function showPlayer(name, filePath) {
  loginView.classList.add("hidden");
  browserView.classList.add("hidden");
  playerView.classList.remove("hidden");
  $("logout").classList.remove("hidden");
  $("now-playing").textContent = name;

  video.pause();
  video.removeAttribute("src");
  video.querySelectorAll("track").forEach((track) => track.remove());
  video.load();

  const streamUrl = `/api/stream?path=${encodeURIComponent(filePath)}`;
  video.src = streamUrl;
  video.load();

  setupSubtitles(name, filePath);
  video.play().catch(() => {});
}

function formatSize(bytes) {
  if (bytes === 0) return "0 bytes";
  const units = ["bytes", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / Math.pow(1024, i)).toFixed(i ? 2 : 0)} ${units[i]}`;
}

function isVideo(file) {
  return /\.(mp4|m4v|webm|ogv|ogg|mov|mkv|avi|wmv|flv|ts|mts|m2ts|mpeg|mpg|3gp|3g2|vob)$/i.test(file.path);
}

function parentPath(p) {
  if (p === MEDIA_ROOT) return MEDIA_ROOT;
  const parts = p.split("/").filter(Boolean);
  parts.pop();
  const parent = "/" + parts.join("/");
  if (!parent || parent === "/") return MEDIA_ROOT;
  return parent;
}

function renderBreadcrumbs(p) {
  breadcrumbs.replaceChildren();
  const root = document.createElement("a");
  root.href = "#";
  root.textContent = "Root";
  root.onclick = (e) => {
    e.preventDefault();
    loadDirectory("/");
  };
  breadcrumbs.append(root);

  const parts = p.split("/").filter(Boolean);
  let current = "";
  for (const part of parts) {
    current += "/" + part;
    const sep = document.createTextNode(" / ");
    const link = document.createElement("a");
    link.href = "#";
    link.textContent = part;
    const target = current;
    link.onclick = (e) => {
      e.preventDefault();
      loadDirectory(target);
    };
    breadcrumbs.append(sep, link);
  }
}

async function loadDirectory(p) {
  browserError.textContent = "";
  showBrowser();
  state.path = p;
  state.parent = parentPath(p);
  renderBreadcrumbs(p);
  $("up").disabled = p === MEDIA_ROOT;

  try {
    const response = await api(`/api/browse?path=${encodeURIComponent(p)}`);
    const data = await response.json();
    state.files = data.fileInfos || [];
    renderFiles(state.files);
  } catch (error) {
    browserError.textContent = error.message;
  }
}

function fileName(filePath) {
  return filePath.split("/").filter(Boolean).pop() || filePath;
}

function subtitleLabel(filePath, videoName) {
  const name = fileName(filePath);
  const videoStem = videoName.replace(/\.[^.]+$/, "").toLowerCase();
  const subtitleStem = name.replace(/\.[^.]+$/, "").toLowerCase();
  if (subtitleStem === videoStem) return "Subtitle";

  let language = subtitleStem.startsWith(videoStem + ".")
    ? subtitleStem.slice(videoStem.length + 1)
    : "";
  language = language.replace(/[-_.]/g, " ").trim();
  return language ? language.toUpperCase() : "Subtitle";
}

function findSubtitles(videoName, videoPath) {
  const directory = videoPath.substring(0, videoPath.lastIndexOf("/")) || "/";
  const videoStem = videoName.replace(/\.[^.]+$/, "").toLowerCase();
  const sidecars = state.files.filter((file) => {
    if (file.isDir) return false;
    const lower = file.path.toLowerCase();
    if (!lower.endsWith(".srt") && !lower.endsWith(".vtt")) return false;
    const sameDirectory = (file.path.substring(0, file.path.lastIndexOf("/")) || "/") === directory;
    if (!sameDirectory) return false;
    const stem = fileName(file.path).replace(/\.[^.]+$/, "").toLowerCase();
    return stem === videoStem || stem.startsWith(videoStem + ".");
  });

  if (sidecars.length === 0) {
    const allInDirectory = state.files.filter((file) => {
      if (file.isDir) return false;
      const lower = file.path.toLowerCase();
      const sameDirectory = (file.path.substring(0, file.path.lastIndexOf("/")) || "/") === directory;
      return sameDirectory && (lower.endsWith(".srt") || lower.endsWith(".vtt"));
    });
    return allInDirectory.length === 1 ? allInDirectory : [];
  }

  return sidecars;
}

function setupSubtitles(videoName, videoPath) {
  const select = $("subtitle");
  select.replaceChildren();
  const off = document.createElement("option");
  off.value = "";
  off.textContent = "Off";
  select.append(off);

  const subtitles = findSubtitles(videoName, videoPath);
  for (const subtitle of subtitles) {
    const option = document.createElement("option");
    option.value = subtitle.path;
    option.textContent = subtitleLabel(subtitle.path, videoName);
    select.append(option);
  }

  select.disabled = subtitles.length === 0;
  if (subtitles.length === 0) return;

  const defaultSubtitle =
    subtitles.find((file) => fileName(file.path).replace(/\.[^.]+$/, "").toLowerCase() === videoName.replace(/\.[^.]+$/, "").toLowerCase()) ||
    subtitles.find((file) => {
      const lower = fileName(file.path).toLowerCase();
      return /(^|[._-])(nl|dut|dutch|nederlands)([._-]|$)/.test(lower);
    });
  if (defaultSubtitle) {
    select.value = defaultSubtitle.path;
    loadSubtitle(defaultSubtitle.path);
  }
}

function loadSubtitle(filePath) {
  video.querySelectorAll("track").forEach((track) => track.remove());
  for (const track of video.textTracks) track.mode = "disabled";

  if (!filePath) return;

  const track = document.createElement("track");
  track.kind = "subtitles";
  track.label = fileName(filePath);
  track.srclang = "und";
  track.src = `/api/subtitle?path=${encodeURIComponent(filePath)}`;
  track.default = true;
  video.append(track);

  track.addEventListener("load", () => {
    if (track.track) track.track.mode = "showing";
  });
}

function renderFiles(files) {
  fileList.replaceChildren();

  files.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.path.localeCompare(b.path, undefined, { sensitivity: "base" });
  });

  if (state.path !== MEDIA_ROOT) {
    addRow("↩", "..", "", "Open", () => loadDirectory(state.parent));
  }

  for (const file of files) {
    const name = file.path.split("/").filter(Boolean).pop() || file.path;
    if (file.isDir) {
      addRow("📁", name, "", "Open", () => loadDirectory(file.path));
      continue;
    }

    const action = isVideo(file) ? "Watch" : "";
    addRow("📄", name, formatSize(file.size), action, () => {
      if (isVideo(file)) showPlayer(name, file.path);
    });
  }
}

function addRow(icon, name, size, action, callback) {
  const row = document.createElement("div");
  row.className = "file-row";

  const iconEl = document.createElement("div");
  iconEl.className = "file-icon";
  iconEl.textContent = icon;

  const nameEl = document.createElement("a");
  nameEl.className = "file-name";
  nameEl.href = "#";
  nameEl.textContent = name;
  nameEl.onclick = (e) => {
    e.preventDefault();
    callback();
  };

  const sizeEl = document.createElement("div");
  sizeEl.className = "file-size";
  sizeEl.textContent = size;

  const actionEl = document.createElement("div");
  actionEl.className = "file-action";

  if (action) {
    const button = document.createElement("button");
    button.className = "watch-button";
    button.type = "button";
    button.textContent = action;
    button.onclick = callback;
    actionEl.append(button);
  }

  row.append(iconEl, nameEl, sizeEl, actionEl);
  fileList.append(row);
}

$("login-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  loginError.textContent = "";

  try {
    const response = await fetch("/api/login", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        username: $("username").value,
        password: $("password").value,
      }),
    });

    if (!response.ok) {
      loginError.textContent = "Invalid username or password.";
      return;
    }

    $("password").value = "";
    showBrowser();
    await loadDirectory(MEDIA_ROOT);
  } catch {
    loginError.textContent = "Unable to connect to the player.";
  }
});

$("logout").onclick = async () => {
  await fetch("/api/logout", { method: "POST", credentials: "same-origin" });
  showLogin();
};

$("up").onclick = () => {
  if (state.path !== MEDIA_ROOT) loadDirectory(state.parent);
};

$("back").onclick = () => {
  video.pause();
  video.removeAttribute("src");
  showBrowser();
  loadDirectory(state.path);
};

$("volume").oninput = (event) => {
  video.volume = Number(event.target.value);
};

$("subtitle").onchange = (event) => {
  loadSubtitle(event.target.value);
};

$("fullscreen").onclick = async () => {
  if (video.requestFullscreen) {
    await video.requestFullscreen();
  } else if (video.webkitEnterFullscreen) {
    video.webkitEnterFullscreen();
  }
};

video.addEventListener("error", () => {
  const error = video.error;
  const message = error?.code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED
    ? "This video format or codec is not supported by this browser."
    : "The video could not be played.";
  const hint = document.querySelector("#player-view .hint");
  if (hint) hint.textContent = message;
});

checkSession();
