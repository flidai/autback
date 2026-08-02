const tabs = [...document.querySelectorAll('[role="tab"]')];
const panels = [...document.querySelectorAll('[role="tabpanel"]')];
const tablist = document.querySelector('[role="tablist"]');
const menuButton = document.querySelector('.menu-button');
const navigation = document.querySelector('#main-navigation');
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

let activeIndex = 0;
let rotation;

function activateTab(index, moveFocus = false) {
  activeIndex = (index + tabs.length) % tabs.length;
  tabs.forEach((tab, tabIndex) => {
    const active = tabIndex === activeIndex;
    tab.setAttribute('aria-selected', String(active));
    tab.tabIndex = active ? 0 : -1;
    if (moveFocus && active) tab.focus();
  });
  panels.forEach((panel, panelIndex) => {
    panel.hidden = panelIndex !== activeIndex;
  });
}

function stopRotation() {
  window.clearInterval(rotation);
}

function startRotation() {
  stopRotation();
  if (reduceMotion.matches || document.hidden) return;
  rotation = window.setInterval(() => activateTab(activeIndex + 1), 7000);
}

tabs.forEach((tab, index) => {
  tab.addEventListener('click', () => {
    activateTab(index);
    startRotation();
  });
});

tablist?.addEventListener('keydown', (event) => {
  const keys = ['ArrowLeft', 'ArrowRight', 'Home', 'End'];
  if (!keys.includes(event.key)) return;
  event.preventDefault();
  if (event.key === 'Home') activateTab(0, true);
  if (event.key === 'End') activateTab(tabs.length - 1, true);
  if (event.key === 'ArrowLeft') activateTab(activeIndex - 1, true);
  if (event.key === 'ArrowRight') activateTab(activeIndex + 1, true);
  startRotation();
});

document.querySelector('.product-window')?.addEventListener('mouseenter', stopRotation);
document.querySelector('.product-window')?.addEventListener('mouseleave', startRotation);
document.addEventListener('visibilitychange', startRotation);
reduceMotion.addEventListener('change', startRotation);

menuButton?.addEventListener('click', () => {
  const open = menuButton.getAttribute('aria-expanded') === 'true';
  menuButton.setAttribute('aria-expanded', String(!open));
  navigation.toggleAttribute('data-open', !open);
});

function closeMenu({ restoreFocus = false } = {}) {
  menuButton?.setAttribute('aria-expanded', 'false');
  navigation?.removeAttribute('data-open');
  if (restoreFocus) menuButton?.focus();
}

navigation?.addEventListener('click', (event) => {
  if (!(event.target instanceof HTMLAnchorElement)) return;
  closeMenu();
});

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && menuButton?.getAttribute('aria-expanded') === 'true') {
    closeMenu({ restoreFocus: true });
  }
});

activateTab(0);
startRotation();
