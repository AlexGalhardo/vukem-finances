import { Head, router, useForm, usePage } from "@inertiajs/react";
import { Archive, ArchiveRestore, Pencil, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import ConfirmDelete from "../components/ConfirmDelete";
import { ColorPicker, IconSelect, Mark } from "../components/categories/look";
import PageHeader from "../components/PageHeader";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Field, FieldError, FieldGroup, FieldLabel } from "../components/ui/field";
import { Input } from "../components/ui/input";
import { NativeSelect, NativeSelectOption } from "../components/ui/native-select";
import { useFinanceI18n } from "../lib/i18n";
import type { Category, Page, SharedProps, Tag } from "../page-props";

const kinds = ["expense", "income"] as const;

interface CategoryForm {
  name: string;
  kind: string;
  parentId: string;
  color: string;
  icon: string;
}

const emptyForm: CategoryForm = { name: "", kind: "expense", parentId: "", color: "", icon: "" };

export default function Categories({ categories, tags, colors, icons, maxTags }: Page<"Categories">) {
  const { t } = useFinanceI18n();
  const role = usePage<SharedProps>().props.workspace?.role;
  const canManage = role === "owner" || role === "admin";
  const [showArchived, setShowArchived] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);
  const [reassign, setReassign] = useState("");
  const [editingTag, setEditingTag] = useState<Tag | null>(null);
  const add = useForm<CategoryForm>(emptyForm);
  const edit = useForm<CategoryForm>(emptyForm);
  const addTag = useForm({ name: "", color: "" });
  const editTag = useForm({ name: "", color: "" });

  const visible = (c: Category): boolean => showArchived || !c.archived_at;
  const childrenOf = (parent: Category): Category[] =>
    categories.filter((c) => c.parent_id.Valid && c.parent_id.V === parent.id && visible(c));
  const tops = (kind: string): Category[] =>
    categories.filter((c) => c.kind === kind && !c.parent_id.Valid && visible(c));
  // A subcategory goes under a live top-level category of its own kind.
  const parentsFor = (kind: string, self?: number): Category[] =>
    categories.filter((c) => c.kind === kind && !c.parent_id.Valid && !c.archived_at && c.id !== self);

  function startEdit(category: Category): void {
    edit.setData({
      name: category.name,
      kind: category.kind,
      parentId: category.parent_id.Valid ? String(category.parent_id.V) : "",
      color: category.color,
      icon: category.icon,
    });
    edit.clearErrors();
    setEditing(category);
  }

  function startDelete(category: Category): void {
    setReassign("");
    setDeleting(category);
  }

  const hasChildren = (category: Category): boolean =>
    categories.some((c) => c.parent_id.Valid && c.parent_id.V === category.id);
  const targets = (category: Category): Category[] =>
    categories.filter((c) => c.kind === category.kind && c.id !== category.id && !c.archived_at);

  function submitAdd(event: FormEvent): void {
    event.preventDefault();
    add.post("/categories", { preserveScroll: true, onSuccess: () => add.reset("name") });
  }

  function submitEdit(event: FormEvent): void {
    event.preventDefault();
    if (!editing) return;
    edit.patch(`/categories/${editing.id}`, { preserveScroll: true, onSuccess: () => setEditing(null) });
  }

  function confirmDelete(): void {
    if (!deleting) return;
    const query = reassign === "" ? "" : `?reassignTo=${reassign}`;
    router.delete(`/categories/${deleting.id}${query}`, { preserveScroll: true, onSuccess: () => setDeleting(null) });
  }

  function row(category: Category, nested: boolean) {
    return (
      <li key={category.id} className={nested ? "ms-5 border-s ps-3" : ""}>
        <div className="flex items-center gap-2 py-1.5">
          <Mark color={category.color} icon={category.icon} />
          <span className={category.archived_at ? "text-muted-foreground" : "font-medium"}>{category.name}</span>
          {category.archived_at && <Badge variant="secondary">{t("archived")}</Badge>}
          {canManage && (
            <span className="ms-auto flex gap-1">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => startEdit(category)}
                aria-label={`${t("edit")} ${category.name}`}
              >
                <Pencil />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() =>
                  router.post(
                    `/categories/${category.id}/${category.archived_at ? "unarchive" : "archive"}`,
                    undefined,
                    {
                      preserveScroll: true,
                    },
                  )
                }
                aria-label={`${t(category.archived_at ? "unarchive" : "archive")} ${category.name}`}
              >
                {category.archived_at ? <ArchiveRestore /> : <Archive />}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => startDelete(category)}
                aria-label={`${t("delete")} ${category.name}`}
              >
                <Trash2 />
              </Button>
            </span>
          )}
        </div>
        {childrenOf(category).length > 0 && (
          <ul aria-label={`${t("subcategories")}: ${category.name}`}>
            {childrenOf(category).map((child) => row(child, true))}
          </ul>
        )}
      </li>
    );
  }

  function lookFields(form: typeof add | typeof edit, prefix: string, kind: string, self?: number) {
    return (
      <>
        <Field data-invalid={!!form.errors.parentId}>
          <FieldLabel htmlFor={`${prefix}-parent`}>{t("parentCategory")}</FieldLabel>
          <NativeSelect
            id={`${prefix}-parent`}
            value={form.data.parentId}
            onChange={(e) => form.setData("parentId", e.target.value)}
          >
            <NativeSelectOption value="">{t("noParent")}</NativeSelectOption>
            {parentsFor(kind, self).map((c) => (
              <NativeSelectOption key={c.id} value={c.id}>
                {c.name}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <FieldError>{form.errors.parentId}</FieldError>
        </Field>
        <IconSelect
          id={`${prefix}-icon`}
          icons={icons}
          value={form.data.icon}
          onChange={(icon) => form.setData("icon", icon)}
        />
        <ColorPicker
          name={`${prefix}-color`}
          colors={colors}
          value={form.data.color}
          onChange={(c) => form.setData("color", c)}
        />
      </>
    );
  }

  return (
    <>
      <Head title={t("categories")} />
      <PageHeader
        title={t("categories")}
        description={t("categoriesDescription")}
        actions={
          <Button variant="outline" aria-pressed={showArchived} onClick={() => setShowArchived((v) => !v)}>
            {t("showArchived")}
          </Button>
        }
      />
      <div className="grid gap-4 md:grid-cols-2">
        {kinds.map((kind) => (
          <Card key={kind}>
            <CardHeader>
              <CardTitle>{t(kind === "expense" ? "expenses" : "income")}</CardTitle>
            </CardHeader>
            <CardContent>
              {tops(kind).length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("emptyCategories")}</p>
              ) : (
                <ul>{tops(kind).map((c) => row(c, false))}</ul>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {canManage && (
        <Card>
          <CardHeader>
            <CardTitle>{t("newCategory")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={submitAdd} className="flex flex-col gap-5">
              <FieldGroup className="grid gap-4 sm:grid-cols-3">
                <Field data-invalid={!!add.errors.name}>
                  <FieldLabel htmlFor="new-category-name">{t("name")}</FieldLabel>
                  <Input
                    id="new-category-name"
                    required
                    maxLength={60}
                    value={add.data.name}
                    aria-invalid={!!add.errors.name}
                    onChange={(e) => add.setData("name", e.target.value)}
                  />
                  <FieldError>{add.errors.name}</FieldError>
                </Field>
                <Field data-invalid={!!add.errors.kind}>
                  <FieldLabel htmlFor="new-category-kind">{t("kind")}</FieldLabel>
                  <NativeSelect
                    id="new-category-kind"
                    value={add.data.kind}
                    onChange={(e) => add.setData({ ...add.data, kind: e.target.value, parentId: "" })}
                  >
                    <NativeSelectOption value="expense">{t("expense")}</NativeSelectOption>
                    <NativeSelectOption value="income">{t("income")}</NativeSelectOption>
                  </NativeSelect>
                  <FieldError>{add.errors.kind}</FieldError>
                </Field>
                {lookFields(add, "new-category", add.data.kind)}
              </FieldGroup>
              <div>
                <Button type="submit" disabled={add.processing}>
                  {t("addCategory")}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>{t("tagsTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-5">
          <p className="text-sm text-muted-foreground">
            {t("tagsDescription")} {tags.length} / {maxTags}
          </p>
          {tags.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("emptyTags")}</p>
          ) : (
            <ul className="flex flex-col">
              {tags.map((tag) => (
                <li key={tag.id} className="flex items-center gap-2 py-1.5">
                  <Mark color={tag.color} icon="tag" />
                  <span className="font-medium">{tag.name}</span>
                  {canManage && (
                    <span className="ms-auto flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`${t("edit")} ${tag.name}`}
                        onClick={() => {
                          editTag.setData({ name: tag.name, color: tag.color });
                          editTag.clearErrors();
                          setEditingTag(tag);
                        }}
                      >
                        <Pencil />
                      </Button>
                      <ConfirmDelete
                        href={`/tags/${tag.id}`}
                        name={`${tag.name}. ${t("confirmDeleteTag")}`}
                        trigger={
                          <Button type="button" variant="ghost" size="icon" aria-label={`${t("delete")} ${tag.name}`}>
                            <Trash2 />
                          </Button>
                        }
                      />
                    </span>
                  )}
                </li>
              ))}
            </ul>
          )}
          {canManage && (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                addTag.post("/tags", { preserveScroll: true, onSuccess: () => addTag.reset("name") });
              }}
              className="flex flex-col gap-4"
            >
              <FieldGroup className="grid gap-4 sm:grid-cols-2">
                <Field data-invalid={!!addTag.errors.name}>
                  <FieldLabel htmlFor="new-tag-name">{t("tagName")}</FieldLabel>
                  <Input
                    id="new-tag-name"
                    required
                    maxLength={40}
                    value={addTag.data.name}
                    aria-invalid={!!addTag.errors.name}
                    onChange={(e) => addTag.setData("name", e.target.value)}
                  />
                  <FieldError>{addTag.errors.name}</FieldError>
                </Field>
                <ColorPicker
                  name="new-tag-color"
                  colors={colors}
                  value={addTag.data.color}
                  onChange={(c) => addTag.setData("color", c)}
                />
              </FieldGroup>
              <div>
                <Button type="submit" disabled={addTag.processing || tags.length >= maxTags}>
                  {t("addTag")}
                </Button>
              </div>
            </form>
          )}
        </CardContent>
      </Card>

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("editCategory")}</DialogTitle>
            <DialogDescription>{editing?.name}</DialogDescription>
          </DialogHeader>
          <form onSubmit={submitEdit} className="flex flex-col gap-6">
            <FieldGroup>
              <Field data-invalid={!!edit.errors.name}>
                <FieldLabel htmlFor="edit-category-name">{t("name")}</FieldLabel>
                <Input
                  id="edit-category-name"
                  required
                  maxLength={60}
                  value={edit.data.name}
                  aria-invalid={!!edit.errors.name}
                  onChange={(e) => edit.setData("name", e.target.value)}
                />
                <FieldError>{edit.errors.name}</FieldError>
              </Field>
              {lookFields(edit, "edit-category", edit.data.kind, editing?.id)}
            </FieldGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={edit.processing}>
                {t("saveCategory")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={editingTag !== null} onOpenChange={(open) => !open && setEditingTag(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("editTag")}</DialogTitle>
            <DialogDescription>{editingTag?.name}</DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (!editingTag) return;
              editTag.patch(`/tags/${editingTag.id}`, { preserveScroll: true, onSuccess: () => setEditingTag(null) });
            }}
            className="flex flex-col gap-6"
          >
            <FieldGroup>
              <Field data-invalid={!!editTag.errors.name}>
                <FieldLabel htmlFor="edit-tag-name">{t("tagName")}</FieldLabel>
                <Input
                  id="edit-tag-name"
                  required
                  maxLength={40}
                  value={editTag.data.name}
                  aria-invalid={!!editTag.errors.name}
                  onChange={(e) => editTag.setData("name", e.target.value)}
                />
                <FieldError>{editTag.errors.name}</FieldError>
              </Field>
              <ColorPicker
                name="edit-tag-color"
                colors={colors}
                value={editTag.data.color}
                onChange={(c) => editTag.setData("color", c)}
              />
            </FieldGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditingTag(null)}>
                {t("cancel")}
              </Button>
              <Button type="submit" disabled={editTag.processing}>
                {t("saveTag")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("deleteCategory")}</DialogTitle>
            <DialogDescription>
              {deleting?.name}. {t("deleteCategoryBody")}
            </DialogDescription>
          </DialogHeader>
          {deleting && hasChildren(deleting) ? (
            <p role="alert" className="text-sm text-destructive">
              {t("deleteCategoryChildren")}
            </p>
          ) : (
            <Field>
              <FieldLabel htmlFor="reassign-to">{t("reassignTo")}</FieldLabel>
              <NativeSelect id="reassign-to" value={reassign} onChange={(e) => setReassign(e.target.value)}>
                <NativeSelectOption value="">{t("chooseCategory")}</NativeSelectOption>
                {(deleting ? targets(deleting) : []).map((c) => (
                  <NativeSelectOption key={c.id} value={c.id}>
                    {c.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDeleting(null)}>
              {t("cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={deleting === null || hasChildren(deleting) || (targets(deleting).length > 0 && reassign === "")}
              onClick={confirmDelete}
            >
              {t("delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
