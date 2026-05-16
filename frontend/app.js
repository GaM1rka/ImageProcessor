const API_URL = window.API_URL || "http://localhost:8080";
const form = document.querySelector("#upload-form");
const input = document.querySelector("#image-input");
const gallery = document.querySelector("#gallery");
const template = document.querySelector("#image-card-template");
const items = new Map();

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!input.files.length) {
    return;
  }

  const body = new FormData();
  body.append("image", input.files[0]);

  const response = await fetch(`${API_URL}/upload`, {
    method: "POST",
    body,
  });
  const image = await response.json();
  input.value = "";
  renderCard(image);
  pollStatus(image.id);
});

function renderCard(image) {
  const fragment = template.content.cloneNode(true);
  const card = fragment.querySelector(".image-card");
  const preview = fragment.querySelector(".preview");
  const img = fragment.querySelector("img");
  const name = fragment.querySelector(".name");
  const status = fragment.querySelector(".status");
  const del = fragment.querySelector(".delete");

  name.textContent = image.originalName || image.id;
  status.textContent = statusText(image.status);
  del.addEventListener("click", async () => {
    await fetch(`${API_URL}/image/${image.id}`, { method: "DELETE" });
    card.remove();
    items.delete(image.id);
  });

  gallery.prepend(card);
  items.set(image.id, { card, preview, img, status });
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
  item.status.textContent = statusText(image.status);

  if (image.status === "done") {
    item.preview.classList.add("ready");
    item.img.src = `${API_URL}/image/${id}?v=${Date.now()}`;
    return;
  }

  if (image.status !== "failed") {
    window.setTimeout(() => pollStatus(id), 1500);
  }
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
