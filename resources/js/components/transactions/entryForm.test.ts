import { describe, expect, test } from "bun:test";
import {
  addTag,
  categoriesFor,
  centsToInput,
  emptyTransactionDraft,
  emptyTransferDraft,
  filterPicker,
  type QuickAddOptions,
  readProblem,
  sendJSON,
} from "./entryForm";

const options: QuickAddOptions = {
  accounts: [
    { id: 4, name: "Checking", kind: "checking" },
    { id: 9, name: "Savings", kind: "savings" },
  ],
  categories: [
    { id: 1, name: "Groceries", kind: "expense" },
    { id: 2, name: "Salary", kind: "income" },
  ],
  today: "2026-11-21",
  currency: "USD",
  tags: ["Trip"],
};

describe("drafts", () => {
  test("a new entry starts on the first account and today, with the kind of the action", () => {
    expect(emptyTransactionDraft("income", options)).toEqual({
      kind: "income",
      amount: "",
      accountId: "4",
      categoryId: "",
      description: "",
      date: "2026-11-21",
      tags: [],
    });
  });

  test("a new transfer goes from the first account to the second", () => {
    const draft = emptyTransferDraft(options);
    expect([draft.fromAccountId, draft.toAccountId]).toEqual(["4", "9"]);
  });

  test("with no account the pickers start empty", () => {
    expect(emptyTransactionDraft("expense", { ...options, accounts: [] }).accountId).toBe("");
  });
});

describe("centsToInput", () => {
  test("writes cents as text without floats", () => {
    expect(centsToInput(0)).toBe("0.00");
    expect(centsToInput(5)).toBe("0.05");
    expect(centsToInput(1250)).toBe("12.50");
    expect(centsToInput(-1250)).toBe("-12.50");
    // 2^53 - 1 cents, the largest amount the server accepts, stays exact.
    expect(centsToInput(9007199254740991)).toBe("90071992547409.91");
  });
});

describe("pickers", () => {
  test("a category picker offers only the categories of the kind", () => {
    expect(categoriesFor(options, "expense").map((c) => c.name)).toEqual(["Groceries"]);
    expect(categoriesFor(options, "income").map((c) => c.name)).toEqual(["Salary"]);
  });

  test("typing narrows the list, and the chosen name shows everything again", () => {
    expect(filterPicker(options.accounts, "sav").map((a) => a.name)).toEqual(["Savings"]);
    expect(filterPicker(options.accounts, "  ").length).toBe(2);
    expect(filterPicker(options.accounts, "zzz")).toEqual([]);
    expect(filterPicker(options.accounts, "checking", options.accounts[0]).length).toBe(2);
  });
});

describe("readProblem", () => {
  test("reads the detail of a problem answer", () => {
    expect(readProblem({ detail: "Choose two different accounts." }, "x").message).toBe(
      "Choose two different accounts.",
    );
  });

  test("reads the first message of each invalid field", () => {
    const problem = readProblem({ detail: "2 field(s) are invalid.", errors: { amount: ["Too short", "Other"] } }, "x");
    expect(problem.fields).toEqual({ amount: "Too short" });
    expect(problem.message).toBe("Too short");
  });

  test("shows the offline text for the offline code only when one is given", () => {
    const body = { code: "offline", detail: "No connection to the AI provider." };
    expect(readProblem(body, "x", "Sem conexão").message).toBe("Sem conexão");
    expect(readProblem(body, "x").message).toBe("No connection to the AI provider.");
    expect(readProblem({ code: "other", detail: "d" }, "x", "Sem conexão").message).toBe("d");
  });

  test("falls back when the answer is not a problem", () => {
    expect(readProblem(null, "fallback").message).toBe("fallback");
    expect(readProblem("<html>", "fallback").message).toBe("fallback");
  });
});

describe("sendJSON", () => {
  const draft = emptyTransactionDraft("expense", options);

  function withFetch(answer: () => Promise<Response>, run: () => Promise<void>): Promise<void> {
    const original = globalThis.fetch;
    globalThis.fetch = answer as unknown as typeof fetch;
    return run().finally(() => {
      globalThis.fetch = original;
    });
  }

  test("sends the amount as typed, never as a number", async () => {
    let body = "";
    const original = globalThis.fetch;
    globalThis.fetch = (async (_url: unknown, init?: RequestInit) => {
      body = String(init?.body);
      return new Response("{}", { status: 201 });
    }) as unknown as typeof fetch;
    try {
      expect(await sendJSON("/transactions", "POST", { ...draft, amount: "1.234,56" }, "x")).toEqual({ ok: true });
    } finally {
      globalThis.fetch = original;
    }
    expect(JSON.parse(body).amount).toBe("1.234,56");
  });

  test("turns a 422 into the message of the server", async () => {
    await withFetch(
      async () => new Response(JSON.stringify({ detail: "Choose two different accounts." }), { status: 422 }),
      async () => {
        const result = await sendJSON("/transfers", "POST", draft, "fallback");
        expect(result).toEqual({ ok: false, message: "Choose two different accounts.", fields: {} });
      },
    );
  });

  test("a network failure gives the fallback message", async () => {
    await withFetch(
      async () => {
        throw new Error("offline");
      },
      async () => {
        const result = await sendJSON("/transactions", "POST", draft, "fallback");
        expect(result).toEqual({ ok: false, message: "fallback", fields: {} });
      },
    );
  });
});

describe("addTag", () => {
  test("trims, keeps the typed case and ignores an empty name", () => {
    expect(addTag([], "  Trip ")).toEqual(["Trip"]);
    expect(addTag(["Trip"], "   ")).toEqual(["Trip"]);
  });

  test("a name that is there in another letter case is not added twice", () => {
    expect(addTag(["Trip"], "trip")).toEqual(["Trip"]);
  });

  test("stops at ten tags", () => {
    const ten = Array.from({ length: 10 }, (_, i) => `t${i}`);
    expect(addTag(ten, "one more")).toEqual(ten);
  });
});
