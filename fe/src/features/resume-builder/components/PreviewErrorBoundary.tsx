import { Component } from "react";
import type { ReactNode, ErrorInfo } from "react";
import i18next from "i18next";
import * as Sentry from "@sentry/react";
import { AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/shared/ui/Button";

interface Props {
  readonly children: ReactNode;
}

interface State {
  hasError: boolean;
}

export class PreviewErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    Sentry.captureException(error, {
      extra: { componentStack: info.componentStack },
    });
  }

  private handleReset = () => {
    this.setState({ hasError: false });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-full flex-col items-center justify-center gap-4 p-8 text-center">
          <AlertCircle className="h-12 w-12 text-destructive" />
          <p className="text-sm text-muted-foreground">
            {i18next.t("resumeBuilder.preview.renderError")}
          </p>
          <Button variant="outline" size="sm" onClick={this.handleReset}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {i18next.t("common.tryAgain")}
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}
