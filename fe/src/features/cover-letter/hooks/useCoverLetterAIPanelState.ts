import { useEffect, useState } from "react";
import { useCoverLetterAI } from "./useCoverLetterAI";

export interface CoverLetterAIPanelState {
  readonly jobDescription: string;
  readonly setJobDescription: (value: string) => void;
  readonly jobSearch: string;
  readonly setJobSearch: (value: string) => void;
  readonly generate: ReturnType<typeof useCoverLetterAI>["generate"];
}

interface PanelFields {
  readonly coverLetterId: string | undefined;
  readonly jobDescription: string;
  readonly jobSearch: string;
}

const emptyFields = (coverLetterId: string | undefined): PanelFields => ({
  coverLetterId,
  jobDescription: "",
  jobSearch: "",
});

/**
 * What the AI panel holds — the job description, the job search and the
 * generated (paid) draft — kept by whoever renders the panel rather than by the
 * panel itself.
 *
 * The editor shows the panel as a side column on desktop and as a sheet on
 * smaller screens. Those are two different mount points, so state kept inside
 * the panel died whenever the viewport crossed the breakpoint, taking an
 * unapplied AI result with it. Owned by the page, it survives the move and
 * resets only when a different letter is opened.
 */
export function useCoverLetterAIPanelState(
  coverLetterId: string | undefined,
): CoverLetterAIPanelState {
  const { generate } = useCoverLetterAI();
  const [fields, setFields] = useState<PanelFields>(() =>
    emptyFields(coverLetterId),
  );

  if (fields.coverLetterId !== coverLetterId) {
    setFields(emptyFields(coverLetterId));
  }

  const { reset } = generate;
  useEffect(() => {
    reset();
  }, [coverLetterId, reset]);

  return {
    jobDescription: fields.jobDescription,
    setJobDescription: (jobDescription) =>
      setFields((current) => ({ ...current, jobDescription })),
    jobSearch: fields.jobSearch,
    setJobSearch: (jobSearch) =>
      setFields((current) => ({ ...current, jobSearch })),
    generate,
  };
}
