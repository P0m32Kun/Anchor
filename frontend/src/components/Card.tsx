import React, { forwardRef } from "react";
import { cn } from "../lib/utils";

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  hover?: boolean;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ hover = false, children, className = "", ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          "rounded-xl border border-border bg-card text-card-foreground shadow-sm",
          hover && "hover:bg-accent/50 transition-colors cursor-pointer",
          className
        )}
        {...props}
      >
        {children}
      </div>
    );
  }
);

export const CardHeader = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ children, className = "", ...props }, ref) => {
    return (
      <div 
        ref={ref} 
        className={cn("flex flex-col space-y-1.5 p-6", className)} 
        {...props}
      >
        {children}
      </div>
    );
  }
);

export const CardTitle = forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ children, className = "", ...props }, ref) => {
    return (
      <h3 
        ref={ref} 
        className={cn("text-lg font-semibold leading-none tracking-tight text-foreground", className)} 
        {...props}
      >
        {children}
      </h3>
    );
  }
);

export const CardDescription = forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
  ({ children, className = "", ...props }, ref) => {
    return (
      <p 
        ref={ref} 
        className={cn("text-sm text-muted-foreground", className)} 
        {...props}
      >
        {children}
      </p>
    );
  }
);

export const CardContent = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ children, className = "", ...props }, ref) => {
    return (
      <div 
        ref={ref} 
        className={cn("p-6 pt-0", className)} 
        {...props}
      >
        {children}
      </div>
    );
  }
);

export const CardFooter = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ children, className = "", ...props }, ref) => {
    return (
      <div 
        ref={ref} 
        className={cn("flex items-center p-6 pt-0", className)} 
        {...props}
      >
        {children}
      </div>
    );
  }
);
