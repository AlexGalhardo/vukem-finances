/**
 * A small validator for the subset of JSON Schema that docs/export-format.schema.json uses: type (one or a list),
 * const, enum, properties, required, additionalProperties false, items, minLength and local $ref. It is not a
 * dependency of the app; the negative tests in backup.test.ts show it rejects what the schema forbids.
 */
export interface Schema {
	$ref?: string;
	type?: string | string[];
	const?: unknown;
	enum?: unknown[];
	properties?: Record<string, Schema>;
	required?: string[];
	additionalProperties?: boolean;
	items?: Schema;
	minLength?: number;
	$defs?: Record<string, Schema>;
}

function typeOf(value: unknown): string {
	if (value === null) return "null";
	if (Array.isArray(value)) return "array";
	if (typeof value === "number") return Number.isInteger(value) ? "integer" : "number";
	return typeof value;
}

export function validate(root: Schema, value: unknown, schema: Schema = root, path = "$"): string[] {
	if (schema.$ref !== undefined) {
		const name = schema.$ref.replace("#/$defs/", "");
		const target = root.$defs?.[name];
		return target === undefined ? [`${path}: unknown $ref ${schema.$ref}`] : validate(root, value, target, path);
	}

	const errors: string[] = [];
	if (schema.const !== undefined && value !== schema.const) errors.push(`${path}: must be ${JSON.stringify(schema.const)}`);
	if (schema.enum !== undefined && !schema.enum.includes(value)) errors.push(`${path}: not one of ${JSON.stringify(schema.enum)}`);

	if (schema.type !== undefined) {
		const allowed = Array.isArray(schema.type) ? schema.type : [schema.type];
		const actual = typeOf(value);
		// An integer is also a number for JSON Schema.
		if (!allowed.includes(actual) && !(actual === "integer" && allowed.includes("number"))) {
			errors.push(`${path}: expected ${allowed.join("|")}, got ${actual}`);
			return errors;
		}
	}
	if (typeof value === "string" && schema.minLength !== undefined && value.length < schema.minLength) {
		errors.push(`${path}: shorter than ${schema.minLength}`);
	}

	if (Array.isArray(value) && schema.items !== undefined) {
		const items = schema.items;
		value.forEach((item, i) => errors.push(...validate(root, item, items, `${path}[${i}]`)));
	}

	if (typeOf(value) === "object" && schema.properties !== undefined) {
		const obj = value as Record<string, unknown>;
		for (const key of schema.required ?? []) {
			if (!(key in obj)) errors.push(`${path}: missing ${key}`);
		}
		for (const [key, v] of Object.entries(obj)) {
			const sub = schema.properties[key];
			if (sub === undefined) {
				if (schema.additionalProperties === false) errors.push(`${path}: unexpected ${key}`);
				continue;
			}
			errors.push(...validate(root, v, sub, `${path}.${key}`));
		}
	}
	return errors;
}
