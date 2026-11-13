/* components/titlebar.js — 自绘顶栏的交互入口
   1) 模块切换下拉（与左侧导航共用同一份 groups 数据）
   2) 当前模块面包屑
   3) 窗口控制按钮（仅 mygo 桌面模式，浏览器降级模式自动隐藏） */
(function () {
  'use strict';
  var UI = window.ResumeUI = window.ResumeUI || {};
  var $ = function (id) { return document.getElementById(id); };
  var btn = $('btnModules'), menu = $('moduleMenu'), crumb = $('crumb');
  var groups = [], active = '';

  function closeMenu() {
    menu.hidden = true;
    btn.setAttribute('aria-expanded', 'false');
  }

  function renderMenu() {
    menu.innerHTML = '';
    groups.forEach(function (g, i) {
      var b = document.createElement('button');
      b.type = 'button';
      b.setAttribute('role', 'menuitem');
      b.className = g.id === active ? 'active' : '';
      var num = document.createElement('span');
      num.className = 'menu-num';
      num.textContent = String(i + 1).padStart(2, '0');
      var name = document.createElement('span');
      name.textContent = g.title;
      b.append(num, name);
      b.addEventListener('click', function () {
        closeMenu();
        document.dispatchEvent(new CustomEvent('resume:select-group', { detail: { id: g.id } }));
      });
      menu.appendChild(b);
    });
  }

  btn.addEventListener('click', function (e) {
    e.stopPropagation();
    if (menu.hidden) { renderMenu(); menu.hidden = false; btn.setAttribute('aria-expanded', 'true'); }
    else closeMenu();
  });
  document.addEventListener('click', function (e) {
    if (!menu.hidden && !menu.contains(e.target)) closeMenu();
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeMenu();
  });

  document.addEventListener('resume:groups', function (e) {
    groups = (e.detail && e.detail.groups) || [];
  });
  document.addEventListener('resume:group', function (e) {
    var d = e.detail || {};
    active = d.id || '';
    crumb.textContent = d.title || '内容模块';
    if (!menu.hidden) renderMenu();
  });

  // 原生标题栏已隐藏，窗口由页面上的三个按钮驱动
  var w = (window.mygo && window.mygo.window) || null;
  if (w) {
    $('winMin').addEventListener('click', function () { w.minimize(); });
    $('winMax').addEventListener('click', function () { w.toggleMaximize(); });
    $('winClose').addEventListener('click', function () { w.close(); });
  }

  UI.titlebar = { closeMenu: closeMenu };
})();
