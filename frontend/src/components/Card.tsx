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
          "rounded-2xl border border-white/5 bg-card/50 backdrop-blur-sm text-card-foreground shadow-xl transition-all duration-300",
          hover && "hover:bg-card/80 hover:border-primary/30 hover:shadow-primary/5 cursor-pointer hover:-translate-y-0.5",
          className
        )}
        {...props}
      >
        {/* 内置一个极其微弱的顶部光辉边框，增加高级感 */}
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent" />
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
        className={cn("text-lg font-bold leading-none tracking-tight text-foreground/90", className)} 
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
        className={cn("text-sm text-muted-foreground font-medium", className)} 
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
