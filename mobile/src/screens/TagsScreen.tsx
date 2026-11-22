import { type ReactNode, useState } from "react";
import { Text } from "react-native";
import { describeError } from "../app/errors";
import { type App, useApp, useLoad } from "../app/state";
import { Button } from "../components/Button";
import { DeleteButton } from "../components/DeleteButton";
import { EmptyState } from "../components/EmptyState";
import { ListRow } from "../components/ListRow";
import { Page } from "../components/Page";
import { TextField } from "../components/TextField";
import { useUi } from "../components/ui";

/** Tags: create, rename, delete (which removes the tag from every transaction that held it). */
export function TagsScreen(props: { onBack?: () => void }): ReactNode {
	const app = useApp();
	const ui = useUi();
	const [editing, setEditing] = useState<string | "new" | null>(null);
	const [name, setName] = useState("");
	const [error, setError] = useState<string | null>(null);
	const tags = useLoad((a: App) => a.repo.list("tag"));

	function open(target: string | "new"): void {
		setName(tags?.find((t) => t.uid === target)?.name ?? "");
		setError(null);
		setEditing(target);
	}

	async function save(): Promise<void> {
		if (name.trim() === "") {
			setError(ui.t("required"));
			return;
		}
		try {
			if (editing === "new") {
				await app.repo.create("tag", { name: name.trim(), color: "" });
			} else if (editing !== null) {
				await app.repo.update("tag", editing, { name: name.trim() });
			}
		} catch (e) {
			setError(describeError(ui, e));
			return;
		}
		setEditing(null);
		app.changed();
	}

	async function remove(uid: string): Promise<void> {
		await app.ledger.deleteTag(uid);
		setEditing(null);
		app.changed();
	}

	if (editing !== null) {
		return (
			<Page title={editing === "new" ? ui.t("tagNew") : ui.t("tagEdit")} onBack={() => setEditing(null)}>
				<TextField label={ui.t("name")} value={name} onChangeText={setName} />
				{error === null ? null : (
					<Text accessibilityRole="alert" style={{ color: ui.colors.expense }}>
						{error}
					</Text>
				)}
				<Button label={ui.t("save")} onPress={() => void save()} />
				{editing === "new" ? null : <DeleteButton onConfirm={() => void remove(editing)} />}
			</Page>
		);
	}

	return (
		<Page title={ui.t("titleTags")} onBack={props.onBack} action={{ label: ui.t("add"), onPress: () => open("new") }}>
			{tags === undefined ? null : tags.length === 0 ? (
				<EmptyState message={ui.t("tagsEmpty")} action={{ label: ui.t("tagNew"), onPress: () => open("new") }} />
			) : (
				tags.map((t) => <ListRow key={t.uid} title={t.name} onPress={() => open(t.uid)} />)
			)}
		</Page>
	);
}
