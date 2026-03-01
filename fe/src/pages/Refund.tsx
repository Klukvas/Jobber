import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Briefcase } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Refund() {
  const { t } = useTranslation();
  usePageMeta({ title: "Refund Policy — Jobber" });

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
      <h1 className="mb-8 text-3xl font-bold">Refund Policy</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: March 1, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Overview</h2>
        <p>
          All payments for Jobber Pro subscriptions are processed by{" "}
          <a
            href="https://www.paddle.com"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline"
          >
            Paddle
          </a>
          , our Merchant of Record. This refund policy outlines the conditions
          under which you may request a refund.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          2. 14-Day Money-Back Guarantee
        </h2>
        <p>
          If you are not satisfied with Jobber Pro, you may request a full
          refund within <strong>14 days</strong> of your initial purchase. No
          questions asked.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          3. How to Request a Refund
        </h2>
        <p className="mb-3">To request a refund, you can:</p>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            Email us at{" "}
            <a
              href="mailto:apavlenko.dev@gmail.com"
              className="text-primary underline"
            >
              apavlenko.dev@gmail.com
            </a>{" "}
            with your account email and the reason for the refund.
          </li>
          <li>
            Use the Paddle customer portal (accessible from Settings &gt;
            Subscription &gt; Manage Subscription) to contact billing support.
          </li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">4. After 14 Days</h2>
        <p>
          Refunds after the 14-day period are handled on a case-by-case basis.
          We may issue a prorated refund or credit at our discretion. Common
          reasons we may approve a late refund include:
        </p>
        <ul className="ml-6 mt-3 list-disc space-y-2">
          <li>Accidental renewal due to a billing issue.</li>
          <li>
            Extended service outage that significantly affected your usage.
          </li>
          <li>Duplicate charges.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">5. Cancellation</h2>
        <p>
          You can cancel your Pro subscription at any time from the Settings
          page. When you cancel:
        </p>
        <ul className="ml-6 mt-3 list-disc space-y-2">
          <li>
            You retain Pro access until the end of your current billing period.
          </li>
          <li>
            Your account automatically reverts to the Free plan after the period
            ends.
          </li>
          <li>No further charges will be made.</li>
          <li>Your data is preserved — nothing is deleted upon downgrade.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">6. Refund Processing</h2>
        <p>
          Approved refunds are processed by Paddle and typically appear in your
          account within 5&ndash;10 business days, depending on your payment
          method and financial institution.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">7. Contact</h2>
        <p>
          For any billing questions or refund requests, contact us at{" "}
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
