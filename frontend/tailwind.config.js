/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: [
          "-apple-system",
          "BlinkMacSystemFont",
          "'SF Pro Text'",
          "'SF Pro Display'",
          "'Helvetica Neue'",
          "Helvetica",
          "system-ui",
          "sans-serif",
        ],
        mono: [
          "'SF Mono'",
          "'SFMono-Regular'",
          "ui-monospace",
          "Menlo",
          "Monaco",
          "'Courier New'",
          "monospace",
        ],
      },
      colors: {
        // Dark theme surface hierarchy (Airport Dashboard)
        surface: {
          DEFAULT: "#0a1628",
          elevated: "#111d32",
          "elevated-2": "#1a2a42",
          "elevated-3": "#223555",
        },
        // Ambient light colors
        ambient: {
          blue: "rgba(100, 150, 255, 0.08)",
          purple: "rgba(180, 130, 255, 0.06)",
          glow: "rgba(255, 255, 255, 0.02)",
        },
        // Vibrant primary colors (Electric variants)
        brand: {
          primary: "#38bdf8",   // CyberOS 主色-蓝 (sky-400)
          secondary: "#7D8590",
          success: "#4ade80",   // CyberOS 成功-绿 (green-400)
          warning: "#facc15",   // CyberOS 警告-黄 (yellow-400)
          danger: "#f87171",    // CyberOS 警告-红 (red-400)
          purple: "#A371F7",    // Electric purple
        },
        // Text colors (CyberOS)
        "text-primary": "#F0F6FC",
        "text-secondary": "#e2e8f0",
        "text-tertiary": "#94a3b8",
        "text-quaternary": "#64748b",
        // Apple accent colors (preserved but vibrant)
        accent: {
          blue: "#0A84FF",
          "blue-hover": "#0063CC",
          "blue-glow": "rgba(10,132,255,0.45)",
          green: "#32D74B",
          "green-glow": "rgba(50,215,75,0.45)",
          red: "#FF453A",
          orange: "#FF9F0A",
          yellow: "#FFD60A",
          purple: "#BF5AF2",
          teal: "#5AC8FA",
        },
        // Liquid glass effects
        glass: {
          border: "rgba(255,255,255,0.1)",
          "border-light": "rgba(255,255,255,0.15)",
          "border-subtle": "rgba(255,255,255,0.06)",
          bg: "rgba(255,255,255,0.03)",
          "bg-hover": "rgba(255,255,255,0.07)",
          "bg-active": "rgba(255,255,255,0.12)",
        },
      },
      spacing: {
        "apple-xs": "4px",
        "apple-sm": "8px",
        "apple-md": "16px",
        "apple-lg": "24px",
        "apple-xl": "32px",
        "apple-2xl": "48px",
      },
      borderRadius: {
        apple: "12px",
        "apple-sm": "8px",
        "apple-md": "12px",
        "apple-lg": "16px",
        "apple-xl": "20px",
      },
      boxShadow: {
        "apple-sm": "0 1px 2px rgba(0,0,0,0.5), 0 0 1px rgba(255,255,255,0.1)",
        apple: "0 4px 12px rgba(0,0,0,0.6), 0 0 1px rgba(255,255,255,0.15)",
        "apple-md": "0 8px 24px rgba(0,0,0,0.7), 0 0 1px rgba(255,255,255,0.15)",
        "apple-lg": "0 16px 48px rgba(0,0,0,0.8), 0 0 1px rgba(255,255,255,0.2)",
        island:
          "0 0 0 1px rgba(255,255,255,0.06), 0 2px 4px rgba(0,0,0,0.3), 0 8px 24px rgba(0,0,0,0.4), 0 24px 60px rgba(0,0,0,0.5)",
        "island-hover":
          "0 0 0 1px rgba(255,255,255,0.12), 0 4px 8px rgba(0,0,0,0.35), 0 16px 32px rgba(0,0,0,0.45), 0 32px 72px rgba(0,0,0,0.55)",
        "island-glow":
          "inset 0 1px 0 rgba(255,255,255,0.08), inset 0 0 20px rgba(255,255,255,0.02)",
        "glow-blue": "0 0 20px rgba(47,129,247,0.3), 0 0 40px rgba(47,129,247,0.1)",
        "glow-green": "0 0 20px rgba(63,185,80,0.3), 0 0 40px rgba(63,185,80,0.1)",
        "glow-red": "0 0 20px rgba(248,81,73,0.3), 0 0 40px rgba(248,81,73,0.1)",
        "glow-purple": "0 0 20px rgba(163,113,247,0.3), 0 0 40px rgba(163,113,247,0.1)",
      },
      backdropBlur: {
        glass: "40px",
      },
      transitionTimingFunction: {
        apple: "cubic-bezier(0.32, 0.72, 0, 1)",
        "apple-bounce": "cubic-bezier(0.34, 1.56, 0.64, 1)",
        "apple-ease": "cubic-bezier(0.25, 0.1, 0.25, 1)",
      },
      animation: {
        "fade-in": "fadeIn 0.3s cubic-bezier(0.25, 0.1, 0.25, 1)",
        "slide-up": "slideUp 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
        "slide-down": "slideDown 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
        "scale-in": "scaleIn 0.3s cubic-bezier(0.32, 0.72, 0, 1)",
        "scale-out": "scaleOut 0.2s cubic-bezier(0.32, 0.72, 0, 1)",
        "slide-in-right": "slideInRight 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
        "glow-pulse": "glowPulse 3s ease-in-out infinite",
        shimmer: "shimmer 2s infinite linear",
        "pulse-slow": "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite",
      },
      keyframes: {
        fadeIn: {
          from: { opacity: "0" },
          to: { opacity: "1" },
        },
        slideUp: {
          from: { opacity: "0", transform: "translateY(12px) scale(0.98)" },
          to: { opacity: "1", transform: "translateY(0) scale(1)" },
        },
        slideDown: {
          from: { opacity: "0", transform: "translateY(-12px) scale(0.98)" },
          to: { opacity: "1", transform: "translateY(0) scale(1)" },
        },
        scaleIn: {
          from: { opacity: "0", transform: "scale(0.95) translateY(8px)" },
          to: { opacity: "1", transform: "scale(1) translateY(0)" },
        },
        scaleOut: {
          from: { opacity: "1", transform: "scale(1) translateY(0)" },
          to: { opacity: "0", transform: "scale(0.95) translateY(8px)" },
        },
        slideInRight: {
          from: { opacity: "0", transform: "translateX(20px)" },
          to: { opacity: "1", transform: "translateX(0)" },
        },
        glowPulse: {
          "0%, 100%": { opacity: "0.6" },
          "50%": { opacity: "1" },
        },
        shimmer: {
          "0%": { backgroundPosition: "-200% 0" },
          "100%": { backgroundPosition: "200% 0" },
        },
      },
    },
  },
  plugins: [],
};
