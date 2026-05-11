import { Component, ReactNode, ErrorInfo } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, ChevronDown, ChevronUp, RotateCcw, Home } from "lucide-react";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
  showDetails: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, showDetails: false };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error, showDetails: false };
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

  private toggleDetails = () => {
    this.setState((prev) => ({ showDetails: !prev.showDetails }));
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="min-h-[50vh] flex items-center justify-center p-8">
          <div className="max-w-lg w-full text-center space-y-4">
            <div className="mx-auto w-12 h-12 rounded-full bg-accent-red/10 flex items-center justify-center">
              <AlertTriangle className="h-6 w-6 text-accent-red" />
            </div>
            <h2 className="text-lg font-semibold text-foreground">
              页面出现错误
            </h2>
            <p className="text-sm text-muted-foreground">
              页面渲染时遇到了问题，请尝试刷新或返回首页。
            </p>
            <div className="flex items-center justify-center gap-3">
              <button
                onClick={this.handleReload}
                className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:opacity-90 transition-opacity"
                type="button"
              >
                <RotateCcw className="h-4 w-4" />
                重新加载
              </button>
              <Link
                to="/"
                className="inline-flex items-center gap-2 px-4 py-2 bg-muted text-muted-foreground rounded-lg text-sm font-medium hover:bg-muted/80 transition-colors"
              >
                <Home className="h-4 w-4" />
                回到首页
              </Link>
            </div>
            {this.state.error && (
              <div className="text-left">
                <button
                  onClick={this.toggleDetails}
                  className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  type="button"
                >
                  {this.state.showDetails ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                  {this.state.showDetails ? "收起详情" : "查看详情"}
                </button>
                {this.state.showDetails && (
                  <pre className="mt-2 p-3 bg-muted/50 rounded-lg border text-xs text-muted-foreground overflow-auto max-h-40 text-left">
                    {this.state.error.message}
                    {this.state.error.stack && `\n\n${this.state.error.stack}`}
                  </pre>
                )}
              </div>
            )}
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
