import { readdirSync, readFileSync, statSync } from "node:fs";
import { act, fireEvent, render, screen } from "@testing-library/react-native";
import { type ReactElement, type ReactNode, useMemo } from "react";
import { PixelRatio, Pressable, Text, View } from "react-native";
import { Button } from "../../src/components/Button";
import { Chip } from "../../src/components/Chip";
import { EmptyState } from "../../src/components/EmptyState";
import { ListRow } from "../../src/components/ListRow";
import { MoneyText } from "../../src/components/MoneyText";
import { Toast } from "../../src/components/Toast";
import { useApp } from "../../src/core/state";
import { createCard } from "../../src/domain/cards";
import { newUid } from "../../src/domain/ids";
import { createInstallments } from "../../src/domain/recurrence";
import { statementRefIn } from "../../src/domain/statements";
import { LockGate } from "../../src/lock/LockGate";
import { AccountsScreen } from "../../src/screens/AccountsScreen";
import { AddScreen } from "../../src/screens/AddScreen";
import { BudgetsScreen } from "../../src/screens/BudgetsScreen";
import { CardsScreen } from "../../src/screens/CardsScreen";
import { CategoriesScreen } from "../../src/screens/CategoriesScreen";
import { FirstRunScreen } from "../../src/screens/FirstRunScreen";
import { HomeScreen } from "../../src/screens/HomeScreen";
import { MoreScreen } from "../../src/screens/MoreScreen";
import { RecurrencesScreen } from "../../src/screens/RecurrencesScreen";
import { ReportsScreen } from "../../src/screens/ReportsScreen";
import { SettingsScreen } from "../../src/screens/SettingsScreen";
import { SyncConflictsScreen } from "../../src/screens/SyncConflictsScreen";
import { SyncConnectScreen } from "../../src/screens/SyncConnectScreen";
import { SyncMergeScreen } from "../../src/screens/SyncMergeScreen";
import { SyncProblemsScreen } from "../../src/screens/SyncProblemsScreen";
import { SyncStatusScreen } from "../../src/screens/SyncStatusScreen";
import { TagsScreen } from "../../src/screens/TagsScreen";
import { TransactionsScreen } from "../../src/screens/TransactionsScreen";
import { SyncProvider } from "../../src/sync/context";
import { SyncController } from "../../src/sync/controller";
import { a11yViolations, hasHeading, layoutViolations, touchableCount } from "../helpers/a11y";
import { renderApp } from "../helpers/app";
import { ACCOUNT, type Env, openEnv, txFields } from "../helpers/fixtures";
import { FakeTransport } from "../sync/fake-transport";

async function settle(): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 60));
  });
}

async function press(name: string | RegExp, role: "button" | "radio" | "checkbox" = "button"): Promise<void> {
  await fireEvent.press(screen.getByRole(role, { name }));
  await settle();
}

/** A database with a little of everything, so every screen has rows, chips and buttons to audit. */
async function seed(): Promise<Env> {
  const env = await openEnv();
  const main = (await env.repo.create("account", { ...ACCOUNT, name: "Main", opening_cents: 250000 })).uid;
  const savings = (await env.repo.create("account", { ...ACCOUNT, name: "Savings", kind: "savings", position: 1 })).uid;
  const food = await env.ledger.createCategory({
    name: "Food",
    kind: "expense",
    parent_uid: null,
    color: "green",
    icon: "",
  });
  await env.ledger.createCategory({ name: "Snacks", kind: "expense", parent_uid: food.uid, color: "", icon: "" });
  await env.ledger.createCategory({ name: "Salary", kind: "income", parent_uid: null, color: "", icon: "" });
  const tag = await env.repo.create("tag", { name: "work", color: "" });
  await env.repo.create(
    "transaction",
    txFields(main, {
      category_uid: food.uid,
      amount_cents: 1250,
      description: "lunch",
      occurred_on: "2027-03-10",
      tag_uids: [tag.uid],
    }),
  );
  await env.repo.create(
    "transaction",
    txFields(main, { kind: "income", amount_cents: 300000, description: "pay", occurred_on: "2027-03-01" }),
  );
  await env.ledger.createTransfer({
    from_account_uid: main,
    to_account_uid: savings,
    amount_cents: 5000,
    date: "2027-03-11",
    description: "save",
  });
  await env.repo.create(
    "budget",
    { category_uid: food.uid, limit_cents: 40000, rollover: true },
    newUid(Date.UTC(2027, 0, 1)),
  );
  await env.repo.create("recurrence", {
    account_uid: main,
    category_uid: null,
    kind: "expense",
    amount_cents: 999,
    description: "gym",
    frequency: "weekly",
    interval: 1,
    start_date: "2027-03-16",
    end_date: null,
    max_count: null,
    generated_through: null,
    active: true,
  });
  await createInstallments(env.db, env.repo, {
    account_uid: main,
    category_uid: null,
    total_cents: 3000,
    parts: 3,
    first_date: "2027-03-18",
    description: "TV",
  });
  const card = await createCard(env.db, env.repo, {
    name: "Visa",
    limit_cents: 500000,
    closing_day: 10,
    due_day: 20,
    payment_account_uid: main,
  });
  const ref = await statementRefIn(env.db, card.account_uid, "2027-02-20");
  await env.repo.create(
    "transaction",
    txFields(card.account_uid, { amount_cents: 30000, occurred_on: "2027-02-20", statement_ref: ref }),
  );
  // What the sync screens list: a conflict and a problem to review.
  await env.db.run(
    "INSERT INTO sync_conflicts (entity, uid, field, kind, lost_value, applied_value, created_at) VALUES ('tag', ?, 'name', 'field', '\"old\"', '\"work\"', 'x')",
    [tag.uid],
  );
  await env.db.run(
    "INSERT INTO sync_problems (entity, uid, op_id, op, code, message, created_at) VALUES ('account', ?, 'op1', 'upsert', 'forbidden', 'Your role cannot change that.', 'x')",
    [main],
  );
  return env;
}

/** The sync screens over the app's own database and a transport that is never called. */
function WithSync(props: { children: ReactNode }): ReactNode {
  const app = useApp();
  const controller = useMemo(
    () =>
      new SyncController({
        db: app.db,
        repo: app.repo,
        tokens: { get: async () => null, set: async () => undefined, clear: async () => undefined },
        makeTransport: () => new FakeTransport(),
        clock: { now: () => 0, setTimeout: () => 0, clearTimeout: () => undefined },
        random: { next: () => 0.5 },
        defaultDeviceName: "Test phone",
        platform: "android",
      }),
    [app.db, app.repo],
  );
  return <SyncProvider controller={controller}>{props.children}</SyncProvider>;
}

const services = {
  lockStore: { isEnabled: async () => false, setEnabled: async () => undefined },
  lockPlatform: { available: async () => true, authenticate: async () => true },
  sharer: { shareText: async () => undefined },
  version: "0.1.0",
};

interface Case {
  name: string;
  element: ReactElement;
  steps?: () => Promise<void>;
  onboarding?: boolean;
}

const cases: Case[] = [
  { name: "first run", element: <FirstRunScreen />, onboarding: true },
  { name: "home", element: <HomeScreen onAdd={() => undefined} /> },
  { name: "add (expense)", element: <AddScreen /> },
  { name: "add (transfer)", element: <AddScreen />, steps: () => press("Transfer", "radio") },
  { name: "add (error)", element: <AddScreen />, steps: () => press("Save") },
  { name: "transactions", element: <TransactionsScreen /> },
  { name: "transactions (row actions)", element: <TransactionsScreen />, steps: () => press(/^lunch, /) },
  { name: "accounts", element: <AccountsScreen /> },
  { name: "accounts (edit)", element: <AccountsScreen />, steps: () => press(/^Main, /) },
  {
    name: "accounts (delete confirmation)",
    element: <AccountsScreen />,
    steps: async () => {
      await press(/^Main, /);
      await press("Delete");
    },
  },
  { name: "cards", element: <CardsScreen /> },
  { name: "cards (statements)", element: <CardsScreen />, steps: () => press(/^Visa, /) },
  {
    name: "cards (pay)",
    element: <CardsScreen />,
    steps: async () => {
      await press(/^Visa, /);
      await press(/^Pay statement /);
    },
  },
  { name: "categories", element: <CategoriesScreen /> },
  { name: "categories (edit)", element: <CategoriesScreen />, steps: () => press("Food") },
  { name: "tags", element: <TagsScreen /> },
  { name: "budgets", element: <BudgetsScreen /> },
  { name: "budgets (edit)", element: <BudgetsScreen />, steps: () => press(/^Food, /) },
  { name: "recurrences", element: <RecurrencesScreen /> },
  { name: "recurrences (rule form)", element: <RecurrencesScreen />, steps: () => press(/^gym, /) },
  {
    name: "recurrences (installment form)",
    element: <RecurrencesScreen />,
    steps: () => press("Installment purchase"),
  },
  { name: "reports (chart)", element: <ReportsScreen /> },
  { name: "reports (table)", element: <ReportsScreen />, steps: () => press("Table", "radio") },
  { name: "reports (by category)", element: <ReportsScreen />, steps: () => press("By category", "radio") },
  {
    name: "sync status",
    element: (
      <WithSync>
        <SyncStatusScreen
          onConnect={() => undefined}
          onChoose={() => undefined}
          onConflicts={() => undefined}
          onProblems={() => undefined}
        />
      </WithSync>
    ),
  },
  {
    name: "sync connect",
    element: (
      <WithSync>
        <SyncConnectScreen onConnected={() => undefined} />
      </WithSync>
    ),
  },
  {
    name: "sync connect (plain http confirmation)",
    element: (
      <WithSync>
        <SyncConnectScreen onConnected={() => undefined} />
      </WithSync>
    ),
    steps: async () => {
      await fireEvent.changeText(screen.getByLabelText("Server address"), "http://192.168.1.20:3000");
      await fireEvent.changeText(screen.getByLabelText("Pairing code"), "ABCD-EFGH");
      await press("Connect");
    },
  },
  {
    name: "sync merge choice",
    element: (
      <WithSync>
        <SyncMergeScreen sharer={services.sharer} onDone={() => undefined} />
      </WithSync>
    ),
  },
  {
    name: "sync merge choice (replace confirmation)",
    element: (
      <WithSync>
        <SyncMergeScreen sharer={services.sharer} onDone={() => undefined} />
      </WithSync>
    ),
    steps: () => press("Replace this phone's data with the server's"),
  },
  {
    name: "sync conflicts",
    element: (
      <WithSync>
        <SyncConflictsScreen />
      </WithSync>
    ),
  },
  {
    name: "sync problems",
    element: (
      <WithSync>
        <SyncProblemsScreen />
      </WithSync>
    ),
  },
  { name: "more", element: <MoreScreen onOpen={() => undefined} /> },
  { name: "settings", element: <SettingsScreen services={services} /> },
  {
    name: "lock screen",
    element: (
      <LockGate
        store={{ isEnabled: async () => true, setEnabled: async () => undefined }}
        platform={{ available: async () => true, authenticate: async () => false }}
      >
        <Text>hidden</Text>
      </LockGate>
    ),
  },
];

describe("accessibility of every screen", () => {
  it.each(cases.map((c) => [c.name, c] as const))(
    "%s: roles, labels, 48 dp targets and no clipped text",
    async (_name, c) => {
      const env = await seed();
      await renderApp(c.element, { env, settings: c.onboarding === true ? { onboarded: false } : {} });
      await settle();
      await c.steps?.();
      await settle();

      const tree = screen.toJSON();
      // The audit really looked at touchables (Home with data is read-only: it has none).
      if (c.name !== "home") expect(touchableCount(tree)).toBeGreaterThan(0);
      expect(a11yViolations(tree)).toEqual([]);
      expect(layoutViolations(tree)).toEqual([]);
      if (c.name !== "lock screen") expect(hasHeading(tree)).toBe(true);
    },
  );
});

describe("accessibility of the shared components", () => {
  it("passes with long text, a selected state and an error", async () => {
    const long =
      "A very long label that has to wrap onto several lines at twice the font size instead of being cut off";
    await render(
      <>
        <Chip label={long} onPress={() => undefined} selected role="radio" />
        <Button label={long} onPress={() => undefined} variant="destructive" />
        <ListRow
          title={long}
          subtitle={long}
          trailing={<MoneyText cents={-123456789} kind="expense" />}
          onPress={() => undefined}
        />
        <EmptyState message={long} action={{ label: long, onPress: () => undefined }} />
        <Toast message={long} action={{ label: "Undo", onPress: () => undefined }} onDismiss={() => undefined} />
      </>,
    );
    const tree = screen.toJSON();
    expect(a11yViolations(tree)).toEqual([]);
    expect(layoutViolations(tree)).toEqual([]);
  });

  it("the checks bite: they flag a touchable with no role, a small target and a fixed-height text box", async () => {
    await render(
      <>
        <Pressable onPress={() => undefined} style={{ minHeight: 20 }}>
          <Text>tiny</Text>
        </Pressable>
        <View style={{ height: 30 }}>
          <Text numberOfLines={1}>clipped</Text>
        </View>
      </>,
    );
    const tree = screen.toJSON();
    expect(a11yViolations(tree).join("|")).toMatch(/no accessibilityRole/u);
    expect(a11yViolations(tree).join("|")).toMatch(/under 48 dp/u);
    expect(layoutViolations(tree).join("|")).toMatch(/fixed height of 30/u);
    expect(layoutViolations(tree).join("|")).toMatch(/numberOfLines/u);
  });
});

describe("font scale 2.0", () => {
  it("keeps the layout rules at twice the system font size (no fixed sizes on anything that holds text)", async () => {
    const spy = jest.spyOn(PixelRatio, "getFontScale").mockReturnValue(2);
    try {
      expect(PixelRatio.getFontScale()).toBe(2);
      const env = await seed();
      await renderApp(<TransactionsScreen />, { env });
      await settle();
      const tree = screen.toJSON();
      expect(layoutViolations(tree)).toEqual([]);
      // This screen has no bar charts; the only fixed-height views anywhere are the 8 dp bars of Budgets and Reports, which hold no text.
      const fixed: Record<string, number> = {};
      const visit = (n: unknown): void => {
        const node = n as { props?: { style?: unknown }; children?: unknown[] };
        const style = Array.isArray(node.props?.style)
          ? Object.assign({}, ...((node.props?.style as object[] | undefined) ?? []))
          : (node.props?.style as Record<string, unknown> | undefined);
        if (typeof style?.height === "number") fixed[String(style.height)] = (fixed[String(style.height)] ?? 0) + 1;
        for (const c of node.children ?? []) if (typeof c !== "string") visit(c);
      };
      visit(tree);
      expect(Object.keys(fixed)).toEqual([]);
    } finally {
      spy.mockRestore();
    }
  });
});

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = `${dir}/${name}`;
    return statSync(path).isDirectory() ? sources(path) : [path];
  });
}

describe("no swipe-only actions and no motion to reduce", () => {
  const root = `${__dirname}/../..`;
  const files = [...sources(`${root}/src`), ...sources(`${root}/app`)].filter((p) => /\.(ts|tsx)$/u.test(p));

  it("has no swipe or gesture handlers, so every action is a visible button", () => {
    const offenders = files.filter((p) =>
      /Swipeable|PanResponder|onSwipe|react-native-gesture-handler|ScrollView[^>]*onScrollEndDrag/u.test(
        readFileSync(p, "utf8"),
      ),
    );
    expect(offenders).toEqual([]);
  });

  it("has no animation, so reduced motion is respected by construction", () => {
    const offenders = files.filter((p) =>
      /\bAnimated\b|LayoutAnimation|react-native-reanimated|useNativeDriver/u.test(readFileSync(p, "utf8")),
    );
    expect(offenders).toEqual([]);
  });
});
