# Vukem Finances final design direction

Open the [FigCraft document](direcao-final.figcraft) on **Revisão · Dashboard · Claro**. The [.fig copy](direcao-final.fig) is available for import. The prototype is a reference for implementation.

## Design decisions

- The final study pairs indigo actions with neutral surfaces in both themes. Inter headings, a compact command field and fine dividers reflect the visual hierarchy studied on [Linear](https://linear.app/).
- Monthly Margin supplies the budget bar with a marked limit and remaining amount. The monthly summary uses separate columns for balance, income and expenses. Transaction amounts align on the right, and row actions use quiet icons with 44 px targets.
- Ledger Line supplies icon navigation, action search, and compact controls for desktop and web.
- Ledger Trace informs tables with aligned amounts, explicit signs, and row actions.

## Review walkthrough

- **Dashboard.** Review the balance, the share of expenses by category, and the share of income by source. Legends show percentages in text. The budget shows spending, its limit, and the remaining amount.
- **Actions.** The web and desktop sidebar opens the new income, new expense, and import dialogs. The CSV or OFX preview distinguishes valid, duplicate, and invalid rows. The mobile design places actions under **Mais** and offers quick entry through **Adicionar**.
- **Transactions.** Review the description search and its empty state. The default page size is 25, with 50, 75, and 100 available. Frames show visible sample rows and the selected page range. Changing the query or page size returns to the first page.
- **Edit and delete.** Each row has icon buttons. Editing opens a populated form. Deletion asks for confirmation with the transaction name and amount. Give each button an accessible name and restore focus to the trigger when a dialog closes.
- **Themes.** The light and dark switch sits in the web and desktop sidebar and under **Mais** on mobile. Both modes have review frames.
- **Landing.** The landing uses product images rendered from the prototype with fixture data. **Landing · Sequência de animação** shows an expense arriving and the budget updating. The implementation should render an equivalent static composition when reduced motion is preferred. The mobile image is marked as a design preview.

Prototype links lead to scripted examples. The SaaS frames identify the active workspace. The document includes Personal and SaaS web, Personal desktop, and native mobile. The public landing follows the existing static site scope and destination links.

## Reproduce the final document

Run `node docs/ui-studies/gerar.mjs <temporary-directory> --final` and convert the resulting `direcao-final.json` with FigCraft CLI. To embed the product screenshots, export the **Visão geral · Dashboard · Pessoal** and **Imagem · Nova despesa** frames in both modes into an `images` subdirectory. Name the files `final-dashboard-light.png`, `final-dashboard-dark.png`, `final-phone-light.png`, and `final-phone-dark.png`. Generate the JSON again and convert it to `.figcraft` and `.fig`.
