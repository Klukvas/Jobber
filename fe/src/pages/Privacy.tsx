import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { LogoMark } from "@/shared/ui/Logo";
import { SUPPORT_EMAIL, SUPPORT_MAILTO } from "@/shared/lib/supportContact";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Privacy() {
  const { t } = useTranslation();
  usePageMeta({ title: "Privacy Policy — Cursopus" });

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
          <LogoMark className="h-6 w-6" />
          <span className="font-bold">Cursopus</span>
        </Link>
      </nav>
      <h1 className="mb-8 text-3xl font-bold">Privacy Policy</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: October 25, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Overview</h2>
        <p>
          Cursopus (&quot;we&quot;, &quot;our&quot;, &quot;us&quot;) is a personal job
          application tracking service available at cursopus.com and as a
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
            page is sent to our server for AI-powered parsing. Relevant page text is also sent to Anthropic for AI parsing when that path is used.
          </li>
          <li>
            <strong>Resume and AI content:</strong> uploaded resume files and links, builder resume
            sections, cover letters, job descriptions, interview practice answers and AI results
            and operation records created when you use these features.
          </li>
          <li>
            <strong>Authentication and preferences:</strong> authentication cookies and locally
            stored extension tokens, language and theme preferences, and cookie consent choices.
          </li>
          <li>
            <strong>Billing and support:</strong> subscription and payment identifiers, billing
            status, and messages you send to support.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">3. How We Use Your Data</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>To provide and maintain the job tracking service.</li>
          <li>
            To run the AI features you request: job and resume parsing, resume tailoring and suggestions,
            cover letter generation, resume matching and interview preparation. Relevant resume,
            job, letter or interview content is sent to Anthropic for those operations.
          </li>
          <li>To authenticate you and secure your account.</li>
          <li>To manage subscriptions and credit usage, respond to support requests and diagnose errors.</li>
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
            <strong>Anthropic</strong> — relevant page text, resume content, job details, cover letters and interview practice content
            are sent to Anthropic&apos;s API to perform the AI features you request.
            Processing and retention by Anthropic are subject to{" "}
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
            <strong>Creem</strong> — payments for paid plans are processed by
            Creem, our Merchant of Record. When you subscribe, Creem collects
            and processes your billing information under{" "}
            <a
              href="https://www.creem.io/privacy"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Creem&apos;s Privacy Policy
            </a>
            . We never see or store your full payment card details.
          </li>
          <li>
            <strong>Google Analytics and PostHog</strong> — when configured, optional usage
            analytics load only if you accept analytics cookies. These may include page URLs,
            interactions and account identifiers (see Cookies below).
          </li>
          <li>
            <strong>Sentry</strong> — when configured, error monitoring. When something breaks,
            an error report is sent to Sentry so we can fix it; reports may
            include your account ID and email address and, for errors in the web
            app, a screen recording of the failing session with text content
            masked.
          </li>
          <li>
            <strong>Infrastructure and connected services</strong> — hosting, database,
            file storage and email providers process data needed to operate Cursopus.
            Providers may process data outside your country.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">5. Data Storage</h2>
        <p>
          Account and saved product data are stored in our database; uploaded files use object
          storage. Passwords are hashed with bcrypt and the public service uses HTTPS.
          Saved documents and AI results remain associated with your account until removed.
          Cancelling Pro does not delete your documents. For account deletion and retention
          questions, contact us below. Some billing, security and operational records may need
          to be retained for legal obligations or service security; provider retention is governed
          by the applicable provider terms. We do not promise immediate deletion from every
          provider or backup.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">6. Cookies</h2>
        <p className="mb-3">We use two kinds of cookies:</p>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>Essential cookies</strong> — required for the service to
            work: secure, httpOnly authentication cookies that keep you signed
            in. Checkout runs on Creem&apos;s own hosted checkout page, which
            sets its own cookies for payment processing and fraud prevention
            while you are on that page. These cannot be switched off.
          </li>
          <li>
            <strong>Analytics cookies (optional)</strong> — set only after you
            accept them in the cookie banner: when configured, Google Analytics and PostHog help
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
          You can edit or remove saved items using the product controls available for those items.
          To request account deletion, data access, correction or export, or ask about
          applicable privacy rights, contact us at{" "}
          <a
            href={SUPPORT_MAILTO}
            className="text-primary underline"
          >
            {SUPPORT_EMAIL}
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
