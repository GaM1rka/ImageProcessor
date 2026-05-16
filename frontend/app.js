const API_URL = window.API_URL || "http://localhost:8080";
const form = document.querySelector("#upload-form");
const input = document.querySelector("#image-input");
const uploadButton = document.querySelector("#upload-button");
const gallery = document.querySelector("#gallery");
const message = document.querySelector("#message");
const emptyState = document.querySelector("#empty-state");
const template = document.querySelector("#image-card-template");
const items = new Map();

loadImages();

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!input.files.length) {
    return;
  }

  const body = new FormData();
  body.append("image", input.files[0]);

  setMessage("");
  setUploading(true);
  try {
    const response = await fetch(`${API_URL}/upload`, {
      method: "POST",
      body,
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}));
      setMessage(payload.error || "Не удалось загрузить изображение");
      return;
    }
    const image = await response.json();
    input.value = "";
    renderCard(image);
    pollStatus(image.id);
  } finally {
    setUploading(false);
  }
});

async function loadImages() {
  try {
    const response = await fetch(`${API_URL}/images`);
    if (!response.ok) {
      setMessage("API пока недоступен");
      return;
    }

    const images = await response.json();
    images.reverse().forEach((image) => {
      renderCard(image);
      if (image.status === "done") {
        showReady(image.id);
      } else if (image.status !== "failed") {
        pollStatus(image.id);
      }
    });
  } catch {
    setMessage("API пока недоступен");
  }
  syncEmptyState();
}

function renderCard(image) {
  const fragment = template.content.cloneNode(true);
  const card = fragment.querySelector(".image-card");
  const preview = fragment.querySelector(".preview");
  const img = fragment.querySelector("img");
  const name = fragment.querySelector(".name");
  const status = fragment.querySelector(".status");
  const del = fragment.querySelector(".delete");

  name.textContent = image.originalName || image.id;
  setStatus({ status }, image.status);
  del.addEventListener("click", async () => {
    const response = await fetch(`${API_URL}/image/${image.id}`, { method: "DELETE" });
    if (!response.ok) {
      setMessage("Не удалось удалить изображение");
      return;
    }
    removeCard(image.id);
  });

  gallery.prepend(card);
  items.set(image.id, { card, preview, img, status });
  syncEmptyState();
}

async function pollStatus(id) {
  const item = items.get(id);
  if (!item) {
    return;
  }

  const response = await fetch(`${API_URL}/status/${id}`);
  if (!response.ok) {
    item.status.textContent = "Не найдено";
    return;
  }

  const image = await response.json();
  setStatus(item, image.status);

  if (image.status === "done") {
    showReady(id);
    return;
  }

  if (image.status !== "failed") {
    window.setTimeout(() => pollStatus(id), 1500);
  }
}

function showReady(id) {
  const item = items.get(id);
  if (!item) {
    return;
  }
  item.preview.classList.add("ready");
  item.img.src = `${API_URL}/image/${id}/thumbnail?v=${Date.now()}`;
}

function removeCard(id) {
  const item = items.get(id);
  if (!item) {
    return;
  }
  item.card.remove();
  items.delete(id);
  syncEmptyState();
}

function setStatus(item, status) {
  item.status.textContent = statusText(status);
  item.status.classList.toggle("failed", status === "failed");
}

function setUploading(isUploading) {
  uploadButton.disabled = isUploading;
  uploadButton.textContent = isUploading ? "Загрузка" : "Загрузить";
}

function setMessage(text) {
  message.textContent = text;
  message.hidden = !text;
}

function syncEmptyState() {
  emptyState.hidden = items.size > 0;
}

function statusText(status) {
  const labels = {
    queued: "В очереди",
    processing: "Обрабатывается",
    done: "Готово",
    failed: "Ошибка",
  };
  return labels[status] || status;
}
