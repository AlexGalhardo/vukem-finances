import { fireEvent, render, screen } from "@testing-library/react-native";
import { useState } from "react";
import { AmountField } from "../../src/components/AmountField";
import { Chip } from "../../src/components/Chip";
import { EmptyState } from "../../src/components/EmptyState";
import { ListRow } from "../../src/components/ListRow";
import { MoneyText } from "../../src/components/MoneyText";
import { ScreenHeader } from "../../src/components/ScreenHeader";
import { Toast } from "../../src/components/Toast";
import { TOUCH_MIN, UiProvider } from "../../src/components/ui";
import { spokenMoney } from "../../src/i18n/spoken";

describe("spokenMoney", () => {
	it("says integer cents in words", () => {
		expect(spokenMoney(-1250, "USD", "en")).toBe("minus 12 dollars 50 cents");
		expect(spokenMoney(100, "USD", "en")).toBe("1 dollar");
		expect(spokenMoney(1, "USD", "en")).toBe("1 cent");
		expect(spokenMoney(0, "USD", "en")).toBe("0 dollars");
		expect(spokenMoney(150, "BRL", "pt-BR")).toBe("1 real 50 centavos");
		expect(spokenMoney(-200, "EUR", "es")).toBe("menos 2 euros");
		expect(spokenMoney(5, "XYZ", "en")).toBe("5 cents");
	});

	it("refuses a float", () => {
		expect(() => spokenMoney(1.5, "USD", "en")).toThrow(RangeError);
	});
});

describe("MoneyText", () => {
	it("marks an expense with a sign and a spoken label, not only a color", async () => {
		await render(<MoneyText cents={1250} kind="expense" />);
		const node = screen.getByLabelText("Expense minus 12 dollars 50 cents");
		expect(node).toHaveTextContent("-$12.50");
	});

	it("marks income with a plus sign", async () => {
		await render(<MoneyText cents={-500} kind="income" />);
		expect(screen.getByLabelText("Income 5 dollars")).toHaveTextContent("+$5.00");
	});

	it("speaks a neutral negative balance", async () => {
		await render(<MoneyText cents={-1250} />);
		expect(screen.getByLabelText("minus 12 dollars 50 cents")).toHaveTextContent("-$12.50");
	});

	it("follows the language and currency of the provider", async () => {
		await render(
			<UiProvider locale="pt-BR" scheme="dark" currency="BRL">
				<MoneyText cents={123456} kind="expense" />
			</UiProvider>,
		);
		expect(screen.getByLabelText("Despesa menos 1234 reais 56 centavos")).toBeTruthy();
	});
});

describe("AmountField", () => {
	function Harness(): React.ReactNode {
		const [value, setValue] = useState("");
		return <AmountField value={value} onChangeText={setValue} />;
	}

	it("has a label, a decimal keypad and reports a bad amount to a screen reader", async () => {
		await render(<Harness />);
		const input = screen.getByLabelText("Amount");
		expect(input.props.keyboardType).toBe("decimal-pad");
		expect(screen.queryByRole("alert")).toBeNull();

		await fireEvent.changeText(input, "1.234,56");
		expect(screen.queryByText("Enter an amount like 12.50 or 12,50.")).toBeNull();

		await fireEvent.changeText(input, "1.005");
		expect(screen.getByText("Enter an amount like 12.50 or 12,50.")).toBeTruthy();
	});

	it("is at least 48 dp tall", async () => {
		await render(<Harness />);
		const style = screen.getByLabelText("Amount").props.style as { minHeight: number };
		expect(style.minHeight).toBeGreaterThanOrEqual(48);
		expect(TOUCH_MIN).toBe(48);
	});
});

describe("Chip", () => {
	it("is a button that can be found by role and name", async () => {
		const onPress = jest.fn();
		await render(<Chip label="Groceries" onPress={onPress} />);
		await fireEvent.press(screen.getByRole("button", { name: "Groceries" }));
		expect(onPress).toHaveBeenCalledTimes(1);
	});

	it("exposes the selected state and draws a check mark besides the color", async () => {
		await render(<Chip label="Food" role="radio" selected onPress={() => {}} />);
		const chip = screen.getByRole("radio", { name: "Food" });
		expect(chip).toBeSelected();
		expect(chip).toHaveTextContent("✓ Food");
	});
});

describe("ListRow", () => {
	it("reads title and subtitle together and shows the trailing slot", async () => {
		const onPress = jest.fn();
		await render(
			<ListRow title="Lunch" subtitle="Checking" trailing={<MoneyText cents={1250} kind="expense" />} onPress={onPress} />,
		);
		await fireEvent.press(screen.getByRole("button", { name: "Lunch, Checking" }));
		expect(onPress).toHaveBeenCalled();
		expect(screen.getByLabelText("Expense minus 12 dollars 50 cents")).toBeTruthy();
	});

	it("is not a button without onPress", async () => {
		await render(<ListRow title="Static" />);
		expect(screen.queryByRole("button")).toBeNull();
		expect(screen.getByText("Static")).toBeTruthy();
	});
});

describe("ScreenHeader", () => {
	it("is the heading of the screen, with back and action buttons", async () => {
		const onBack = jest.fn();
		const onAction = jest.fn();
		await render(<ScreenHeader title="Accounts" onBack={onBack} action={{ label: "Add", onPress: onAction }} />);
		expect(screen.getByRole("header", { name: "Accounts" })).toBeTruthy();
		await fireEvent.press(screen.getByRole("button", { name: "Back" }));
		await fireEvent.press(screen.getByRole("button", { name: "Add" }));
		expect(onBack).toHaveBeenCalled();
		expect(onAction).toHaveBeenCalled();
	});
});

describe("EmptyState", () => {
	it("shows the message and a button when something can be done", async () => {
		const onPress = jest.fn();
		await render(<EmptyState message="No accounts yet." action={{ label: "New account", onPress }} />);
		expect(screen.getByText("No accounts yet.")).toBeTruthy();
		await fireEvent.press(screen.getByRole("button", { name: "New account" }));
		expect(onPress).toHaveBeenCalled();
	});

	it("has no button without an action", async () => {
		await render(<EmptyState message="Nothing" />);
		expect(screen.queryByRole("button")).toBeNull();
	});
});

describe("Toast", () => {
	it("is a polite alert with undo and close buttons", async () => {
		const undo = jest.fn();
		const dismiss = jest.fn();
		await render(<Toast message="Deleted" action={{ label: "Undo", onPress: undo }} onDismiss={dismiss} />);
		expect(screen.getByRole("alert")).toBeTruthy();
		expect(screen.getByText("Deleted")).toBeTruthy();
		await fireEvent.press(screen.getByRole("button", { name: "Undo" }));
		await fireEvent.press(screen.getByRole("button", { name: "Close" }));
		expect(undo).toHaveBeenCalled();
		expect(dismiss).toHaveBeenCalled();
	});
});
