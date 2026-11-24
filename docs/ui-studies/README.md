# Vukem Finances interface studies

The [final direction](direcao-final.md) combines the independent studies below. Open the [FigCraft document](direcao-final.figcraft) or import the [Figma file](direcao-final.fig) to review it before implementation.

| Proposal | Direction | FigCraft | Figma | Specification |
| --- | --- | --- | --- | --- |
| Ledger Trace | Review balances, transactions, and decisions with a linear reading order | [Document](proposta-1.figcraft) | [Figma file](proposta-1.fig) | [Research and design system](proposta-1.md) · [Tokens](proposta-1.tokens.json) |
| Monthly Margin | Understand the month and plan with guidance in context | [Document](proposta-2.figcraft) | [Figma file](proposta-2.fig) | [Research and design system](proposta-2.md) · [Tokens](proposta-2.tokens.json) |
| Ledger Line | Find actions quickly and work with keyboard shortcuts | [Document](proposta-3.figcraft) | [Figma file](proposta-3.fig) | [Research and design system](proposta-3.md) · [Tokens](proposta-3.tokens.json) |

Each original study has 284 screen and foundation frames, 24 reference components, and 11 color variables with light and dark modes. The pages group Personal and SaaS web, Personal desktop, native mobile, and the public landing at wide and narrow widths. The examples use Brazilian Portuguese, BRL, and fixture data.

The specifications cover additional states. The screens use editable layers. The Figma files were exported by FigCraft and can be imported separately.

## Reproduce the original studies

The [inventory](telas.txt) lists the edition, visual pattern, and sample content for each screen. The [generator](gerar.mjs) combines the inventory with each proposal's tokens. With Node and FigCraft CLI installed, generate intermediate JSON in a temporary directory.

```powershell
$studyTemp = Join-Path $env:TEMP 'vukem-ui-studies'
New-Item -ItemType Directory -Force $studyTemp | Out-Null
node docs/ui-studies/gerar.mjs $studyTemp
$figcraft = 'C:/Program Files/ArtCraft/FigCraft/figcraft-cli.exe'
foreach ($n in 1..3) {
  & $figcraft convert (Join-Path $studyTemp "proposta-$n.json") "docs/ui-studies/proposta-$n.figcraft"
  & $figcraft convert "docs/ui-studies/proposta-$n.figcraft" "docs/ui-studies/proposta-$n.fig"
}
```
