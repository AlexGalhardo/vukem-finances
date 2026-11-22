package tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func TestBotCommands(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, w)
	linkChat(t, bot, w, 9601)

	fin := softService(t, app)
	ctx := tgCtx(w)
	main := firstAccount(t, app, w)

	savings, err := fin.CreateAccount(ctx, "Savings & loans", models.AccountSavings, "100.00")
	if err != nil {
		t.Fatal(err)
	}

	for _, in := range []finance.TxInput{
		{AccountID: main, Kind: models.KindIncome, Amount: "3000.00", Description: "salary", Date: today()},
		{AccountID: main, Kind: models.KindExpense, Amount: "1234.56", Description: "<b>rent</b> & fees", Date: today()},
		{AccountID: main, CategoryID: categoryID(t, app, w, "Groceries"), Kind: models.KindExpense, Amount: "80.00", Description: "market", Date: today()},
	} {
		if _, err := fin.CreateTransaction(ctx, in); err != nil {
			t.Fatal(err)
		}
	}

	if err := fin.SetBudget(ctx, categoryID(t, app, w, "Groceries"), "100.00"); err != nil {
		t.Fatal(err)
	}

	t.Run("balance", func(t *testing.T) {
		say(t, bot, 9601, "/balance")

		// 3000 - 1234.56 - 80 = 1685.44 on the main account, 100 on savings, 1785.44 in all.
		mustContain(t, lastText(t, f, 9601), "Main account: $1,685.44", "Savings &amp; loans: $100.00", "Net worth: $1,785.44")
	})

	t.Run("month", func(t *testing.T) {
		say(t, bot, 9601, "/month")
		mustContain(t, lastText(t, f, 9601), "Income: $3,000.00", "Expense: $1,314.56", "Net: $1,685.44")
	})

	t.Run("last escapes what the person wrote", func(t *testing.T) {
		say(t, bot, 9601, "/last")

		text := lastText(t, f, 9601)
		mustContain(t, text, "-$1,234.56", "&lt;b&gt;rent&lt;/b&gt; &amp; fees", "+$3,000.00", "market")

		if strings.Contains(text, "<b>rent</b>") {
			t.Fatalf("a description reached the chat as markup: %q", text)
		}
	})

	t.Run("budgets", func(t *testing.T) {
		say(t, bot, 9601, "/budgets")
		mustContain(t, lastText(t, f, 9601), "Groceries: $80.00 of $100.00 (80%)")
	})

	t.Run("help and unknown commands", func(t *testing.T) {
		for _, text := range []string{"/help", "/start@fake_finance_bot", "/nonsense", "/HELP@fake_finance_bot"} {
			say(t, bot, 9601, text)
		}

		mustContain(t, lastText(t, f, 9601), "Commands")
	})

	t.Run("account picks where entries go", func(t *testing.T) {
		say(t, bot, 9601, "/account")

		call, _ := f.LastSent(9601)
		if got := fakes.Buttons(call); len(got) != 2 {
			t.Fatalf("account buttons = %v", got)
		}

		tap(t, bot, 9601, buttonData(t, f, 9601, "Savings"))
		mustContain(t, lastText(t, f, 9601), "Default account: <b>Savings &amp; loans</b>")

		say(t, bot, 9601, "-7 stamp")
		mustContain(t, lastText(t, f, 9601), "Account: Savings &amp; loans")
		tap(t, bot, 9601, buttonData(t, f, 9601, "No category"))

		txs := transactionsOf(t, app, w)
		if last := txs[len(txs)-1]; last.AccountID != savings.ID || last.AmountCents != 700 {
			t.Fatalf("saved %+v, want the Savings account", last)
		}

		// An account of another workspace cannot be chosen by its uid.
		other := signUp(t, app, "Zed", "zed@example.com")
		foreign := firstOf[models.Account](t, app, other, "")

		tap(t, bot, 9601, "ac:"+foreign.UID)

		link, _, _ := bot.Store.LinkByChat(t.Context(), 9601)
		if link.DefaultAccountUID.V != savings.UID {
			t.Fatalf("a foreign account became the default: %q", link.DefaultAccountUID.V)
		}
	})

	t.Run("language", func(t *testing.T) {
		say(t, bot, 9601, "/lang")
		tap(t, bot, 9601, "lg:pt-BR")
		mustContain(t, lastText(t, f, 9601), "Idioma")

		say(t, bot, 9601, "/help")
		mustContain(t, lastText(t, f, 9601), "Comandos")

		say(t, bot, 9601, "/month")
		mustContain(t, lastText(t, f, 9601), "Receitas: $3,000.00")

		tap(t, bot, 9601, "lg:es")
		say(t, bot, 9601, "/help")
		mustContain(t, lastText(t, f, 9601), "Comandos")
		mustContain(t, lastText(t, f, 9601), "ayer")

		// An unknown language is refused and changes nothing.
		tap(t, bot, 9601, "lg:fr")

		link, _, _ := bot.Store.LinkByChat(t.Context(), 9601)
		if link.Language != "es" {
			t.Fatalf("language = %q", link.Language)
		}
	})
}

// Switching workspace checks the membership at the tap, and the new workspace
// brings its own role and data.
func TestWorkspaceSwitchRechecksMembership(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	cat := signUp(t, app, "Cat", "cat@example.com")

	join(t, app, ana, bob.User, models.RoleViewer)

	// Bob starts in his own workspace.
	linkChat(t, bot, bob, 9701)
	say(t, bot, 9701, "-1 mine")
	tap(t, bot, 9701, buttonData(t, f, 9701, "No category"))

	if got := len(transactionsOf(t, app, bob)); got != 1 {
		t.Fatalf("%d entries in Bob's own workspace", got)
	}

	// He picks Ana's, where he is a viewer: reading shows Ana's data, writing is refused.
	say(t, bot, 9701, "/workspace")

	call, _ := f.LastSent(9701)
	if got := fakes.Buttons(call); len(got) != 2 {
		t.Fatalf("workspace choices = %v, want his own and Ana's", got)
	}

	tap(t, bot, 9701, "ws:"+strconv.FormatInt(ana.Workspace.ID, 10))
	mustContain(t, lastText(t, f, 9701), "Workspace: <b>Ana household</b>")

	say(t, bot, 9701, "-2 theirs")
	mustContain(t, lastText(t, f, 9701), "read only")

	// A workspace he does not belong to is refused, and the choice stays.
	f.Reset()
	tap(t, bot, 9701, "ws:"+strconv.FormatInt(cat.Workspace.ID, 10))

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "not a member") {
		t.Fatalf("answer = %v", got)
	}

	for _, forged := range []string{"ws:0", "ws:-1", "ws:abc", "ws:", "ws:99999999999999999999"} {
		tap(t, bot, 9701, forged)
	}

	link, _, _ := bot.Store.LinkByChat(t.Context(), 9701)
	if link.CurrentWorkspaceID.V != ana.Workspace.ID {
		t.Fatalf("current workspace = %d, want Ana's", link.CurrentWorkspaceID.V)
	}

	// Promoted to member, the next message can record, at once.
	db, _ := app.DB()

	if _, err := db.SQL().ExecContext(t.Context(), "UPDATE memberships SET role = 'member' WHERE user_id = ? AND workspace_id = ?", bob.User.ID, ana.Workspace.ID); err != nil {
		t.Fatal(err)
	}

	say(t, bot, 9701, "-3 theirs now")
	tap(t, bot, 9701, buttonData(t, f, 9701, "No category"))

	if got := transactionsOf(t, app, ana); len(got) != 1 || got[0].AmountCents != 300 {
		t.Fatalf("Ana's workspace: %+v", got)
	}
}
