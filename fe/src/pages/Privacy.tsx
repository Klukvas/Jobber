import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Privacy() {
  usePageMeta({ titleKey: "privacy.title" });

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 text-foreground">
      <h1 className="mb-8 text-3xl font-bold">Privacy Policy</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: February 28, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Overview</h2>
        <p>
          Jobber (&quot;we&quot;, &quot;our&quot;, &quot;us&quot;) is a job
          application tracking service available at jobber-app.com and as a
          Chrome browser extension. This policy describes how we collect, use,
          and protect your data.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">2. Data We Collect</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>Account information:</strong> email address, name, and
            hashed password when you register.
          </li>
          <li>
            <strong>Job data:</strong> job titles, company names, URLs, notes,
            and application details you save to your account.
          </li>
          <li>
            <strong>Page content (extension only):</strong> when you click
            &quot;Import This Job&quot;, the text content of the current web page
            is sent to our server for AI-powered parsing. We do not store the raw
            page text after parsing is complete.
          </li>
          <li>
            <strong>Authentication tokens:</strong> JWT tokens are stored locally
            in your browser to keep you signed in.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">3. How We Use Your Data</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>To provide and maintain the job tracking service.</li>
          <li>
            To parse job postings using AI (Anthropic Claude) when you use the
            browser extension.
          </li>
          <li>To authenticate you and secure your account.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">4. Data Sharing</h2>
        <p>
          We do not sell, trade, or transfer your personal data to third parties.
          Page text is sent to Anthropic&apos;s API solely for job parsing and is
          subject to{" "}
          <a
            href="https://www.anthropic.com/privacy"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline"
          >
            Anthropic&apos;s Privacy Policy
          </a>
          .
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">5. Data Storage</h2>
        <p>
          Your data is stored on secure servers in the EU (Hetzner, Germany).
          Passwords are hashed using bcrypt. All connections use HTTPS
          encryption.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          6. Browser Extension Permissions
        </h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>activeTab:</strong> read the current page when you click the
            extension.
          </li>
          <li>
            <strong>storage:</strong> store authentication tokens locally.
          </li>
          <li>
            <strong>scripting:</strong> extract page text for job parsing.
          </li>
          <li>
            <strong>alarms:</strong> refresh authentication tokens periodically.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">7. Your Rights</h2>
        <p>
          You can delete your account and all associated data at any time from
          the Settings page. To request data export or have questions, contact us
          at{" "}
          <a
            href="mailto:apavlenko.dev@gmail.com"
            className="text-primary underline"
          >
            apavlenko.dev@gmail.com
          </a>
          .
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">8. Changes</h2>
        <p>
          We may update this policy from time to time. Changes will be posted on
          this page with an updated date.
        </p>
      </section>
    </div>
  );
}
