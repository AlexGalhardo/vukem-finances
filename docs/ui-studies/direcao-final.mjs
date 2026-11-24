const scenarios = [
  ["income", "Nova receita"], ["expense", "Nova despesa"], ["import", "Importar transações"],
  ["import-review", "Revisar importação"], ["edit", "Editar transação"], ["delete", "Excluir transação"],
  ["page-size", "Escolher linhas por página"], ["search", "Busca por mercado"], ["empty", "Busca sem resultados"],
  ["page-2", "Transações · Página 2"], ["size-50", "Transações · 50 por página"],
  ["size-75", "Transações · 75 por página"], ["size-100", "Transações · 100 por página"],
];
export const finalScenarios = scenarios.map(([id, title]) => ({ id: `Review/${id}`, title }));

const icons = {
  plus: "M11 4h2v7h7v2h-7v7h-2v-7H4v-2h7z",
  down: "M11 4h2v12l4-4 1.5 1.5-6.5 6.5-6.5-6.5L7 12l4 4z",
  up: "M11 20h2V8l4 4 1.5-1.5L12 4l-6.5 6.5L7 12l4-4z",
  upload: "M4 14h2v5h12v-5h2v7H4z M11 4h2l5 5-1.5 1.5L13 7v9h-2V7l-3.5 3.5L6 9z",
  edit: "M3 17v4h4l12-12-4-4z M16 4l2-2 4 4-2 2z",
  trash: "M8 3h8v2h5v2H3V5h5z M6 8h2v11h8V8h2v13H6z M10 9h1v8h-1z M13 9h1v8h-1z",
  search: "M10 3a7 7 0 1 0 4.2 12.6L20 21l1.4-1.4-5.5-5.7A7 7 0 0 0 10 3z M10 5a5 5 0 1 1 0 10 5 5 0 0 1 0-10z",
  moon: "M15 2a10 10 0 1 0 7 13 8 8 0 0 1-7-13z",
  sun: "M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10z M11 1h2v4h-2z M11 19h2v4h-2z M1 11h4v2H1z M19 11h4v2h-4z M4 3l3 3-1 1-3-3z M20 3l1 1-3 3-1-1z M20 21l-3-3 1-1 3 3z M4 21l-1-1 3-3 1 1z",
  chevron: "M5 8l7 7 7-7 1.5 1.5-8.5 8.5L3.5 9.5z",
  close: "M5 3.5l7 7 7-7L20.5 5l-7 7 7 7-1.5 1.5-7-7-7 7L3.5 19l7-7-7-7z",
  home: "M2 10l10-8 10 8-1.5 1.5L20 11v10h-6v-6h-4v6H4V11l-.5.5z",
  list: "M4 5h16v2H4z M4 11h16v2H4z M4 17h16v2H4z",
  wallet: "M3 5h17v3h2v10h-2v2H3z M13 10v6h7v-6z",
  dots: "M4 10a2 2 0 1 0 0 4 2 2 0 0 0 0-4z M12 10a2 2 0 1 0 0 4 2 2 0 0 0 0-4z M20 10a2 2 0 1 0 0 4 2 2 0 0 0 0-4z",
  left: "M15 4l-8 8 8 8 1.5-1.5L10 12l6.5-6.5z",
  right: "M9 4l8 8-8 8-1.5-1.5L14 12 7.5 5.5z",
};

function icon(g, name, x, y, color, size = 18) {
  g.path(`Ícone · ${name}`, x, y, size, size, icons[name], color);
}

export function drawFinalNavIcon(g, c, title, x, y, selected) {
  const glyph = { "Visão geral": "home", Contas: "wallet", Cartões: "wallet", Transações: "list", Relatórios: "list",
    Configurações: "dots", Categorias: "list", Orçamentos: "wallet", Metas: "up", Recorrências: "home" }[title] ?? "list";
  icon(g, glyph, x, y, selected ? c.accent : c.muted, 14);
}

function label(g, c, name, x, y, width, value, size = 14, weight = 500, options = {}) {
  const { color = c.ink, ...layout } = options;
  g.text(name, x, y, width, value, size, color, weight, layout);
}

function action(g, c, { title, x, y, width, mode = "secondary", symbol, to }) {
  const fill = mode === "primary" ? c.accent : mode === "danger" ? c.negative : c.surface;
  const ink = mode === "primary" ? c.onAccent : mode === "danger" ? c.bg : c.ink;
  g.rect(title, x, y, width, 44, fill, 8, mode === "secondary" ? c.control ?? c.border : undefined);
  if (symbol) icon(g, symbol, x + 12, y + 13, ink);
  label(g, c, "Rótulo", x + (symbol ? 38 : 8), y + 12, width - (symbol ? 44 : 16), title, 13, 720, { color: ink, align: symbol ? "LEFT" : "CENTER" });
  if (to) g.link(title, x, y, width, 44, to);
}

function destination(id, edition, theme) {
  const scenario = finalScenarios.find(frame => frame.id === id);
  const title = scenario?.title ?? ({ "Mobile/Transactions": "Transações", "Mobile/More": "Mais", "Mobile/Accounts": "Contas", Transactions: "Transações" }[id] ?? "Visão geral");
  return `${title} · ${id} · ${edition} · ${theme}`;
}

export function drawFinalSidebarActions(g, c, { rail, top, h, theme, admin, edition }) {
  const target = id => destination(`Review/${id}`, edition, theme);
  if (!admin) {
    action(g, c, { title: "Nova receita", x: 15, y: top + 105, width: rail - 30, mode: "primary", symbol: "down", to: target("income") });
    action(g, c, { title: "Nova despesa", x: 15, y: top + 153, width: rail - 30, symbol: "up", to: target("expense") });
    action(g, c, { title: "Importar arquivo", x: 15, y: top + 201, width: rail - 30, symbol: "upload", to: target("import") });
    g.link("Abrir visão geral", 12, top + 248, rail - 24, 37, destination("Dashboard", edition, theme));
    g.link("Abrir transações", 12, top + 285, rail - 24, 37, destination("Transactions", edition, theme));
  }
  const x = 15;
  const y = h - 144;
  const width = rail - 30;
  const split = width / 2;
  g.rect("Tema da interface", x, y, width, 44, c.surface, 10, c.control ?? c.border);
  g.rect("Tema selecionado", theme === "Claro" ? x + 3 : x + split, y + 3, split - 3, 38, c.soft, 8);
  for (const [i, option] of ["Claro", "Escuro"].entries()) {
    const bx = x + i * split;
    icon(g, i ? "moon" : "sun", bx + 8, y + 12, option === theme ? c.accent : c.muted, 19);
    label(g, c, "Tema", bx + 35, y + 13, split - 38, option, 12, 700);
    if (!admin && option !== theme) g.link(`Trocar para ${option}`, bx, y, split, 44, destination("Dashboard", edition, option));
  }
}

const expenses = [["Moradia", 42], ["Alimentação", 26], ["Transporte", 18], ["Outros", 14]];
const income = [["Salário", 78], ["Freelance", 17], ["Outras receitas", 5]];
const entries = [
  ["24 nov", "Mercado do bairro", "Alimentação", "Conta principal", "− R$ 186,42"],
  ["23 nov", "Freelance", "Serviços", "Conta principal", "+ R$ 1.152,00"],
  ["22 nov", "Internet", "Moradia", "Conta principal", "− R$ 119,90"],
  ["21 nov", "Mercado Central", "Alimentação", "Cartão pessoal", "− R$ 98,70"],
  ["20 nov", "Combustível", "Transporte", "Cartão pessoal", "− R$ 165,22"],
  ["19 nov", "Padaria", "Alimentação", "Cartão pessoal", "− R$ 24,50"],
];

function ring(g, x, y, size, data, theme) {
  const colors = theme === "Claro" ? ["#3149B4", "#7179D7", "#157466", "#A55D1C"] : ["#AEBEFF", "#8B84EF", "#63CAB7", "#F7BA77"];
  const radius = size * .48;
  const inner = size * .33;
  const point = (angle, r) => [size / 2 + Math.cos(angle) * r, size / 2 + Math.sin(angle) * r].map(n => n.toFixed(2)).join(" ");
  let start = -Math.PI / 2;
  return data.map(([name, percent], i) => {
    const end = start + Math.PI * 2 * percent / 100;
    const a = start + 0.021;
    const b = end - 0.021;
    const large = b - a > Math.PI ? 1 : 0;
    const data = `M ${point(a, radius)} A ${radius} ${radius} 0 ${large} 1 ${point(b, radius)} L ${point(b, inner)} A ${inner} ${inner} 0 ${large} 0 ${point(a, inner)} Z`;
    g.path(`${name} · ${percent}%`, x, y, size, size, data, colors[i]);
    start = end;
    return colors[i];
  });
}

function chart(g, c, x, y, width, height, title, amount, data, theme, phone) {
  g.rect("Painel do gráfico", x, y, width, height, c.surface, 10, c.border);
  label(g, c, "Título do gráfico", x + 16, y + 14, width - 32, title, 16, 640);
  const size = phone ? 100 : 124;
  const rx = x + 18;
  const ry = y + (phone ? 45 : 54);
  const colors = ring(g, rx, ry, size, data, theme);
  label(g, c, "Total do gráfico", rx - 1, ry + size / 2 - 8, size + 2, amount, phone ? 10 : 12, 740, { align: "CENTER" });
  data.forEach(([name, percent], i) => {
    const lx = rx + size + 17;
    const rowY = ry + 4 + i * (phone ? 23 : 27);
    g.rect("Identificador da categoria", lx, rowY + 5, 9, 9, colors[i], 2);
    label(g, c, "Categoria", lx + 16, rowY, width - size - 76, name, 12, 600);
    label(g, c, "Participação", x + width - 58, rowY, 43, `${percent}%`, 12, 760, { align: "RIGHT" });
  });
}

function budget(g, c, x, y, width, phone, step = 2) {
  const used = step === 2 ? "R$ 544 de R$ 800" : "R$ 357,58 de R$ 800";
  const rest = step === 2 ? "Restam R$ 256" : "Restam R$ 442,42";
  g.rect("Contexto do orçamento", x, y, width, phone ? 92 : 94, c.soft, 12);
  label(g, c, "Orçamento", x + 14, y + 11, width - 28, "Orçamento de alimentação", phone ? 13 : 15, 730);
  label(g, c, "Disponível", x + 14, y + 32, width - 28, rest, phone ? 18 : 20, 800);
  const railY = y + 62;
  const railWidth = width - 30;
  g.rect("Trilho", x + 14, railY, railWidth, 6, c.border, 3);
  g.rect("Consumido", x + 14, railY, railWidth * (step === 2 ? .68 : .447), 6, c.accent, 3);
  g.rect("Limite", x + railWidth + 13, railY - 3, 2, 12, c.ink, 1);
  label(g, c, "Gasto e limite", x + 14, y + 72, width - 28, used, 10, 650, { color: c.muted, align: "RIGHT" });
}

function iconAction(g, c, name, symbol, x, y, color, to) {
  g.rect(name, x, y, 44, 44, c.surface, 8, ["left", "right"].includes(symbol) ? c.border : undefined);
  icon(g, symbol, x + 13, y + 13, color);
  if (to) g.link(name, x, y, 44, 44, to);
}

function transactionTable(g, c, { x, y, width, phone, scenario, target, ownPage }) {
  const filtered = scenario === "search" || scenario === "empty";
  const pageSize = scenario?.startsWith("size-") ? Number(scenario.slice(5)) : 25;
  const pageTwo = scenario === "page-2";
  const items = scenario === "search" ? entries.filter(row => row[1].startsWith("Mercado")) : scenario === "empty" ? []
    : pageTwo ? [["12 nov", "Farmácia", "Saúde", "Conta principal", "− R$ 82,00"], ...entries.slice(1).map(row => ["11 nov", ...row.slice(1)])] : entries;
  label(g, c, "Título da tabela", x, y, width, ownPage ? "Histórico de transações" : "Transações", 22, 800);
  const searchY = y + 36;
  const searchWidth = phone ? width : 368;
  g.rect("Pesquisar transações", x, searchY, searchWidth, 44, c.surface, 10, c.control ?? c.border);
  icon(g, "search", x + 13, searchY + 13, c.muted);
  label(g, c, "Pesquisa", x + 41, searchY + 12, searchWidth - 88, scenario === "search" ? "mercado" : scenario === "empty" ? "zoológico" : "Buscar transações", 14, 520, { color: filtered ? c.ink : c.muted });
  if (filtered) {
    icon(g, "close", x + searchWidth - 31, searchY + 14, c.muted, 16);
    g.link("Limpar pesquisa", x + searchWidth - 46, searchY, 44, 44, target("home"));
  } else {
    g.link("Ver exemplo de pesquisa", x, searchY, searchWidth, 44, target("search"));
  }
  if (!phone) label(g, c, "Resultado", x, searchY + 13, width, filtered ? `${items.length} resultados` : "124 transações em novembro", 13, 570, { color: c.muted, align: "RIGHT" });
  const tableY = searchY + 56;
  const rowH = phone ? 59 : 49;
  const rowCount = phone && !filtered ? 4 : items.length;
  const head = phone ? 0 : 36;
  const bodyHeight = rowCount ? rowCount * rowH : 90;
  g.rect("Área rolável da tabela", x, tableY, width, head + bodyHeight, c.surface, 10, c.border);
  if (!phone) {
    g.rect("Cabeçalho", x + 1, tableY + 1, width - 2, 35, c.soft, 9);
    [["Data", .014], ["Descrição", .115], ["Categoria", .38], ["Conta", .55], ["Valor", .69], ["Ações", .891]].forEach(([column, offset]) => {
      label(g, c, "Coluna", x + width * offset, tableY + 8, column === "Valor" ? width * .145 : 90, column, 12, 600, { color: c.muted, align: column === "Valor" ? "RIGHT" : "LEFT" });
    });
  }
  if (!rowCount) label(g, c, "Estado vazio", x + 14, tableY + head + 27, width - 28, "Nenhuma transação encontrada. Tente outro termo.", phone ? 13 : 15, 650, { align: "CENTER" });
  items.slice(0, rowCount).forEach(([day, desc, category, account, amount], i) => {
    const yy = tableY + head + i * rowH;
    g.rect("Divisor da linha", x + 1, yy, width - 2, 1, c.border);
    if (phone) {
      label(g, c, "Descrição", x + 10, yy + 8, 145, desc, 12, 740);
      label(g, c, "Categoria e data", x + 10, yy + 30, 146, `${category} · ${day}`, 10, 530, { color: c.muted });
      label(g, c, "Valor", x + width - 183, yy + 13, 79, amount, 10, 730, { align: "RIGHT", color: amount.startsWith("+") ? c.positive : c.negative });
    } else {
      label(g, c, "Data", x + width * .014, yy + 15, 92, day, 13, 540, { color: c.muted });
      label(g, c, "Descrição", x + width * .115, yy + 14, width * .26, desc, 14, 700);
      label(g, c, "Categoria", x + width * .38, yy + 15, width * .16, category, 13);
      label(g, c, "Conta", x + width * .55, yy + 15, width * .14, account, 13, 530, { color: c.muted });
      label(g, c, "Valor", x + width * .69, yy + 15, width * .145, amount, 13, 650, { align: "RIGHT", color: amount.startsWith("+") ? c.positive : c.negative });
    }
    iconAction(g, c, "Editar transação", "edit", x + width - 98, yy + (phone ? 7 : 2), c.muted, target("edit"));
    iconAction(g, c, "Excluir transação", "trash", x + width - 49, yy + (phone ? 7 : 2), c.muted, target("delete"));
  });
  const footerY = tableY + head + bodyHeight + 10;
  const selectX = phone ? x : x + width - 290;
  const result = filtered ? `${items.length} de ${items.length}` : pageTwo ? "26–50 de 124" : `1–${pageSize} de 124`;
  label(g, c, "Contagem", phone ? x + 131 : x, footerY + 12, phone ? width - 132 : 280, result, 13, 630, { color: c.muted });
  g.rect("Linhas por página", selectX, footerY, 126, 44, c.surface, 8, c.control ?? c.border);
  label(g, c, "Quantidade", selectX + 10, footerY + 13, 98, `${pageSize} por página`, 12, 650);
  icon(g, "chevron", selectX + 105, footerY + 15, c.muted, 14);
  g.link("Alterar linhas por página", selectX, footerY, 126, 44, target("page-size"));
  iconAction(g, c, "Página anterior", "left", x + width - 98, footerY, pageTwo ? c.accent : c.muted, pageTwo ? target("home") : undefined);
  iconAction(g, c, "Próxima página", "right", x + width - 48, footerY, !filtered && !pageTwo ? c.accent : c.muted,
    !filtered && !pageTwo && pageSize === 25 ? target("page-2") : undefined);
  if (scenario === "page-size") {
    const menuY = footerY - 172;
    g.rect("Opções de paginação", selectX, menuY, 156, 166, c.surface, 10, c.control ?? c.border);
    [25, 50, 75, 100].forEach((size, i) => {
      if (!i) g.rect("Padrão selecionado", selectX + 4, menuY + 5, 148, 37, c.soft, 7);
      label(g, c, "Opção de paginação", selectX + 13, menuY + 13 + i * 38, 132, `${size} transações`, 13, !i ? 750 : 570);
      g.link(`Mostrar ${size} por página`, selectX + 4, menuY + 5 + i * 38, 148, 37, target(i ? `size-${size}` : "home"));
    });
  }
}

function mockModal(g, c, scenario, x, y, width, phone, target) {
  const titles = {
    income: "Nova receita", expense: "Nova despesa", import: "Importar transações",
    "import-review": "Revisar importação", edit: "Editar transação", delete: "Excluir transação?",
  };
  const title = titles[scenario];
  const importFlow = scenario.startsWith("import");
  const removing = scenario === "delete";
  const height = removing ? 242 : importFlow ? 402 : 490;
  g.rect("Escurecimento do fundo", 0, 0, g.w, g.h, "#0A1021", 0, undefined, .53);
  g.rect("Superfície do modal", x, y, width, height, c.surface, 18, c.control ?? c.border);
  g.rect("Faixa de contexto", x + 1, y + 1, width - 2, 70, c.soft, 17);
  label(g, c, "Título do modal", x + 20, y + 19, width - 75, title, phone ? 23 : 26, 800);
  iconAction(g, c, "Fechar modal", "close", x + width - 56, y + 12, c.muted, target("home"));
  const field = (name, value, fx, fy, fw) => {
    label(g, c, "Rótulo do campo", fx, fy, fw, name, 13, 700);
    g.rect("Campo", fx, fy + 21, fw, 43, c.surface, 8, c.control ?? c.border);
    label(g, c, "Valor do campo", fx + 10, fy + 33, fw - 20, value, 14, 520);
  };
  const inner = width - 40;
  const half = (inner - 12) / 2;
  let primary;
  if (removing) {
    label(g, c, "Transação selecionada", x + 20, y + 89, inner, "Mercado do bairro · − R$ 186,42", 16, 760);
    label(g, c, "Consequência", x + 20, y + 123, inner, "A exclusão é permanente e atualiza seus saldos e relatórios.", 13, 550,
      { color: c.muted, height: 44 });
    primary = "Excluir transação";
  } else if (scenario === "import") {
    label(g, c, "Instrução", x + 20, y + 86, inner, "Escolha um CSV ou OFX para revisar antes de importar.", 13, 550, { height: 42 });
    g.rect("Área de envio", x + 20, y + 132, inner, 83, c.soft, 10, c.control ?? c.border);
    icon(g, "upload", x + 33, y + 150, c.accent, 22);
    label(g, c, "Arquivo", x + 65, y + 149, inner - 55, "Selecionar arquivo", 15, 760);
    label(g, c, "Formatos", x + 65, y + 177, inner - 55, "Arraste ou escolha um CSV ou OFX", 12, 540, { color: c.muted });
    field("Conta de destino", "Conta principal", x + 20, y + 228, inner);
    primary = "Revisar arquivo";
  } else if (scenario === "import-review") {
    label(g, c, "Resumo de importação", x + 20, y + 91, inner, "extrato-novembro.csv · Conta principal", 14, 740);
    [["Mercado, R$ 186,42", "Pronta"], ["Assinatura, R$ 39,90", "Duplicata"], ["Internet, R$ 119,90", "Data inválida"]].forEach(([entry, status], i) => {
      const at = y + 130 + i * 51;
      g.rect("Linha de prévia", x + 20, at, inner, 44, c.bg, 8, c.border);
      label(g, c, "Lançamento", x + 30, at + 12, inner - 110, entry, 12, 620);
      label(g, c, "Estado", x + 25, at + 12, inner - 11, status, 12, 700, { align: "RIGHT", color: [c.positive, c.warning, c.negative][i] });
    });
    primary = "Importar 1 transação";
  } else {
    const editing = scenario === "edit";
    field("Tipo", scenario === "income" ? "Receita" : "Despesa", x + 20, y + 88, half);
    if (!editing) {
      icon(g, "chevron", x + half - 1, y + 123, c.muted, 14);
      g.link("Alternar tipo da transação", x + 20, y + 109, half, 43, target(scenario === "income" ? "expense" : "income"));
    }
    field("Valor", editing ? "R$ 186,42" : "R$ 0,00", x + 32 + half, y + 88, half);
    field("Descrição", editing ? "Mercado do bairro" : "Ex.: compra no mercado", x + 20, y + 164, inner);
    field("Conta", "Conta principal", x + 20, y + 240, half);
    field("Categoria", editing ? "Alimentação" : "Selecionar", x + 32 + half, y + 240, half);
    field("Data", "24/11/2026", x + 20, y + 316, half);
    primary = editing ? "Salvar alterações" : scenario === "income" ? "Salvar receita" : "Salvar despesa";
  }
  const primaryW = phone ? 168 : 188;
  action(g, c, { title: "Cancelar", x: x + width - primaryW - 126, y: y + height - 62, width: 96, to: target("home") });
  action(g, c, { title: primary, x: x + width - primaryW - 20, y: y + height - 62, width: primaryW, mode: removing ? "danger" : "primary",
    to: scenario === "import" ? target("import-review") : target("home") });
}

export function drawFinalMobileNav(g, c, edition, theme, id) {
  const bottom = g.h - 78;
  g.rect("Navegação do celular", 0, bottom, g.w, 78, c.surface, 0, c.border);
  [["Início", "home", "Mobile/Home"], ["Transações", "list", "Mobile/Transactions"],
    ["Adicionar", "plus", "Review/expense"], ["Contas", "wallet", "Mobile/Accounts"],
    ["Mais", "dots", "Mobile/More"]].forEach(([title, symbol, to], i) => {
    const item = g.w / 5;
    const selected = id === to;
    icon(g, symbol, i * item + item / 2 - 10, bottom + 12, selected ? c.accent : c.muted, 20);
    label(g, c, "Aba", i * item, bottom + 38, item, title, 10, 690, { align: "CENTER", color: selected ? c.accent : c.muted });
    g.link(title, i * item, bottom, item, 72, destination(to, edition, theme));
  });
}

export function drawFinalDashboard(g, c, { entry, platform, edition, theme, x, top, inset }) {
  const phone = platform === "Mobile";
  const ownPage = entry.id.includes("Transactions");
  const scenario = entry.id.startsWith("Review/") ? entry.id.slice(7) : undefined;
  const target = id => destination(id === "home" ? (phone ? "Mobile/Home" : "Dashboard") : `Review/${id}`, edition, theme);
  const width = g.w - x - inset;
  const titleY = top + (phone ? 54 : 76);
  if (phone && entry.id === "Mobile/More") {
    label(g, c, "Título", x, titleY, width, "Mais", 29, 650, { family: "Inter" });
    action(g, c, { title: "Nova receita", x, y: titleY + 55, width, mode: "primary", symbol: "down", to: target("income") });
    action(g, c, { title: "Nova despesa", x, y: titleY + 107, width, symbol: "up", to: target("expense") });
    action(g, c, { title: "Importar arquivo", x, y: titleY + 159, width, symbol: "upload", to: target("import") });
    label(g, c, "Aparência", x, titleY + 232, width, "Aparência", 18, 800);
    action(g, c, { title: theme === "Claro" ? "Ativar tema escuro" : "Ativar tema claro", x, y: titleY + 268, width,
      symbol: theme === "Claro" ? "moon" : "sun", to: destination("Mobile/More", edition, theme === "Claro" ? "Escuro" : "Claro") });
    drawFinalMobileNav(g, c, edition, theme, entry.id);
    return;
  }
  label(g, c, "Título", x, titleY, width - (phone ? 0 : 170), ownPage ? "Transações" : "Visão geral", phone ? 29 : 38, 650, { family: "Inter" });
  const periodY = titleY + (phone ? 39 : 14);
  label(g, c, "Período", phone ? x : x + width - 225, periodY, phone ? width : 204, "Novembro de 2026", 13, 670,
    { color: c.muted, align: phone ? "LEFT" : "RIGHT" });
  if (!phone) icon(g, "chevron", x + width - 15, periodY + 4, c.muted, 13);
  let y = titleY + (phone ? 66 : 63);
  if (!ownPage) {
    g.rect("Resumo do mês", x, y, width, phone ? 112 : 91, c.surface, 10, c.border);
    const stats = [["Saldo total", "R$ 12.840,30", c.ink], ["Entradas", "+ R$ 6.800,00", c.positive], ["Saídas", "− R$ 4.551,94", c.negative]];
    stats.forEach(([name, amount, ink], i) => {
      if (i && !phone) g.rect("Separador de indicadores", x + i * width / 3, y + 17, 1, 56, c.border);
      const dx = x + 14 + (phone ? (i === 2 ? (width - 25) / 2 : 0) : i * width / 3);
      const dy = y + (phone ? (i ? 64 : 9) : 14);
      label(g, c, "Indicador", dx, dy, phone ? 147 : width / 3 - 26, name, phone ? 11 : 13, 680, { color: c.muted });
      label(g, c, "Valor", dx, dy + 19, phone ? 152 : width / 3 - 27, amount, phone ? (i ? 15 : 24) : (i ? 25 : 32), 650, { color: ink });
    });
    y += phone ? 123 : 105;
    const chartWidth = phone ? width : (width - 16) / 2;
    const chartHeight = phone ? 156 : 193;
    chart(g, c, x, y, chartWidth, chartHeight, "Para onde foi o dinheiro", "R$ 4.551,94", expenses, theme, phone);
    chart(g, c, phone ? x : x + chartWidth + 16, phone ? y + chartHeight + 11 : y, chartWidth, chartHeight, "De onde veio o dinheiro", "R$ 6.800,00", income, theme, phone);
    y += phone ? chartHeight * 2 + 20 : chartHeight + 12;
    budget(g, c, x, y, width, phone);
    y += 111;
  }
  transactionTable(g, c, { x, y, width, phone, scenario, target, ownPage });
  if (phone) drawFinalMobileNav(g, c, edition, theme, entry.id);
  if (["income", "expense", "import", "import-review", "edit", "delete"].includes(scenario)) {
    const modalW = phone ? g.w - 22 : scenario.startsWith("import") ? 584 : 532;
    mockModal(g, c, scenario, (g.w - modalW) / 2, phone ? 168 : 154, modalW, phone, target);
  }
}

function preview(g, c, theme, kind, x, y, width, height) {
  const mode = theme === "Claro" ? "light" : "dark";
  if (g.photo(`Imagem do produto · ${kind}`, x, y, width, height, `final-${kind}-${mode}`)) return;
  g.rect("Prévia do produto", x, y, width, height, c.surface, 12, c.border);
  label(g, c, "Prévia", x + 16, y + 16, width - 32, "Visão geral", 21, 800);
  budget(g, c, x + 16, y + 58, width - 32, width < 400);
}

export function drawFinalLanding(g, c, theme, phone, step) {
  const { w, h } = g;
  if (step !== undefined) {
    const titles = ["Antes da compra", "Uma despesa entrou", "O orçamento foi atualizado"];
    const detail = ["Confira quanto ainda cabe no mês.", "Mercado do bairro · − R$ 186,42", "O limite segue em R$ 800. Agora restam R$ 256."];
    label(g, c, "Cena", 34, 28, w - 68, titles[step], 32, 650, { family: "Inter" });
    label(g, c, "O que mudou", 34, 79, w - 68, detail[step], 17, 550, { color: c.muted });
    budget(g, c, 34, 130, w - 68, false, step);
    if (step > 0) {
      g.rect("Lançamento recente", 34, 245, w - 68, 63, c.surface, 9, c.border);
      icon(g, "up", 51, 266, c.negative);
      label(g, c, "Linha de transação", 83, 265, w - 136, "Mercado do bairro                         − R$ 186,42", 16, 720);
    }
    label(g, c, "Duração", 34, 343, 430, ["Estado inicial", "Entrada de 160 ms", "Atualização de 220 ms"][step], 12, 570, { color: c.muted });
    if (step < 2) action(g, c, { title: "Próxima cena", x: w - 175, y: 330, width: 141, mode: "primary", to: `Cena ${step + 2} · ${theme}` });
    return;
  }

  const pad = phone ? 23 : 76;
  const width = w - pad * 2;
  const cx = phone ? pad : 673;
  const cw = phone ? width : w - cx - pad;
  const head = phone ? 86 : 128;
  const headWidth = phone ? width : 548;
  label(g, c, "Marca", pad, 26, 245, "Vukem Finances", 18, 800);
  label(g, c, "Navegação", phone ? w - 131 : w - 570, 28, phone ? 109 : 492,
    phone ? "PT-BR · Menu" : "Recursos       Edições       Instalação       Documentação       PT-BR", 13, 630, { align: "RIGHT" });
  g.rect("Divisor do cabeçalho", pad, 65, width, 1, c.border);
  g.rect("Superfície do hero", cx - (phone ? 0 : 20), phone ? 357 : 106, cw + (phone ? 0 : 20), phone ? 282 : 462, c.soft, 25);
  label(g, c, "Título principal", pad, head, headWidth, "Entenda seu mês.\nPlaneje o próximo.", phone ? 33 : 59, 830,
    { family: "Inter", height: phone ? 117 : 168 });
  const textY = head + (phone ? 125 : 164);
  label(g, c, "Descrição", pad, textY, headWidth - (phone ? 0 : 26),
    "Organize contas, transações e orçamentos em um app de código aberto que roda no seu computador ou servidor.",
    phone ? 15 : 18, 540, { color: c.muted, height: 78 });
  action(g, c, { title: "Ver opções de instalação", x: pad, y: textY + 78, width: phone ? width : 220, mode: "primary" });
  if (!phone) action(g, c, { title: "Ver código no GitHub", x: pad + 235, y: textY + 78, width: 182 });
  label(g, c, "Disponibilidade", pad, textY + 137, headWidth, "Software autohospedado. Sem serviço hospedado.", 12, 630, { color: c.muted });
  preview(g, c, theme, "dashboard", cx + 13, phone ? 378 : 137, cw - 27, phone ? 238 : 410);
  label(g, c, "Legenda", cx + 17, phone ? 650 : 583, cw - 27, "Visão geral com dados fictícios", 11, 600, { color: c.muted });
  const sectionY = phone ? 705 : 674;
  label(g, c, "Título da seção", pad, sectionY, width, "Veja quanto ainda cabe no mês", phone ? 27 : 43, 660, { family: "Inter", height: 77 });
  const sectionCopyY = sectionY + (phone ? 74 : 62);
  label(g, c, "Descrição da seção", pad, sectionCopyY, phone ? width : 550, "Acompanhe o gasto, o limite e o valor que resta em cada orçamento.", 16, 550,
    { color: c.muted, height: 66 });
  budget(g, c, pad, sectionCopyY + (phone ? 69 : 58), phone ? width : 550, phone);
  const mobileX = phone ? pad + 39 : 819;
  const mobileY = phone ? sectionCopyY + 180 : sectionY - 31;
  const mobileW = phone ? width - 78 : 207;
  preview(g, c, theme, "phone", mobileX, mobileY, mobileW, phone ? 385 : 335);
  if (phone) {
    label(g, c, "Legenda da prévia", pad, mobileY + 391, width, "Prévia de design para celular", 11, 650,
      { color: c.muted, align: "CENTER" });
  } else {
    label(g, c, "Tarefa ilustrada", 1045, mobileY + 111, 295, "Registre rápido. Revise com calma.", 25, 650, { family: "Inter", height: 78 });
    label(g, c, "Resumo da tarefa", 1045, mobileY + 200, 270, "Prévia de design para celular, com dados de demonstração.", 14, 540, { color: c.muted, height: 75 });
  }
  const editionsY = phone ? mobileY + 426 : 1046;
  label(g, c, "Título das edições", pad, editionsY, width, "Escolha sua edição", phone ? 30 : 42, 660, { family: "Inter" });
  const cardWidth = phone ? width : (width - 20) / 2;
  [
    ["Pessoal", "Rode offline, com um espaço para suas finanças."],
    ["SaaS autohospedado", "Gerencie vários espaços, permissões e planos Free e Pro."],
  ].forEach(([name, description], i) => {
    const bx = pad + (phone ? 0 : i * (cardWidth + 20));
    const by = editionsY + 60 + (phone ? i * 100 : 0);
    g.rect("Edição", bx, by, cardWidth, 87, i ? c.surface : c.soft, 12, c.border);
    label(g, c, "Nome da edição", bx + 14, by + 11, cardWidth - 28, name, 17, 800);
    label(g, c, "Descrição da edição", bx + 14, by + 39, cardWidth - 28, description, 13, 550, { color: c.muted, height: 40 });
  });
  const installY = editionsY + (phone ? 278 : 209);
  g.rect("Sessão de instalação", pad, installY, width, phone ? 189 : 179, c.soft, 18);
  label(g, c, "Título de instalação", pad + 19, installY + 18, width - 38, "Comece com a instalação", phone ? 25 : 34, 660, { family: "Inter", height: 54 });
  label(g, c, "Opções", pad + 19, installY + 71, width - 38, "Escolha o aplicativo desktop ou instale com Docker.", 13, 550, { color: c.muted, height: 41 });
  action(g, c, { title: "Ver guia de instalação", x: pad + 19, y: installY + 118, width: 207, mode: "primary" });
  const footerY = Math.min(h - 60, installY + (phone ? 217 : 259));
  g.rect("Divisor do rodapé", pad, footerY - 12, width, 1, c.border);
  label(g, c, "Rodapé", pad, footerY, width, "GitHub    Documentação    Privacidade", 13, 640, { color: c.muted });
}

export function drawFinalMobilePreview(g, c) {
  label(g, c, "Marca do produto", 19, 13, 340, "Vukem Finances", 14, 720);
  mockModal(g, c, "expense", 11, 45, g.w - 22, true, () => undefined);
}
