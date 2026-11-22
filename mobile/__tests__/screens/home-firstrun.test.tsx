import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { createInstallments } from "../../src/domain/recurrence";
import { FirstRunScreen } from "../../src/screens/FirstRunScreen";
import { HomeScreen } from "../../src/screens/HomeScreen";
import { renderApp } from "../helpers/app";
import { ACCOUNT, openEnv, txFields } from "../helpers/fixtures";

describe("FirstRun", () => {
  it("makes the first account, the default categories and the settings, in the chosen language", async () => {
    const onFinished = jest.fn();
    const env = await openEnv();
    await renderApp(<FirstRunScreen onFinished={onFinished} />, { env, settings: { onboarded: false } });

    await fireEvent.press(screen.getByRole("radio", { name: "Português (Brasil)" }));
    expect(screen.getByRole("header", { name: "Boas-vindas" })).toBeTruthy();
    await fireEvent.press(screen.getByRole("radio", { name: "BRL" }));
    await fireEvent.changeText(screen.getByLabelText("Nome da primeira conta"), "Nubank");
    await fireEvent.changeText(screen.getByLabelText("Saldo atual"), "1.234,56");
    await fireEvent.press(screen.getByRole("button", { name: "Começar" }));

    await waitFor(() => expect(onFinished).toHaveBeenCalled());
    const accounts = await env.repo.list("account");
    expect(accounts).toHaveLength(1);
    expect(accounts[0]).toMatchObject({ name: "Nubank", opening_cents: 123456 });
    const names = (await env.repo.list("category")).map((c) => c.name);
    expect(names).toEqual(expect.arrayContaining(["Mercado", "Salário"]));
    const meta = await env.db.all<{ key: string; value: string }>(
      "SELECT key, value FROM meta WHERE key != 'schema_version' ORDER BY key",
    );
    expect(meta).toEqual([
      { key: "currency", value: "BRL" },
      { key: "locale", value: "pt-BR" },
      { key: "onboarded", value: "1" },
      { key: "timezone", value: "UTC" },
    ]);
  });

  it("needs an account name and a readable balance", async () => {
    const env = await openEnv();
    await renderApp(<FirstRunScreen />, { env, settings: { onboarded: false } });
    await fireEvent.press(screen.getByRole("button", { name: "Start" }));
    expect(await screen.findByText("Required")).toBeTruthy();

    await fireEvent.changeText(screen.getByLabelText("First account name"), "Cash");
    await fireEvent.changeText(screen.getByLabelText("Current balance"), "1.005");
    await fireEvent.press(screen.getByRole("button", { name: "Start" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/Enter an amount/);
    expect(await env.repo.list("account")).toHaveLength(0);
  });

  it("can skip the default categories", async () => {
    const env = await openEnv();
    await renderApp(<FirstRunScreen />, { env, settings: { onboarded: false } });
    await fireEvent.changeText(screen.getByLabelText("First account name"), "Cash");
    await fireEvent.press(screen.getByRole("checkbox", { name: "Start with the default categories" }));
    await fireEvent.press(screen.getByRole("button", { name: "Start" }));
    await waitFor(async () => expect(await env.repo.list("account")).toHaveLength(1));
    expect(await env.repo.list("category")).toHaveLength(0);
  });
});

describe("Home", () => {
  it("shows net worth, this month's totals and what is due in 7 days", async () => {
    const env = await openEnv();
    const acc = (await env.repo.create("account", { ...ACCOUNT, opening_cents: 100000 })).uid;
    await env.repo.create(
      "transaction",
      txFields(acc, { kind: "income", amount_cents: 250000, occurred_on: "2027-03-05" }),
    );
    await env.repo.create("transaction", txFields(acc, { amount_cents: 1250, occurred_on: "2027-03-06" }));
    await env.repo.create("transaction", txFields(acc, { amount_cents: 9999, occurred_on: "2027-02-06" }));
    await createInstallments(env.db, env.repo, {
      account_uid: acc,
      category_uid: null,
      total_cents: 3000,
      parts: 3,
      first_date: "2027-03-18",
      description: "TV",
    });
    await renderApp(<HomeScreen />, { env });

    // 100000 + 250000 - 1250 - 9999
    expect(await screen.findByLabelText("3387 dollars 51 cents")).toHaveTextContent("$3,387.51");
    expect(screen.getByLabelText("Income 2500 dollars")).toHaveTextContent("+$2,500.00");
    expect(screen.getByLabelText("Expense minus 22 dollars 50 cents")).toHaveTextContent("-$22.50"); // 12.50 plus the TV part dated 2027-03-18, which the month reaches
    expect(screen.getByText("TV")).toBeTruthy();
    expect(screen.getByText("2027-03-18")).toBeTruthy();
  });

  it("invites the first transaction when there is none", async () => {
    const env = await openEnv();
    await env.repo.create("account", ACCOUNT);
    const onAdd = jest.fn();
    await renderApp(<HomeScreen onAdd={onAdd} />, { env });
    await fireEvent.press(await screen.findByRole("button", { name: "Add" }));
    expect(onAdd).toHaveBeenCalled();
    expect(screen.getByText("Nothing due in the next 7 days.")).toBeTruthy();
  });
});
