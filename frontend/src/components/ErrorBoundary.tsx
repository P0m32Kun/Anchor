import { Component, ReactNode, ErrorInfo } from "react";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[ErrorBoundary] 未捕获的渲染错误:", error);
    if (info.componentStack) {
      console.error("[ErrorBoundary] 组件堆栈:", info.componentStack);
    }
  }

  private handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="min-h-[50vh] flex items-center justify-center p-8">
          <div className="max-w-lg w-full text-center space-y-4">
            <h2 className="text-lg font-semibold text-brand-danger">
              页面出现错误
            </h2>
            <p className="text-sm text-text-secondary mt-2">
              {this.state.error?.message || "未知错误"}
            </p>
            <button
              onClick={this.handleReload}
              className="mt-4 inline-flex items-center justify-center px-4 py-2 bg-brand-primary text-white rounded-apple text-sm font-medium hover:opacity-90 transition-opacity"
              type="button"
            >
              重新加载
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
