import { describe, it, expect } from "vitest";
import {
  buildUnifiedColumns,
  placeJob,
  columnMoveTarget,
  shouldHideBase,
} from "../unifiedBoard";
import type { JobDTO, StageTemplateDTO } from "@/shared/types/api";

const tpl = (
  id: string,
  name: string,
  order: number,
  phase: StageTemplateDTO["phase"],
): StageTemplateDTO => ({
  id,
  name,
  order,
  phase,
  created_at: "2026-01-01T00:00:00Z",
});

const templates: StageTemplateDTO[] = [
  tpl("t-screen", "Screening", 1, "in_progress"),
  tpl("t-tech", "Technical Interview", 2, "in_progress"),
  tpl("t-nego", "Negotiating", 1, "offer"),
  tpl("t-research", "Researching", 1, "wishlist"),
];

const job = (over: Partial<JobDTO>): JobDTO =>
  ({
    id: "j1",
    title: "Job",
    status: "applied",
    is_favorite: false,
    created_at: "",
    updated_at: "",
    ...over,
  }) as JobDTO;

describe("buildUnifiedColumns", () => {
  it("lays out base columns and stage columns in pipeline order", () => {
    const ids = buildUnifiedColumns(templates, false).map((c) => c.id);
    expect(ids).toEqual([
      "phase:wishlist",
      "stage:t-research",
      "phase:applied",
      "stage:t-screen",
      "stage:t-tech",
      "phase:offer",
      "stage:t-nego",
      "phase:rejected",
    ]);
  });

  it("appends the archived column only when the filter is on", () => {
    const ids = buildUnifiedColumns(templates, true).map((c) => c.id);
    expect(ids[ids.length - 1]).toBe("phase:archived");
  });

  it("works with zero templates — bare four-column tracker", () => {
    const ids = buildUnifiedColumns([], false).map((c) => c.id);
    expect(ids).toEqual([
      "phase:wishlist",
      "phase:applied",
      "phase:offer",
      "phase:rejected",
    ]);
  });
});

describe("placeJob", () => {
  it("terminal statuses win over the current stage", () => {
    const j = job({ status: "rejected", current_stage_name: "Screening" });
    expect(placeJob(j, templates)).toBe("phase:rejected");
  });

  it("active job stands on its stage column", () => {
    const j = job({ status: "applied", current_stage_name: "Screening" });
    expect(placeJob(j, templates)).toBe("stage:t-screen");
  });

  it("on_hold keeps the stage position (pause is a badge, not a column)", () => {
    const j = job({ status: "on_hold", current_stage_name: "Technical Interview" });
    expect(placeJob(j, templates)).toBe("stage:t-tech");
  });

  it("active job without stage falls into the Applied base", () => {
    expect(placeJob(job({ status: "applied" }), templates)).toBe(
      "phase:applied",
    );
  });

  it("saved with a wishlist stage stands on it, otherwise Wishlist base", () => {
    expect(
      placeJob(
        job({ status: "saved", current_stage_name: "Researching" }),
        templates,
      ),
    ).toBe("stage:t-research");
    expect(placeJob(job({ status: "saved" }), templates)).toBe(
      "phase:wishlist",
    );
  });

  it("offer with an offer stage stands on it, otherwise Offer base", () => {
    expect(
      placeJob(
        job({ status: "offer", current_stage_name: "Negotiating" }),
        templates,
      ),
    ).toBe("stage:t-nego");
    expect(placeJob(job({ status: "offer" }), templates)).toBe("phase:offer");
  });

  it("a stage from another phase does not capture an active card", () => {
    // active card whose current stage is an offer-phase stage → Applied base
    const j = job({ status: "applied", current_stage_name: "Negotiating" });
    expect(placeJob(j, templates)).toBe("phase:applied");
  });
});

describe("columnMoveTarget", () => {
  it("maps stage columns to stage targets", () => {
    const cols = buildUnifiedColumns(templates, false);
    const stageCol = cols.find((c) => c.id === "stage:t-screen")!;
    expect(columnMoveTarget(stageCol)).toEqual({
      type: "stage",
      stage_template_id: "t-screen",
    });
  });

  it("maps phase columns to phase targets including rejected", () => {
    const cols = buildUnifiedColumns(templates, false);
    const rejected = cols.find((c) => c.id === "phase:rejected")!;
    expect(columnMoveTarget(rejected)).toEqual({
      type: "phase",
      phase: "rejected",
    });
  });

  it("archived is not a move target", () => {
    const cols = buildUnifiedColumns(templates, true);
    const archived = cols.find((c) => c.id === "phase:archived")!;
    expect(columnMoveTarget(archived)).toBeNull();
  });
});

describe("shouldHideBase", () => {
  const withAppliedStage = buildUnifiedColumns(
    [tpl("t-a", "Applied", 1, "applied")],
    false,
  );
  const appliedBase = withAppliedStage.find((c) => c.id === "phase:applied")!;
  const rejectedBase = withAppliedStage.find((c) => c.id === "phase:rejected")!;

  it("hides an empty base when the phase has stage columns", () => {
    expect(shouldHideBase(appliedBase, 0, withAppliedStage)).toBe(true);
  });

  it("keeps the base when it holds cards", () => {
    expect(shouldHideBase(appliedBase, 2, withAppliedStage)).toBe(false);
  });

  it("keeps the base when the phase has no stages", () => {
    const bare = buildUnifiedColumns([], false);
    const base = bare.find((c) => c.id === "phase:applied")!;
    expect(shouldHideBase(base, 0, bare)).toBe(false);
  });

  it("never hides Rejected", () => {
    expect(shouldHideBase(rejectedBase, 0, withAppliedStage)).toBe(false);
  });
});
