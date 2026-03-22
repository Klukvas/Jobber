import { Component } from "react";
import type { ReactNode, ErrorInfo } from "react";
import * as Sentry from "@sentry/react";
import i18n from "@/shared/lib/i18n";

interface Props {
  readonly children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class GlobalErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    Sentry.captureException(error, {
      extra: { componentStack: info.componentStack },
    });
  }

  private handleReload = () => {
    window.location.reload();
  };

  private handleGoHome = () => {
    window.location.href = "/";
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4 text-foreground">
          <div className="flex flex-col items-center text-center">
            <div className="mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-muted">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="40"
                height="40"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="text-muted-foreground"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" x2="12" y1="8" y2="12" />
                <line x1="12" x2="12.01" y1="16" y2="16" />
              </svg>
            </div>

            <h1 className="mb-2 text-2xl font-semibold">
              {i18n.t("errors.somethingWentWrong")}
            </h1>

            <p className="mb-8 max-w-md text-muted-foreground">
              {i18n.t("errors.unexpectedError")}
            </p>

            <div className="flex gap-3">
              <button
                onClick={this.handleReload}
                className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
              >
                {i18n.t("errors.reloadPage")}
              </button>
              <button
                onClick={this.handleGoHome}
                className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
              >
                {i18n.t("common.backToHome")}
              </button>
            </div>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
