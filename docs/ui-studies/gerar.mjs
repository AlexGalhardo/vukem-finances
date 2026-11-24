import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { existsSync } from "node:fs";
import { drawFinalDashboard, drawFinalLanding, drawFinalMobileNav, drawFinalMobilePreview, drawFinalNavIcon, drawFinalSidebarActions, finalScenarios } from "./direcao-final.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(join(here, "telas.txt"), "utf8").replace(/^\uFEFF/, "");
const screens = source.trim().split(/\r?\n/).filter(line => !line.startsWith("#"))
  .map(line => {
    const [id, edition, kind, title, action, body] = line.split("|");
    return { id, edition, kind, title, action, rows: body.split(";").map(row => row.split(/,(?!\d{2}(?:\s|$))/)) };
  });

const output = process.argv[2];
if (!output) throw new Error("Informe um diretório temporário para os arquivos JSON.");

const finalOnly = process.argv.includes("--final");
for (const direction of finalOnly ? [4] : [1, 2, 3]) {
  const final = direction === 4;
  const tokens = JSON.parse(readFileSync(join(here, `proposta-${final ? 2 : direction}.tokens.json`), "utf8").replace(/^\uFEFF/, ""));
  if (final) {
    tokens.name = "Vukem Finances · Direção final";
    Object.assign(tokens.light, { bg: "#FAFAFB", soft: "#F0F0F5", ink: "#202126", muted: "#575963", border: "#DDDDE3", control: "#878792" });
    Object.assign(tokens.dark, { bg: "#101113", surface: "#191A1D", soft: "#242529", ink: "#F1F1F3", muted: "#B1B2BB", border: "#34353B", control: "#747580" });
  }
  let serial = 0;
  const node = (type, name, extra = {}) => ({ id: `1:${++serial}`, name, type, ...extra });
  const box = (x, y, width, height) => ({ x, y, width, height });
  const paint = (hex) => {
    const rgb = hex.slice(1).match(/../g).map(pair => parseInt(pair, 16) / 255);
    return { type: "SOLID", color: { r: rgb[0], g: rgb[1], b: rgb[2] }, opacity: 1 };
  };
  const pages = [];
  const collection = "VariableCollectionId:1:1";
  const tokenIds = Object.fromEntries(["bg", "surface", "ink", "muted", "border", "accent", "onAccent", "soft", "positive", "negative", "warning"].map((key, i) => [key, `VariableID:1:${i + 2}`]));
  const modes = [{ modeId: "1:0", name: "Claro" }, { modeId: "1:1", name: "Escuro" }];
  const meta = {
    variableCollections: { [collection]: { id: collection, key: "palette", name: "Cores semânticas", defaultModeId: modes[0].modeId, modes, variableIds: Object.values(tokenIds) } },
    variables: Object.fromEntries(Object.entries(tokenIds).map(([key, id]) => [id, {
      id, key, name: `Cores/${key}`, variableCollectionId: collection, resolvedType: "COLOR", scopes: ["ALL_SCOPES"],
      valuesByMode: { "1:0": { ...paint(tokens.light[key]).color, a: 1 }, "1:1": { ...paint(tokens.dark[key]).color, a: 1 } },
    }])),
  };
  let activePalette;
  function fillFor(hex, preferred) {
    const key = preferred ?? Object.keys(tokenIds).find(name => activePalette[name] === hex);
    return key ? { ...paint(hex), boundVariables: { color: { type: "VARIABLE_ALIAS", id: tokenIds[key] } } } : paint(hex);
  }
  const page = (name) => {
    const value = node("CANVAS", name, { children: [] });
    pages.push(value);
    return value;
  };

  function artboard(target, title, w, h, color, draw, columns = 3, type = "FRAME") {
    const index = target.children.length;
    const x = (index % columns) * (Math.max(w, ...target.children.map(frame => frame.absoluteBoundingBox.width)) + 72);
    let y = 0;
    for (let i = 0; i < index - index % columns; i += columns) {
      y += Math.max(...target.children.slice(i, i + columns).map(frame => frame.absoluteBoundingBox.height)) + 80;
    }
    const children = [];
    const frame = node(type, title, { absoluteBoundingBox: box(x, y, w, h), fills: [fillFor(color)], clipsContent: true, explicitVariableModes: { [collection]: activePalette === tokens.dark ? "1:1" : "1:0" }, children });
    target.children.push(frame);
    const rect = (name, left, top, width, height, fill, radius = 0, stroke, opacity = 1) => {
      if (width <= 0 || height <= 0) throw new Error(`Invalid box: ${name}`);
      children.push(node("RECTANGLE", name, {
        absoluteBoundingBox: box(x + left, y + top, width, height),
        fills: [fillFor(fill)], cornerRadius: radius, opacity,
        ...(stroke ? { strokes: [fillFor(stroke)], strokeWeight: 1, strokeAlign: "INSIDE" } : {}),
      }));
    };
    const text = (name, left, top, width, value, size, fill, weight = 400, options = {}) => {
      children.push(node("TEXT", name, {
        absoluteBoundingBox: box(x + left, y + top, width, options.height ?? Math.ceil(size * 1.5)),
        characters: value, fills: [fillFor(fill, fill === activePalette.onAccent ? "onAccent" : undefined)],
        style: { fontFamily: options.family ?? "Inter", fontSize: size, fontWeight: weight,
          lineHeightPx: Math.ceil(size * 1.34), textAlignHorizontal: options.align ?? "LEFT",
          textAlignVertical: "TOP", textAutoResize: "NONE" },
      }));
    };
    const path = (name, left, top, width, height, data, fill) => {
      children.push(node("VECTOR", name, {
        absoluteBoundingBox: box(x + left, y + top, width, height),
        fills: [fillFor(fill)], fillGeometry: [{ path: data, windingRule: "NONZERO" }],
      }));
    };
    const photo = (name, left, top, width, height, imageRef) => {
      if (!existsSync(join(output, "images", `${imageRef}.png`))) return false;
      children.push(node("RECTANGLE", name, {
        absoluteBoundingBox: box(x + left, y + top, width, height),
        fills: [{ type: "IMAGE", imageRef, scaleMode: "FILL" }], cornerRadius: 12,
      }));
      return true;
    };
    const link = (name, left, top, width, height, destination) => {
      children.push(node("RECTANGLE", name, {
        absoluteBoundingBox: box(x + left, y + top, width, height), fills: [paint("#ffffff")],
        opacity: 0.001, destination,
      }));
    };
    draw({ rect, text, path, photo, link, w, h });
  }

  const navigation = ["Visão geral", "Transações", "Contas", "Cartões", "Categorias", "Orçamentos", "Metas", "Recorrências", "Relatórios", "Importar", "Configurações"];
  const style = [
    null,
    { radius: 5, rail: 202, inset: 34, family: "Inter", weight: 700, marker: "ledger" },
    { radius: 18, rail: 232, inset: 44, family: "Nunito Sans", weight: 800, marker: "guide" },
    { radius: 8, rail: 184, inset: 32, family: "Inter", weight: 700, marker: "command" },
    { radius: 8, rail: 222, inset: 36, family: "Inter", weight: 650 },
  ][direction];

  function app(target, entry, platform, edition, theme, c) {
    const phone = platform === "Mobile";
    const desktop = platform === "Desktop";
    const w = phone ? 390 : desktop ? 1280 : 1440;
    const modalState = final && entry.id.startsWith("Review/");
    const isFinalDashboard = final && (["Dashboard", "Transactions", "Mobile/Home", "Mobile/Transactions"].includes(entry.id) || modalState);
    const h = isFinalDashboard ? (phone ? 1150 : desktop ? 1110 : 1080) : phone ? 844 : desktop ? 850 : 900;
    const top = desktop ? 34 : 0;
    const rail = phone ? 0 : style.rail;
    const inset = phone ? 22 : style.inset;
    const signedOut = entry.kind === "auth";
    const admin = entry.id.startsWith("Admin/");
    const display = { family: style.family };
    artboard(target, `${entry.title} · ${entry.id} · ${edition} · ${theme}`, w, h, c.bg, (drawing) => {
      const { rect: r, text: t } = drawing;
      function button(label, x, y, width, mode = "primary") {
        const primary = mode === "primary";
        r(label, x, y, width, 44, primary ? c.accent : c.surface, style.radius, primary ? undefined : c.border);
        t("Rótulo", x + 8, y + 12, width - 16, label, 14, primary ? c.onAccent : c.ink, 650, { align: "CENTER" });
      }
      function rule(x, y, width) { r("Separador", x, y, width, 1, c.border); }
      if (desktop) {
        r("Barra da janela", 0, 0, w, top, c.surface);
        t("Menu nativo", 14, 8, 500, "Arquivo     Editar     Visualizar     Ajuda", 12, c.ink);
        t("Título da janela", w - 204, 8, 185, "Vukem Finances", 12, c.muted, 600, { align: "RIGHT" });
        rule(0, top - 1, w);
      }
      if (signedOut) {
        const cardWidth = phone ? 346 : 440;
        const x = phone ? 22 : direction === 3 ? (w - cardWidth) / 2 : w * .55;
        const y = top + (phone ? 102 : 160);
        if (!phone && direction !== 3) {
          r("Painel de entrada", 0, top, Math.round(w * .46), h - top, c.soft);
          t("Marca", 54, top + 76, 380, "Vukem Finances", 24, c.ink, style.weight, display);
          t("Contexto", 54, top + 240, 490, direction === 1 ? "Seu dinheiro, em ordem." : "Entenda seu mês.", 48, c.ink, style.weight, { ...display, height: 140 });
          t("Edição", 56, h - 100, 370, edition === "SaaS" ? "Espaço de trabalho" : "Edição pessoal", 15, c.muted);
        } else {
          t("Marca", x, y - 50, cardWidth, "Vukem Finances", 17, c.accent, 700);
        }
        t("Título", x, y, cardWidth, entry.title, phone ? 28 : 37, c.ink, style.weight, { ...display, height: 102 });
        let line = y + (entry.title.length > (phone ? 23 : 25) ? 106 : 72);
        for (const [field] of entry.rows) {
          if (field.endsWith(".")) {
            t("Instrução", x, line, cardWidth, field, 15, c.muted, 400, { height: 66 });
            line += 80;
          } else {
            t("Campo", x, line, cardWidth, field, 14, c.ink, 550);
            r("Entrada", x, line + 25, cardWidth, 46, c.surface, style.radius, c.border);
            line += 82;
          }
        }
        button(entry.action, x, line + 10, cardWidth);
        return;
      }

      if (phone) {
        t("Marca", 22, 30, 260, "Vukem Finances", 13, c.muted, 650);
      } else {
        r("Navegação", 0, top, rail, h - top, direction === 3 ? c.soft : c.surface);
        r("Limite do menu", rail - 1, top, 1, h - top, c.border);
        r("Símbolo", 18, top + 21, 28, 28, c.accent, style.radius);
        t("Marca", 24, top + 24, 20, "V", 16, c.onAccent, 800);
        t("Produto", 55, top + 25, rail - 57, admin ? "Administração" : "Finances", 18, c.ink, 700);
        t("Espaço", 19, top + 77, rail - 35, edition === "SaaS" ? "Finanças da casa  ⌄" : "Edição pessoal", 13, c.muted, 600);
        const items = admin ? ["Visão geral", "Usuários", "Espaços", "Receita administrativa"] : final ? navigation.filter(item => item !== "Importar") : navigation;
        items.forEach((item, i) => {
          const y = top + (final && !admin ? 253 : 118) + i * 37;
          const selected = item === entry.title || modalState && item === "Visão geral";
          if (selected) r("Item ativo", 12, y - 3, rail - 24, 34, c.soft, style.radius);
          if (final) drawFinalNavIcon(drawing, c, item, 17, y + 1, selected);
          else r("Marcador", 21, y + 9, 8, 8, selected ? c.accent : c.muted, direction === 2 ? 4 : 1);
          t("Rótulo de navegação", final ? 44 : 40, y + 3, rail - 50, item, 13, selected ? c.accent : c.ink, selected ? 700 : 450);
        });
        rule(16, h - 88, rail - 32);
        t("Rodapé", 19, h - 70, rail - 25, "Conta    Documentação", 12, c.muted, 550);
        if (final) {
          r("Busca de ações", rail + inset, top + 8, 236, 35, c.surface, 7, c.border);
          t("Busca", rail + inset + 12, top + 17, 140, "Buscar ações", 13, c.muted, 500);
          r("Tecla de atalho", rail + inset + 177, top + 14, 51, 23, c.soft, 4);
          t("Atalho", rail + inset + 177, top + 18, 51, "Ctrl K", 11, c.muted, 600, { align: "CENTER" });
        } else t("Busca", rail + inset, top + 20, 240, "Buscar ações  ·  Ctrl K", 13, c.muted, 550);
        t("Aparência", w - 220, top + 20, 180, final ? "Ajuda" : "Ajuda     Tema", 13, c.muted, 550, { align: "RIGHT" });
        rule(rail, top + 51, w - rail);
        if (final) drawFinalSidebarActions(drawing, c, { rail, top, h, theme, admin, edition });
      }

      if (isFinalDashboard || final && entry.id === "Mobile/More") {
        drawFinalDashboard(drawing, c, { entry, platform, edition, theme, x: rail + inset, top, inset });
        return;
      }

      const x = rail + inset;
      const wide = w - x - inset;
      const hy = phone ? 58 : top + 88;
      const titleHeight = phone && entry.title.length > 24 ? 76 : 51;
      t("Título", x, hy, phone ? wide : wide - 240, entry.title, phone ? 29 : 42, c.ink, style.weight, { ...display, height: titleHeight });
      const contextual = ["dashboard", "table", "progress"].includes(entry.kind) && !admin;
      const label = contextual ? (direction === 2 ? "Seu mês em andamento · Novembro de 2026" : "Novembro de 2026  ⌄") : "";
      let body = hy + titleHeight + 16;
      if (label) {
        t("Período", x, body - 10, wide, label, 13, c.muted, 550);
        body += 22;
      }
      const buttonWidth = Math.min(240, Math.max(130, entry.action.length * 8 + 32));
      if (!phone) button(entry.action, w - inset - buttonWidth, hy + 2, buttonWidth);
      const footer = phone ? h - 154 : h - 36;
      if (entry.kind === "dashboard") {
        const adminMetrics = ["28", "34", "12"];
        const amounts = entry.id === "Reports" || entry.id === "Mobile/Reports"
          ? ["+ R$ 2.248,06", "+ R$ 6.800,00", "− R$ 4.551,94"]
          : ["R$ 12.840,30", "+ R$ 6.800,00", "− R$ 4.551,94"];
        const metricGap = 14;
        const metricWidth = (wide - metricGap * 2) / 3;
        if (phone) {
          r("Resumo principal", x, body, wide, 140, c.soft, style.radius);
          t("Indicador", x + 15, body + 13, wide - 30, entry.rows[0][0], 14, c.muted, 650);
          t("Valor", x + 15, body + 39, wide - 30, amounts[0], 32, c.ink, 760, display);
          [1, 2].forEach(i => {
            t("Indicador", x + 15 + (i - 1) * 160, body + 94, 150, entry.rows[i][0], 11, c.muted, 600);
            t("Valor", x + 15 + (i - 1) * 160, body + 111, 150, amounts[i], 14, i === 1 ? c.positive : c.negative, 690);
          });
          body += 154;
        } else if (direction === 2) {
          const lead = Math.round(wide * .49);
          r("Resumo do mês", x, body, wide, 126, c.soft, style.radius);
          entry.rows.slice(0, 3).forEach(([name], i) => {
            const mx = x + 22 + (i ? lead + (i - 1) * (wide - lead) / 2 : 0);
            t("Indicador", mx, body + 23, 180, name, 14, c.muted, 650);
            t("Valor", mx, body + 45, i ? 210 : lead, (admin ? adminMetrics : amounts)[i], i ? 24 : 50, i === 2 && !admin ? c.negative : c.ink, 760, display);
          });
          body += 144;
        } else if (direction === 3) {
          r("Faixa de indicadores", x, body, wide, 88, c.surface, style.radius, c.border);
          entry.rows.slice(0, 3).forEach(([name], i) => {
            const mx = x + i * wide / 3;
            if (i) r("Divisor", mx, body + 14, 1, 60, c.border);
            t("Indicador", mx + 15, body + 13, wide / 3 - 30, name, 12, c.muted, 650);
            t("Valor", mx + 15, body + 35, wide / 3 - 30, (admin ? adminMetrics : amounts)[i], 28, i === 2 && !admin ? c.negative : c.ink, 760);
          });
          body += 105;
        } else {
          entry.rows.slice(0, 3).forEach(([name], i) => {
            const mx = x + i * (metricWidth + metricGap);
            r("Indicador", mx, body, metricWidth, 106, i === 0 ? c.soft : c.surface, style.radius, c.border);
            t("Rótulo", mx + 16, body + 15, metricWidth - 32, name, 14, c.muted, 600);
            t("Valor", mx + 16, body + 43, metricWidth - 32, (admin ? adminMetrics : amounts)[i], 33, i === 2 && !admin ? c.negative : c.ink, 750);
          });
          body += 124;
        }
        const chartWidth = phone || direction === 1 ? wide : Math.round(wide * .65);
        const chartHeight = phone ? 167 : 221;
        r("Gráfico", x, body, chartWidth, chartHeight, c.surface, style.radius, c.border);
        t("Título do gráfico", x + 15, body + 13, chartWidth - 30, entry.rows[3][0], 16, c.ink, 700);
        const step = (chartWidth - 50) / 8;
        [36, 63, 51, 73, 53, 90, 71, 98].forEach((bar, i) => {
          const baseline = body + chartHeight - 30;
          r("Entrada no período", x + 25 + i * step, baseline - bar, step * .39, bar, c.accent, 3);
          r("Saída no período", x + 25 + i * step + step * .42, baseline - bar * .55, step * .32, bar * .55, c.soft, 3, c.border);
        });
        ["01 nov", "15 nov", "30 nov"].forEach((label, i) => {
          t("Eixo", x + 18 + i * (chartWidth - 36) / 3, body + chartHeight - 23, (chartWidth - 36) / 3, label, 10, c.muted, 450,
            { align: ["LEFT", "CENTER", "RIGHT"][i] });
        });
        const right = chartWidth < wide;
        const ax = right ? x + chartWidth + 15 : x;
        const ay = right ? body : body + chartHeight + 13;
        const aw = right ? wide - chartWidth - 15 : wide;
        const ah = phone ? 119 : right ? chartHeight : 143;
        r("Atividade", ax, ay, aw, ah, c.surface, style.radius, c.border);
        t("Título da seção", ax + 14, ay + 12, aw - 28, entry.rows[4][0], 15, c.ink, 700);
        const activities = admin ? ["Marina Costa", "Rafael Lima"] : entry.title === "Relatórios" ? ["Moradia", "Alimentação"] : ["Supermercado", "Salário"];
        activities.forEach((name, i) => {
          const yy = ay + 43 + i * 28;
          t("Item", ax + 14, yy, aw - 28, name, 13, c.ink, 550);
          t("Valor", ax + 14, yy, aw - 28, admin ? "Verificado" : ["− R$ 186,42", "+ R$ 6.800,00"][i], 13, i ? c.positive : c.muted, 650, { align: "RIGHT" });
        });
        if (!phone && direction === 2 && entry.id === "Dashboard") {
          const guideY = body + chartHeight + 18;
          r("Orientação do mês", x, guideY, wide, 106, c.soft, style.radius);
          t("Título da orientação", x + 20, guideY + 17, wide - 42, "O que merece atenção neste mês?", 19, c.ink, 760, display);
          t("Contexto", x + 20, guideY + 52, wide - 250, "Você usou 92% do orçamento de Moradia. Revise o limite antes do próximo mês.", 14, c.muted);
          button("Ver orçamento", x + wide - 185, guideY + 32, 165, "secondary");
        }
        if (!phone && direction === 3 && entry.id === "Dashboard") {
          const commandY = body + chartHeight + 14;
          r("Ações rápidas", x, commandY, wide, 92, c.soft, style.radius);
          t("Título de ações", x + 15, commandY + 12, wide - 30, "Ações rápidas", 15, c.ink, 700);
          ["Novo lançamento  N", "Transações  G T", "Buscar ações  Ctrl K"].forEach((name, i) => {
            t("Atalho", x + 15 + i * wide / 3, commandY + 46, wide / 3 - 20, name, 14, c.accent, 650);
          });
        }
      } else if (entry.kind === "table") {
        const rowH = phone ? 67 : direction === 3 ? 48 : 59;
        r("Busca e filtros", x, body, wide, 44, c.surface, style.radius, c.border);
        t("Busca", x + 14, body + 12, wide - 28, "Buscar", 14, c.muted);
        t("Filtro", x + 14, body + 12, wide - 28, "Filtros", 14, c.accent, 700, { align: "RIGHT" });
        body += 58;
        r("Lista", x, body, wide, (entry.rows.length + (phone ? 0 : 1)) * rowH, c.surface, style.radius, c.border);
        if (!phone) {
          r("Cabeçalho da tabela", x + 1, body + 1, wide - 2, rowH - 1, c.soft, style.radius);
          t("Coluna", x + 15, body + 15, 250, "Nome", 13, c.muted, 700);
          t("Coluna", x + wide * .52, body + 15, 160, "Detalhes", 13, c.muted, 700);
          t("Coluna", x + 15, body + 15, wide - 30, "Valor / situação", 13, c.muted, 700, { align: "RIGHT" });
          body += rowH;
        }
        entry.rows.forEach(([name, detail, value], i) => {
          const yy = body + i * rowH;
          if (i) rule(x + 12, yy, wide - 24);
          t("Nome", x + 14, yy + 12, phone ? wide * .57 : wide * .48, name, 15, c.ink, 650);
          if (detail) t("Detalhe", phone ? x + 14 : x + wide * .52, yy + (phone ? 35 : 14), wide * .37, detail, 12, c.muted, 500);
          if (value) t("Valor ou situação", x + 14, yy + 13, wide - 28, value, 13, value.startsWith("+") ? c.positive : value.startsWith("−") ? c.negative : c.ink, 650, { align: "RIGHT" });
        });
      } else if (entry.kind === "progress") {
        const cardWidth = phone || direction === 1 ? wide : (wide - 20) / 2;
        entry.rows.forEach(([name, progress, value], i) => {
          const px = x + (cardWidth === wide ? 0 : (i % 2) * (cardWidth + 20));
          const yy = body + Math.floor(i / (cardWidth === wide ? 1 : 2)) * 107;
          r("Progresso", px, yy, cardWidth, 93, c.surface, style.radius, c.border);
          t("Categoria", px + 14, yy + 10, cardWidth - 70, name, 16, c.ink, 700);
          t("Percentual", px + 14, yy + 12, cardWidth - 28, `${progress}%`, 14, Number(progress) > 100 ? c.negative : c.muted, 650, { align: "RIGHT" });
          r("Trilho", px + 14, yy + 43, cardWidth - 28, 8, c.soft, 3);
          r("Consumo", px + 14, yy + 43, (cardWidth - 28) * Math.min(1, Number(progress) / 100), 8, Number(progress) > 100 ? c.negative : c.accent, 3);
          t("Valor", px + 14, yy + 60, cardWidth - 28, value, 13, c.muted, 550);
        });
      } else if (entry.kind === "form") {
        const fieldWidth = phone ? wide : Math.min(650, wide);
        entry.rows.forEach(([field], i) => {
          const yy = body + i * 76;
          t("Rótulo do campo", x, yy, fieldWidth, field, 14, c.ink, 600);
          r("Campo", x, yy + 24, fieldWidth, 43, c.surface, style.radius, c.border);
        });
      } else {
        const itemH = phone ? 93 : direction === 3 ? 63 : 78;
        entry.rows.forEach(([name, description], i) => {
          const yy = body + i * (itemH + 9);
          r("Linha de configuração", x, yy, wide, itemH, i === 0 && direction === 2 ? c.soft : c.surface, style.radius, c.border);
          t("Título da linha", x + 15, yy + 10, wide - 30, name, 16, c.ink, 700);
          if (description) t("Descrição", x + 15, yy + 35, wide - 30, description, 14, c.muted, 450, { height: 52 });
        });
      }
      if (phone) {
        button(entry.action, x, footer + 18, wide);
        if (final) {
          drawFinalMobileNav(drawing, c, edition, theme, entry.id);
          return;
        }
        r("Abas", 0, h - 77, w, 77, c.surface);
        rule(0, h - 77, w);
        ["Início", "Transações", "Adicionar", "Contas", "Mais"].forEach((item, i) => {
          const selected = ["Mobile/Home", "Mobile/Transactions", "Mobile/Add", "Mobile/Accounts", "Mobile/More"][i] === entry.id;
          r("Indicador", i * 78 + 33, h - 60, 12, 12, selected ? c.accent : c.soft, 3, selected ? undefined : c.muted);
          t("Aba", i * 78, h - 42, 78, item, 10, selected ? c.accent : c.muted, 650, { align: "CENTER" });
        });
      }
    }, phone ? 6 : 3);
  }


  function system(target, theme, c) {
    artboard(target, `Design system · ${theme}`, 1440, 1010, c.bg, ({ rect: r, text: t }) => {
      t("Marca", 54, 36, 1120, tokens.name, 17, c.accent, 700);
      t("Título", 54, 67, 1140, "Sistema de design", 56, c.ink, style.weight, { family: style.family });
      t("Cor", 54, 170, 500, "Cor e semântica", 24, c.ink, 700);
      const swatches = ["bg", "surface", "ink", "muted", "border", "accent", "onAccent", "soft", "positive", "negative", "warning"];
      swatches.forEach((key, i) => {
        const x = 54 + (i % 6) * 214;
        const y = 218 + Math.floor(i / 6) * 106;
        r(key, x, y, 195, 59, c[key], style.radius, c.border);
        t("Token", x, y + 67, 195, `${key}   ${c[key]}`, 14, c.ink, 600);
      });
      t("Tipografia", 54, 462, 490, "Tipografia e valores", 24, c.ink, 700);
      t("Título", 54, 506, 510, "Visão geral", 46, c.ink, style.weight, { family: style.family });
      t("Corpo", 54, 581, 480, "Acompanhe entradas e saídas no período.", 18, c.muted);
      t("Valor de entrada", 54, 625, 230, "+ R$ 6.800,00", 25, c.positive, 720);
      t("Valor de saída", 286, 625, 230, "− R$ 186,42", 25, c.negative, 720);
      t("Controles", 650, 462, 520, "Controles e estados", 24, c.ink, 700);
      [["Salvar", c.accent, c.onAccent], ["Cancelar", c.surface, c.ink], ["Excluir", c.negative, c.bg], ["Indisponível", c.soft, c.muted]].forEach(([label, fill, ink], i) => {
        const x = 650 + i * 177;
        r(label, x, 512, 156, 44, fill, style.radius, c.border);
        t("Rótulo", x, 523, 156, label, 14, ink, 680, { align: "CENTER" });
      });
      r("Focus", 648, 618, 334, 48, c.accent, style.radius);
      r("Input", 650, 620, 330, 44, c.surface, style.radius);
      t("Rótulo", 650, 589, 330, "Busca com foco", 14, c.ink, 650);
      t("Valor", 663, 630, 301, "Supermercado", 14, c.ink);
      r("Alerta", 1010, 610, 322, 63, c.soft, style.radius);
      t("Atenção", 1024, 624, 296, "Atenção · limite quase atingido", 14, c.warning, 700);
      t("Padrões", 54, 743, 600, "Padrões de interação", 24, c.ink, 700);
      const patterns = ["Alvos de toque de pelo menos 44 px", "Foco visível em todo controle", "Sinal e rótulo para valores e status", "Tabela com leitura vertical no celular", "Ajuda junto à decisão e erros junto ao campo", "Respeitar tema do sistema e movimento reduzido"];
      patterns.forEach((label, i) => {
        const x = 54 + (i % 2) * 596;
        const y = 790 + Math.floor(i / 2) * 51;
        r("Divisor", x, y + 36, 550, 1, c.border);
        t("Diretriz", x, y, 550, label, 15, c.ink, 550);
      });
    }, 2);
  }

  function library(target, c) {
    const examples = [
      ["Botão / Primário", "Salvar lançamento", "Ação principal", c.accent, c.onAccent],
      ["Botão / Secundário", "Cancelar", "Ação secundária", c.surface, c.ink],
      ["Botão / Perigo", "Excluir transação", "Confirmação necessária", c.negative, c.bg],
      ["Campo / Texto", "Descrição", "Ex.: supermercado", c.surface, c.ink],
      ["Campo / Erro", "Valor", "Informe um valor maior que zero", c.surface, c.negative],
      ["Feedback / Sucesso", "Lançamento salvo", "A alteração já aparece na lista", c.surface, c.positive],
      ["Feedback / Atenção", "Orçamento quase no limite", "Restam R$ 56,00 para gastar", c.soft, c.warning],
      ["Linha / Transação", "Supermercado", "Alimentação · − R$ 186,42", c.surface, c.ink],
      ["Ajuda / Atalho", "Buscar ações", "Ctrl + K", c.surface, c.ink],
      ["Estado / Vazio", "Nenhuma transação", "Adicione um lançamento para começar", c.surface, c.ink],
      ["Estado / Carregando", "Carregando transações", "Aguarde", c.surface, c.muted],
      ["Navegação / Selecionada", "Transações", "Seção atual", c.soft, c.accent],
    ];
    for (const [name, label, helper, fill, ink] of examples) {
      artboard(target, name, 334, 104, c.bg, ({ rect: r, text: t }) => {
        r("Superfície", 10, 9, 314, 86, fill, style.radius, c.border);
        t("Rótulo", 24, 25, 286, label, 16, ink, 690, { family: style.family });
        t("Detalhe", 24, 56, 286, helper, 13, ink, 500);
      }, 4, "COMPONENT");
    }
  }
  function landing(target, theme, c, phone) {
    const w = phone ? 390 : 1440;
    const h = phone ? 2480 : 1950;
    const margin = phone ? 22 : 84;
    const wide = w - margin * 2;
    const family = style.family;
    artboard(target, `Landing pública · ${phone ? "Mobile web" : "Web"} · ${theme}`, w, h, c.bg, ({ rect: r, text: t }) => {
      const centered = direction === 2 && !phone;
      const align = centered ? "CENTER" : "LEFT";
      t("Marca", margin, 24, 260, "Vukem Finances", 19, c.ink, 760);
      t("Navegação", phone ? 243 : 800, 27, phone ? 126 : 550, phone ? "PT-BR  ·  Menu" : "Recursos       Edições       Instalação       Documentação       PT-BR", 13, c.muted, 650, { align: "RIGHT" });
      r("Separador", margin, 64, wide, 1, c.border);
      const headlines = ["Finanças pessoais,\ncom os dados sob seu controle.", "Entenda seu mês.\nPlaneje o próximo.", "Organize seu dinheiro\nno seu computador."];
      const heroY = phone ? 98 : 121;
      const heroWidth = phone || centered ? wide : 725;
      t("Título", margin, heroY, heroWidth, headlines[direction - 1], phone ? 39 : 66, c.ink, style.weight, { family, align, height: phone ? 178 : 192 });
      const copyY = heroY + (phone ? 180 : 190);
      t("Descrição", margin, copyY, phone || centered ? wide : 600, "Acompanhe contas, transações e orçamentos em um aplicativo de código aberto que você instala e executa.", phone ? 16 : 19, c.muted, 450, { align, height: 94 });
      const actionY = copyY + (phone ? 96 : 72);
      const bx = centered ? (w - 240) / 2 : margin;
      r("CTA", bx, actionY, 240, 48, c.accent, style.radius);
      t("Instalação", bx, actionY + 13, 240, "Ver opções de instalação", 14, c.onAccent, 700, { align: "CENTER" });
      const previewY = actionY + 76;
      const chartW = phone ? wide : direction === 1 ? 760 : wide;
      const chartX = centered ? (w - chartW) / 2 : margin;
      const chartH = phone ? 266 : 275;
      r("Prévia desenhada do produto", chartX, previewY, chartW, chartH, c.surface, style.radius, c.border);
      t("Prévia", chartX + 18, previewY + 14, chartW - 36, "Resumo de novembro", 16, c.muted, 650);
      t("Saldo de demonstração", chartX + 18, previewY + 44, chartW - 36, "R$ 12.840,30", phone ? 34 : 42, c.ink, 750, { family });
      [37, 58, 39, 80, 55, 88, 73, 93].forEach((value, i) => {
        const dx = (chartW - 44) / 8;
        r("Atividade", chartX + 22 + i * dx, previewY + 210 - value, dx * .68, value, i % 3 ? c.accent : c.soft, 4);
      });
      t("Nota", chartX + 18, previewY + 233, chartW - 36, "Exemplo com dados fictícios", 12, c.muted);
      if (!phone && direction === 1) {
        t("Descrição lateral", margin + 815, previewY + 36, 375, "Veja o que entrou, o que saiu e quanto sobrou no mês.", 29, c.ink, 700, { family, height: 130 });
      }
      let y = previewY + chartH + 74;
      const blocks = [
        ["Como funciona", ["Instale no computador ou servidor", "Organize as contas e transações", "Acompanhe relatórios e orçamentos"]],
        ["Escolha sua edição", ["Pessoal · Um espaço, com uso offline", "SaaS autohospedado · Mais espaços e planos Free e Pro"]],
        ["Privacidade e controle", ["Dados no computador ou servidor que você escolher", "Importação CSV e OFX e exportação para CSV e JSON"]],
        ["Instalação", ["Aplicativo desktop", "Instalação com Docker"]],
        ["Perguntas frequentes", ["Existe um serviço hospedado?", "Onde os dados ficam?"]],
      ];
      for (const [title, items] of blocks) {
        t("Seção", margin, y, wide, title, phone ? 27 : 35, c.ink, style.weight, { family });
        const itemW = phone ? wide : (wide - 22 * (items.length - 1)) / items.length;
        items.forEach((item, i) => {
          const x = margin + (phone ? 0 : i * (itemW + 22));
          const top = y + 54 + (phone ? i * 66 : 0);
          r("Recurso", x, top, itemW, phone ? 56 : 70, i === 0 && direction === 2 ? c.soft : c.surface, style.radius, c.border);
          t("Texto do recurso", x + 12, top + 12, itemW - 24, item, phone ? 14 : 16, c.ink, 600, { height: 46 });
        });
        y += phone ? 74 + items.length * 66 : 176;
      }
      r("Separador", margin, y, wide, 1, c.border);
      t("Rodapé", margin, y + 14, wide, phone ? "GitHub  ·  Documentação  ·  Privacidade" : "GitHub       Documentação       Privacidade       Português  /  English  /  Español", 14, c.muted, 650);
    }, 2);
  }

  for (const [theme, c] of [["Claro", tokens.light], ["Escuro", tokens.dark]]) {
    activePalette = c;
    if (final) {
      const overview = page(`Revisão · Dashboard · ${theme}`);
      const home = screens.find(screen => screen.id === "Dashboard");
      app(overview, home, "Web", "Pessoal", theme, c);
      app(overview, home, "Web", "SaaS", theme, c);
      app(overview, home, "Desktop", "Pessoal desktop", theme, c);
      app(overview, screens.find(screen => screen.id === "Mobile/Home"), "Mobile", "Local", theme, c);
      overview.flowStartingPoints = overview.children.map((frame, i) => ({ nodeId: frame.id, name: ["Web Pessoal", "Web SaaS", "Desktop Pessoal", "Mobile"][i] }));
      for (const [platform, edition] of [["Web", "Pessoal"], ["Web", "SaaS"], ["Desktop", "Pessoal desktop"], ["Mobile", "Local"]]) {
        const flows = page(`Revisão · ${platform} ${edition} · Fluxos · ${theme}`);
        finalScenarios.forEach(scenario => app(flows, scenario, platform, edition, theme, c));
      }
    }
    system(page(`Sistema de design · ${theme}`), theme, c);
    library(page(`Componentes · ${theme}`), c);
    const publicSite = page(`Landing pública · ${theme}`);
    if (final) {
      for (const phone of [false, true]) {
        artboard(publicSite, `Landing final · ${phone ? "Celular" : "Web"} · ${theme}`,
          phone ? 390 : 1440, phone ? 2170 : 1800, c.bg, drawing => drawFinalLanding(drawing, c, theme, phone), 2);
      }
      const motion = page(`Landing · Sequência de animação · ${theme}`);
      for (let step = 0; step < 3; step++) {
        artboard(motion, `Cena ${step + 1} · ${theme}`, 720, 420, c.bg, drawing => drawFinalLanding(drawing, c, theme, false, step));
      }
      motion.flowStartingPoints = [{ nodeId: motion.children[0].id, name: "Orçamento em movimento" }];
      artboard(page(`Landing · Imagem do formulário · ${theme}`), `Imagem · Nova despesa · ${theme}`, 390, 570, c.bg,
        drawing => drawFinalMobilePreview(drawing, c));
    } else {
      landing(publicSite, theme, c, false);
      landing(publicSite, theme, c, true);
    }
    for (const [platform, edition, allow] of [
      ["Web", "Pessoal", ["ambas", "pessoal"]],
      ["Web", "SaaS", ["ambas", "saas"]],
      ["Desktop", final ? "Pessoal desktop" : "Pessoal", ["ambas", "pessoal"]],
      ["Mobile", "Local", ["local"]],
    ]) {
      const entries = screens.filter(screen => allow.includes(screen.edition)
        && (!screen.id.startsWith("Desktop/") || platform === "Desktop"));
      const groups = platform === "Mobile"
        ? [["Telas", entries]]
        : [["Finanças", entries.filter(s => !/^(Admin|Auth|State|Desktop)\//.test(s.id) && s.kind !== "auth")],
          ["Conta e acesso", entries.filter(s => /^(Auth|Desktop)\//.test(s.id) || s.kind === "auth" && !s.id.startsWith("State/"))],
          ["Estados", entries.filter(s => s.id.startsWith("State/"))],
          ["Administração", entries.filter(s => s.id.startsWith("Admin/"))]];
      for (const [group, items] of groups) {
        if (!items.length) continue;
        const target = page(`${platform} · ${edition} · ${group} · ${theme}`);
        items.forEach(entry => app(target, entry, platform, edition, theme, c));
      }
    }
  }
  if (final) {
    const frames = new Map();
    for (const target of pages) {
      for (const frame of target.children) {
        if (!frames.has(frame.name)) frames.set(frame.name, frame.id);
      }
    }
    for (const target of pages) {
      for (const frame of target.children) {
        for (const child of frame.children) {
          if (!child.destination) continue;
          const destinationId = frames.get(child.destination);
          if (destinationId) {
            const transition = child.destination.startsWith("Cena ")
              ? { type: "SMART_ANIMATE", duration: child.destination.startsWith("Cena 2") ? 0.16 : 0.22, easing: { type: "EASE_IN_AND_OUT" } }
              : undefined;
            child.interactions = [
              { trigger: { type: "ON_CLICK" }, actions: [{ type: "NODE", navigation: "NAVIGATE", destinationId, transition }] },
            ];
          }
          delete child.destination;
        }
      }
    }
  }
  const document = node("DOCUMENT", tokens.name, { children: pages });
  writeFileSync(join(output, final ? "direcao-final.json" : `proposta-${direction}.json`), JSON.stringify({ name: tokens.name, document, components: {}, styles: {}, meta }));
  console.log(`${tokens.name}: ${pages.length} páginas, ${pages.reduce((count, page) => count + page.children.length, 0)} frames`);
}
