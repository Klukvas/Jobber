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

  render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            height: "100vh",
            gap: "1rem",
            fontFamily: "system-ui, sans-serif",
          }}
        >
          <h1 style={{ fontSize: "1.5rem", fontWeight: 600 }}>
            {i18n.t("errors.somethingWentWrong")}
          </h1>
          <p style={{ color: "#666" }}>{i18n.t("errors.unexpectedError")}</p>
          <button
            onClick={this.handleReload}
            style={{
              padding: "0.5rem 1.5rem",
              borderRadius: "0.375rem",
              border: "1px solid #d1d5db",
              background: "#fff",
              cursor: "pointer",
              fontSize: "0.875rem",
            }}
          >
            {i18n.t("errors.reloadPage")}
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
