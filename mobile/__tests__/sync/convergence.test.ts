import { ENTITIES, type EntityName } from "../../src/db/entities";
import { runConvergence } from "../helpers/convergence";
import { ReferenceServer } from "../helpers/reference-server";

describe("convergence against the reference server", () => {
  it("leaves two phones and a third client with the live rows of the server after 50 seeded runs", async () => {
    const conflicts: Record<string, number> = {};
    for (let seed = 1; seed <= 50; seed++) {
      const server = new ReferenceServer({ plan: "pro", devices: { A: "owner", B: "owner", W: "owner" } });
      const report = await runConvergence(seed, {
        newTransport: async (name) => server.transportFor(name),
        serverRows: async () => {
          const out = new Map<EntityName, Map<string, Record<string, unknown>>>();
          for (const entity of Object.keys(ENTITIES) as EntityName[]) {
            out.set(
              entity,
              new Map(
                [...server.live(entity)].map(([uid, row]) => [
                  uid,
                  Object.fromEntries(Object.keys(ENTITIES[entity].fields).map((n) => [n, row[n]])),
                ]),
              ),
            );
          }
          return out;
        },
      });
      expect({ seed, differences: report.differences, problems: report.problems }).toEqual({
        seed,
        differences: [],
        problems: [],
      });
      for (const [k, n] of Object.entries(report.conflicts)) conflicts[k] = (conflicts[k] ?? 0) + n;
    }
    // The runs do exercise conflicts: a property test that never conflicts proves little.
    expect(Object.keys(conflicts).length).toBeGreaterThan(0);
  });
});
