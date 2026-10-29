import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { LogoMark } from "@/shared/ui/Logo";
import { SUPPORT_EMAIL, SUPPORT_MAILTO } from "@/shared/lib/supportContact";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Terms() {
  const { t } = useTranslation();
  usePageMeta({ title: "Terms of Service — Cursopus" });

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
      <h1 className="mb-8 text-3xl font-bold">Terms of Service</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: October 25, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Acceptance of Terms</h2>
        <p>
          By accessing or using Cursopus (&quot;Service&quot;), available at
          cursopus.com and as a Chrome browser extension, you agree to be
          bound by these Terms of Service. If you do not agree, please do not
          use the Service.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          2. Description of Service
        </h2>
        <p>
          Cursopus is a personal job application tracking platform that helps users
          organize their job search. The Service includes a web application and
          a Chrome browser extension for importing job postings. We offer Free and Pro plans, with AI tools for resume tailoring, cover letters,
          resume matching and interview preparation. Cursopus does not publish job listings for employers,
          recruit candidates, or guarantee interviews or employment.
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
            <strong>Free Plan:</strong> includes 25 tracked jobs, 1 uploaded resume, 2 builder resumes,
            1 cover letter and 30 shared AI credits once per account. Free credits do not reset monthly.
          </li>
          <li>
            <strong>Pro Plan:</strong> $24 USD per month, with 100 tracked jobs, 10 uploaded resumes,
            10 builder resumes, 10 cover letters in total and 500 AI credits per confirmed paid billing period.
            Unused Pro credits expire at the end of that period and do not roll over.
            Taxes and the final amount payable are shown at checkout.
          </li>
          <li>
            Payments are processed by{" "}
            <a
              href="https://www.creem.io"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Creem
            </a>
            , our Merchant of Record. Creem processes payment, applicable transaction taxes and billing.
            Pro access is activated after payment confirmation.
          </li>
          <li>
            By subscribing to a paid plan, you agree to Creem&apos;s{" "}
            <a
              href="https://www.creem.io/buyer-terms"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              Buyer Terms of Service
            </a>{" "}
            and{" "}
            <a
              href="https://www.creem.io/privacy"
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
            cancelled before the renewal date. Cancel in Settings &gt; Subscription &gt;
            Manage Subscription, or open the Creem customer portal from the same dialog.
            Cancellation stops future renewals; access continues until the paid period ends.
          </li>
          <li>
            AI credits are shared across AI features. The credit cost is shown before each AI start.
            Credits are reserved before processing and charged once AI processing starts, including
            when a later error occurs. Checks that fail before AI starts return the reservation.
            New operations use new credits; retrieving an existing cached result does not.
            Credit, input-size, provider availability and safety spending limits apply, so AI may
            be unavailable even while credits remain.
          </li>
          <li>
            Our <Link to="/refund" className="text-primary underline">Refund Policy</Link> provides
            a full refund on request within 14 days of the initial purchase, without a reason requirement.
            Mandatory consumer rights remain unaffected.
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
          Cursopus. You retain ownership of any data you upload. By using the
          Service, you grant us a limited license to process your data solely to
          provide the Service to you.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">7. AI-Powered Features</h2>
        <p>
          Certain features use AI (powered by Anthropic Claude) to parse job postings and resumes, suggest or tailor resume content, draft cover letters,
          analyze resume-job matching and prepare interview practice materials. Relevant content
          is sent to Anthropic when you request these features. AI results are provided
          &quot;as is&quot; and should not be considered professional career
          advice. Review generated content for accuracy before using it. Do not add qualifications
          or experience you do not have. AI scores are estimates and do not predict an
          employer&apos;s decision or guarantee that an applicant tracking system will accept a resume.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">8. Termination</h2>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            To request account closure or deletion, contact the support email below.
            Cancel any paid subscription separately through Settings or the Creem customer portal.
          </li>
          <li>
            We reserve the right to suspend or terminate accounts that violate
            these terms.
          </li>
          <li>
            Data handling and deletion requests are described in our <Link to="/privacy" className="text-primary underline">Privacy Policy</Link>.
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
          To the maximum extent permitted by law, Cursopus shall not be liable for
          any indirect, incidental, special, consequential, or punitive damages,
          including loss of data, profits, or business opportunities arising
          from your use of the Service. These terms do not limit liability or consumer rights
          that cannot be excluded under applicable law.
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
            href={SUPPORT_MAILTO}
            className="text-primary underline"
          >
            {SUPPORT_EMAIL}
          </a>
          . We aim to respond within 3 business days.
        </p>
      </section>
    </div>
  );
}
