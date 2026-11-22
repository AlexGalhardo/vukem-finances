import { referenceHarness } from "../helpers/reference-server";
import { loadScenario, runScenario, scenarioFiles } from "../helpers/scenario-runner";

describe("the shared sync scenarios, run through the client engine", () => {
  it("finds the scenario files", () => {
    expect(scenarioFiles().length).toBeGreaterThanOrEqual(13);
  });

  it.each(scenarioFiles())("%s", async (file) => {
    const scenario = loadScenario(file);
    const harness = referenceHarness(scenario.workspace.plan, scenario.workspace.devices);
    expect(await runScenario(scenario, harness)).toEqual([]);
  });
});
