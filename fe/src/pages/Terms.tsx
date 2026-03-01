import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Briefcase } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Terms() {
  const { t } = useTranslation();
  usePageMeta({ title: "Terms of Service — Jobber" });

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
      <h1 className="mb-8 text-3xl font-bold">Terms of Service</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: March 1, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Acceptance of Terms</h2>
        <p>
          By accessing or using Jobber (&quot;Service&quot;), available at
          jobber-app.com and as a Chrome browser extension, you agree to be
          bound by these Terms of Service. If you do not agree, please do not
          use the Service.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          2. Description of Service
        </h2>
        <p>
          Jobber is a job application tracking platform that helps users
          organize their job search. The Service includes a web application and
          a Chrome browser extension for importing job postings. We offer both
          free and paid subscription plans.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">3. Account Registration</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            You must provide accurate and complete information when creating an
            account.
          </li>
          <li>
            You are responsible for maintaining the security of your account
            credentials.
          </li>
          <li>You must be at least 16 years old to use the Service.</li>
          <li>One person may not maintain more than one account.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          4. Subscriptions and Payments
        </h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            <strong>Free Plan:</strong> provides limited access to core features
            with usage caps on jobs, resumes, and applications.
          </li>
          <li>
            <strong>Pro Plan:</strong> a paid monthly subscription providing
            unlimited access to all features, including AI-powered tools.
          </li>
          <li>
            Payments are processed by{" "}
            <a
              href="https://www.paddle.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Paddle
            </a>
            , our Merchant of Record. Paddle handles all billing, taxes, and
            compliance on our behalf.
          </li>
          <li>
            By subscribing to a paid plan, you agree to Paddle&apos;s{" "}
            <a
              href="https://www.paddle.com/legal/terms"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Terms of Service
            </a>{" "}
            and{" "}
            <a
              href="https://www.paddle.com/legal/privacy"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Privacy Policy
            </a>
            .
          </li>
          <li>
            Subscriptions automatically renew each billing period unless
            cancelled before the renewal date.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">5. Acceptable Use</h2>
        <p className="mb-3">You agree not to:</p>
        <ul className="ml-6 list-disc space-y-2">
          <li>Use the Service for any unlawful purpose.</li>
          <li>
            Attempt to gain unauthorized access to our systems or other
            users&apos; accounts.
          </li>
          <li>Interfere with or disrupt the Service or its infrastructure.</li>
          <li>
            Upload malicious content, spam, or automated scraping beyond normal
            usage.
          </li>
          <li>
            Resell or redistribute the Service without our written consent.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">6. Intellectual Property</h2>
        <p>
          The Service, including its design, code, and branding, is owned by
          Jobber. You retain ownership of any data you upload. By using the
          Service, you grant us a limited license to process your data solely to
          provide the Service to you.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">7. AI-Powered Features</h2>
        <p>
          Certain features use AI (powered by Anthropic Claude) to parse job
          postings and analyze resume-job matching. AI results are provided
          &quot;as is&quot; and should not be considered professional career
          advice. We do not guarantee the accuracy of AI-generated content.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">8. Termination</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            You may delete your account at any time from the Settings page.
          </li>
          <li>
            We reserve the right to suspend or terminate accounts that violate
            these terms.
          </li>
          <li>
            Upon termination, your data will be permanently deleted within 30
            days.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          9. Disclaimers and Limitation of Liability
        </h2>
        <p className="mb-3">
          The Service is provided &quot;as is&quot; and &quot;as available&quot;
          without warranties of any kind, either express or implied.
        </p>
        <p>
          To the maximum extent permitted by law, Jobber shall not be liable for
          any indirect, incidental, special, consequential, or punitive damages,
          including loss of data, profits, or business opportunities arising
          from your use of the Service.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">10. Changes to Terms</h2>
        <p>
          We may update these terms from time to time. We will notify registered
          users of material changes via email or an in-app notice. Continued use
          of the Service after changes constitutes acceptance of the updated
          terms.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">11. Contact</h2>
        <p>
          If you have questions about these terms, contact us at{" "}
          <a
            href="mailto:apavlenko.dev@gmail.com"
            className="text-primary underline"
          >
            apavlenko.dev@gmail.com
          </a>
          .
        </p>
      </section>
    </div>
  );
}
