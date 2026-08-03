const header = document.querySelector("[data-header]");
const copyButton = document.querySelector("[data-copy]");
const command = document.querySelector("[data-command]");
const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

const updateHeader = () => {
  header?.classList.toggle("scrolled", window.scrollY > 24);
};

updateHeader();
window.addEventListener("scroll", updateHeader, { passive: true });

if (reducedMotion) {
  document.querySelectorAll(".reveal").forEach((element) => element.classList.add("visible"));
} else {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add("visible");
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.12 }
  );

  document.querySelectorAll(".reveal").forEach((element) => observer.observe(element));
}

copyButton?.addEventListener("click", async () => {
  const label = copyButton.querySelector(".copy-label");

  try {
    await navigator.clipboard.writeText(command?.textContent?.replace(/^\$ /gm, "") ?? "");
    if (label) label.textContent = "Copied";
  } catch {
    if (label) label.textContent = "Select text";
  }

  window.setTimeout(() => {
    if (label) label.textContent = "Copy";
  }, 1800);
});
