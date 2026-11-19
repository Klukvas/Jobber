import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { LogoMark } from "@/shared/ui/Logo";
import { SUPPORT_EMAIL, SUPPORT_MAILTO } from "@/shared/lib/supportContact";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { ContentLanguageNotice } from "@/shared/ui/ContentLanguageNotice";

const MCP_ENDPOINT = "https://cursopus.com/api/mcp";
const CLAUDE_CODE_COMMAND = `claude mcp add --transport http cursopus ${MCP_ENDPOINT}`;
const TOKEN_CLIENT_CONFIG = JSON.stringify(
  {
    mcpServers: {
      cursopus: {
        url: MCP_ENDPOINT,
        headers: { Authorization: "Bearer <YOUR_TOKEN>" },
      },
    },
  },
  null,
  2,
);

interface ToolRow {
  readonly name: string;
  readonly does: string;
  readonly writes: boolean;
  readonly credits?: string;
}

// Mirrors the tools registered in be/modules/mcp. `writes` is the inverse of
// the server's readOnlyHint, so this table and the confirmation prompts agree.
const TOOLS: readonly ToolRow[] = [
  { name: "list_stages", does: "Lists your pipeline stages in order.", writes: false },
  { name: "list_jobs", does: "Lists your jobs, with filters and paging.", writes: false },
  { name: "get_job", does: "Reads one job with its full description.", writes: false },
  { name: "create_job", does: "Saves a job to your pipeline and can create its company.", writes: true },
  { name: "move_job", does: "Moves a job to another of your stages.", writes: true },
  { name: "archive_job", does: "Archives or restores a job.", writes: true },
  { name: "list_resumes", does: "Lists your builder resumes and uploaded resumes.", writes: false },
  { name: "get_resume", does: "Reads a builder resume with its sections.", writes: false },
  { name: "update_resume_section", does: "Edits one section of a builder resume.", writes: true },
  { name: "list_resume_snapshots", does: "Lists the saved versions of a resume.", writes: false },
  { name: "restore_resume_snapshot", does: "Restores a saved version of a resume.", writes: true },
  { name: "parse_job_page", does: "Turns page text you supply into a job draft without saving it.", writes: true, credits: "2" },
  { name: "tailor_resume", does: "Proposes edits to a resume for one job. Nothing is saved.", writes: true, credits: "7" },
  { name: "apply_tailoring", does: "Saves the edits you selected as a new resume. The original stays.", writes: true },
  { name: "generate_cover_letter", does: "Creates a cover letter for a saved job.", writes: true, credits: "5" },
  { name: "check_match", does: "Compares an uploaded resume with a job.", writes: true, credits: "5, none for a cached result" },
  { name: "get_resume_pdf_link", does: "Exports a resume to PDF and returns a private link valid for 15 minutes.", writes: true },
];

const SECTION = "mb-8";
const HEADING = "mb-3 text-xl font-semibold";
const LIST = "ml-6 list-disc space-y-2";
const CODE_BLOCK =
  "mt-3 max-w-full overflow-x-auto rounded-md border bg-muted/50 p-3 text-xs";
const INLINE_LINK = "text-primary-ink underline";

export default function McpDocs() {
  const { t, i18n } = useTranslation();
  usePageMeta({
    title: "Connect Cursopus to Claude and other AI assistants (MCP)",
    description:
      "How to connect the Cursopus MCP server, what each of its 17 tools reads or changes, and how your data is handled.",
  });

  // The documentation is published in English only; lang tells assistive
  // technology and translators which language the body is in.
  return (
    <div lang="en" className="mx-auto max-w-3xl px-4 py-12 text-foreground">
      <nav lang={i18n.language} className="mb-8 flex items-center justify-between">
        <Link
          to="/"
          className="flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground max-sm:min-h-11"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("common.backToHome")}
        </Link>
        <Link to="/" className="flex items-center gap-2 max-sm:min-h-11">
          <LogoMark className="h-6 w-6" />
          <span className="font-bold">Cursopus</span>
        </Link>
      </nav>
      <ContentLanguageNotice contentLanguage="en" />
      <h1 className="mb-8 text-3xl font-bold">Cursopus MCP connector</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: November 19, 2026
      </p>

      <section className={SECTION}>
        <h2 className={HEADING}>1. What it does</h2>
        <p className="mb-3">
          The connector lets an AI assistant work with your Cursopus account
          over the Model Context Protocol (MCP). Your assistant can save a job
          posting, move jobs between your stages, compare a resume with a job,
          propose resume edits and draft a cover letter.
        </p>
        <p>
          It needs a Cursopus account on the Pro plan. On the Free plan the
          server answers with an upgrade message.
        </p>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>2. Connect</h2>
        <p className="mb-3">
          The server address is{" "}
          <code className="break-all text-sm">{MCP_ENDPOINT}</code>. It uses
          Streamable HTTP and OAuth.
        </p>
        <h3 className="mb-2 mt-4 font-semibold">Claude</h3>
        <p>
          In Claude, open Settings, then Connectors, choose to add a custom
          connector and paste the address above. Claude opens a Cursopus page
          where you sign in and allow access.
        </p>
        <h3 className="mb-2 mt-4 font-semibold">Claude Code</h3>
        <pre className={CODE_BLOCK}>
          <code>{CLAUDE_CODE_COMMAND}</code>
        </pre>
        <h3 className="mb-2 mt-4 font-semibold">Clients without OAuth</h3>
        <p>
          Create a personal token in Settings, under AI assistants (MCP), and
          send it as a bearer token. The token is shown once, so copy it when
          you create it.
        </p>
        <pre className={CODE_BLOCK}>
          <code>{TOKEN_CLIENT_CONFIG}</code>
        </pre>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>3. Tools</h2>
        <p className="mb-3">
          Tools marked Read only return data. Every other tool is marked as a
          write, so an assistant that follows the protocol asks you before it
          runs one. Some tools use AI credits from your plan.
        </p>
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b">
                <th scope="col" className="py-2 pr-4 font-semibold">Tool</th>
                <th scope="col" className="py-2 pr-4 font-semibold">What it does</th>
                <th scope="col" className="py-2 pr-4 font-semibold">Access</th>
                <th scope="col" className="py-2 font-semibold">AI credits</th>
              </tr>
            </thead>
            <tbody>
              {TOOLS.map((tool) => (
                <tr key={tool.name} className="border-b align-top">
                  <td className="py-2 pr-4">
                    <code>{tool.name}</code>
                  </td>
                  <td className="py-2 pr-4">{tool.does}</td>
                  <td className="py-2 pr-4">
                    {tool.writes ? "Write, asks first" : "Read only"}
                  </td>
                  <td className="py-2">{tool.credits ?? "None"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>4. What an assistant cannot do</h2>
        <ul className={LIST}>
          <li>
            Delete anything. There is no delete tool. A job can be archived and
            restored.
          </li>
          <li>
            Overwrite a resume without a way back. Editing a resume saves a
            version first, at most one every 30 minutes, and the last 20
            versions are kept. Restoring a version saves the current one too.
          </li>
          <li>
            Save tailored edits in one step. Proposing edits and saving the
            ones you chose are two separate tools, and saving creates a new
            resume.
          </li>
          <li>Edit an uploaded resume. Uploaded resumes are read only.</li>
          <li>
            Reach another account. The server takes your identity from the
            token and never from what the assistant sends.
          </li>
          <li>
            Open web pages. The server never fetches a URL; parsing works on
            text the assistant passes in.
          </li>
        </ul>
        <p className="mt-3">
          Job descriptions come from third parties. The server tells the
          assistant to treat them as data and not as instructions.
        </p>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>5. Your data</h2>
        <ul className={LIST}>
          <li>
            Through the connector an assistant can read and change your jobs,
            stages, companies, resumes and cover letters, within the tools
            listed above.
          </li>
          <li>
            Whatever a tool returns goes to the assistant you connected and is
            handled under that provider&apos;s terms.
          </li>
          <li>
            Tools that use AI credits send the relevant resume and job text to
            Anthropic, the same as in the web app. See the{" "}
            <Link to="/privacy" className={INLINE_LINK}>
              Privacy Policy
            </Link>
            .
          </li>
          <li>
            Each tool call is recorded with the tool name, whether it
            succeeded, how long it took and which token or app made it. You
            can see recent activity in Settings, under AI assistants (MCP).
          </li>
          <li>
            Only a hash of each token is stored. OAuth access tokens last one
            hour and are renewed with a refresh token that lasts 30 days.
          </li>
        </ul>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>6. Limits</h2>
        <ul className={LIST}>
          <li>60 requests per minute for each account.</li>
          <li>A request body can be up to 1 MB.</li>
          <li>
            AI credits and plan limits are the same as in the web app. A tool
            that needs more credits than you have returns an error and changes
            nothing.
          </li>
        </ul>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>7. Disconnect</h2>
        <p>
          Open Settings, then AI assistants (MCP). Disconnecting an app revokes
          all of its tokens at once, and a personal token can be revoked on the
          same page. You can also remove the connector in your assistant.
        </p>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>8. Troubleshooting</h2>
        <ul className={LIST}>
          <li>
            <strong>The assistant says authentication is required.</strong>{" "}
            The connection expired or was revoked. Connect again.
          </li>
          <li>
            <strong>The server asks you to upgrade.</strong> The account is on
            the Free plan. The connector needs Pro.
          </li>
          <li>
            <strong>Too many requests.</strong> Wait a minute and try again.
          </li>
        </ul>
      </section>

      <section className={SECTION}>
        <h2 className={HEADING}>9. Contact</h2>
        <p>
          Questions and problems:{" "}
          <a href={SUPPORT_MAILTO} className={INLINE_LINK}>
            {SUPPORT_EMAIL}
          </a>
          . The connector is covered by our{" "}
          <Link to="/terms" className={INLINE_LINK}>
            Terms of Service
          </Link>{" "}
          and{" "}
          <Link to="/privacy" className={INLINE_LINK}>
            Privacy Policy
          </Link>
          .
        </p>
      </section>
    </div>
  );
}
