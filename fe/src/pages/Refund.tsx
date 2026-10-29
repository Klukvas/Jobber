import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { LogoMark } from "@/shared/ui/Logo";
import { SUPPORT_EMAIL, SUPPORT_MAILTO } from "@/shared/lib/supportContact";
import { usePageMeta } from "@/shared/lib/usePageMeta";

export default function Refund() {
  const { t } = useTranslation();
  usePageMeta({ title: "Refund Policy — Cursopus" });

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
      <h1 className="mb-8 text-3xl font-bold">Refund Policy</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Last updated: October 25, 2026
      </p>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">1. Overview</h2>
        <p>
          All payments for Cursopus Pro subscriptions are processed by{" "}
          <a
            href="https://www.creem.io"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline"
          >
            Creem
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
          If you are not satisfied with Cursopus Pro, you may request a full
          refund within <strong>14 days</strong> of your initial purchase. No
          questions asked.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">
          3. How to Request a Refund
        </h2>
        <p className="mb-3">To request a refund:</p>
        <ul className="ml-6 list-disc space-y-2">
          <li>
            Email us at{" "}
            <a
              href={SUPPORT_MAILTO}
              className="text-primary underline"
            >
              {SUPPORT_EMAIL}
            </a>{" "}
            with your account email and purchase reference, if available. No reason is required
            for a request within 14 days of the initial purchase.
          </li>
          <li>
            You can also access Creem billing support through its customer portal, opened from
            Settings &gt; Subscription &gt; Manage Subscription.
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
          You can cancel in Settings &gt; Subscription &gt; Manage Subscription,
          or open the Creem customer portal from that dialog. Cancel before the renewal
          date to stop the next charge. When cancellation is confirmed:
        </p>
        <ul className="ml-6 mt-3 list-disc space-y-2">
          <li>
            You retain Pro access until the end of your current billing period.
          </li>
          <li>
            Your account automatically reverts to the Free plan after the period
            ends.
          </li>
          <li>No future subscription renewals will be charged. Cancellation itself does not refund an earlier charge.</li>
          <li>Downgrading does not delete saved documents. Free resource limits apply when adding new items, and unused Pro credits expire at the end of the paid period.</li>
        </ul>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">6. Refund Processing</h2>
        <p>
          Approved refunds are processed by Creem to the original payment method.
          The time for funds to appear depends on Creem, your payment method and your
          financial institution. This policy does not limit mandatory consumer rights,
          including any applicable withdrawal or statutory refund rights.
        </p>
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-xl font-semibold">7. Contact</h2>
        <p>
          For any billing questions or refund requests, contact us at{" "}
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
