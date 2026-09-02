import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Briefcase } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Privacy() {
  const { t } = useTranslation();
  usePageMeta({ title: "Privacy Policy — Jobber" });

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 text-foreground">
      <nav className="mb-8 flex items-center justify-between">
        <Link
          to="/"
          className="flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("common.backToHome")}
        </Link>
        <Link to="/" className="flex items-center gap-2">
          <Briefcase className="h-5 w-5 text-primary" />
          <span className="font-bold">Jobber</span>
        </Link>
      </nav>
      <h1 className="mb-8 text-3xl font-bold">Privacy Policy</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: September 3, 2026
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
            &quot;Import This Job&quot;, the text content of the current web
            page is sent to our server for AI-powered parsing. We do not store
            the raw page text after parsing is complete.
          </li>
          <li>
            <strong>Authentication tokens:</strong> JWT tokens are stored
            locally in your browser to keep you signed in.
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
        <p className="mb-3">
          We do not sell your personal data. We share limited data only with the
          service providers below, and only to operate the product:
        </p>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>Anthropic</strong> — page text you import and resume content
            you extract are sent to Anthropic&apos;s API solely for AI parsing,
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
          </li>
          <li>
            <strong>Paddle</strong> — payments for paid plans are processed by
            Paddle, our Merchant of Record. When you subscribe, Paddle collects
            and processes your billing information under{" "}
            <a
              href="https://www.paddle.com/legal/privacy"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Paddle&apos;s Privacy Policy
            </a>
            . We never see or store your full payment card details.
          </li>
          <li>
            <strong>Google Analytics and PostHog</strong> — optional usage
            analytics, enabled only if you accept analytics cookies (see Cookies
            below).
          </li>
          <li>
            <strong>Sentry</strong> — error monitoring. When something breaks,
            an error report is sent to Sentry so we can fix it; reports may
            include your account identifier and, for errors in the web app, a
            screen recording of the failing session with text content masked.
          </li>
        </ul>
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
        <h2 className="mb-3 text-xl font-semibold">6. Cookies</h2>
        <p className="mb-3">We use two kinds of cookies:</p>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>Essential cookies</strong> — required for the service to
            work: secure, httpOnly authentication cookies that keep you signed
            in, and cookies set by Paddle to enable secure checkout and fraud
            prevention. These cannot be switched off.
          </li>
          <li>
            <strong>Analytics cookies (optional)</strong> — set only after you
            accept them in the cookie banner: Google Analytics and PostHog help
            us understand how the product is used so we can improve it. If you
            choose &quot;Essential only&quot;, no analytics cookies are set.
          </li>
        </ul>
        <p className="mt-3">
          You can change your choice at any time via the &quot;Cookie
          settings&quot; link in the footer — the banner will reappear and your
          new choice takes effect immediately. (Clearing this site&apos;s data
          in your browser also resets the choice, but signs you out too.)
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          7. Browser Extension Permissions
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
        <h2 className="mb-3 text-xl font-semibold">8. Your Rights</h2>
        <p>
          You can delete your account and all associated data at any time from
          the Settings page. To request data export or have questions, contact
          us at{" "}
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
        <h2 className="mb-3 text-xl font-semibold">9. Changes</h2>
        <p>
          We may update this policy from time to time. Changes will be posted on
          this page with an updated date.
        </p>
      </section>
    </div>
  );
}
