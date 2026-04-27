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
        // Dark theme surface hierarchy (Electric Dark)
        surface: {
          DEFAULT: "#0B0E14", // Deep dark slate with a hint of navy
          elevated: "#161B22",
          "elevated-2": "#21262D",
          "elevated-3": "#30363D",
        },
        // Vibrant primary colors (Electric variants)
        brand: {
          primary: "#2F81F7",   // More vibrant electric blue
          secondary: "#7D8590",
          success: "#3FB950",   // Vibrant green
          warning: "#D29922",   // Vibrant gold/orange
          danger: "#F85149",    // Vibrant red
          purple: "#A371F7",    // Electric purple
        },
        // Text colors for dark mode
        "text-primary": "#F0F6FC",
        "text-secondary": "#8B949E",
        "text-tertiary": "#6E7681",
        "text-quaternary": "#484F58",
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
      borderRadius: {
        apple: "12px",
        "apple-sm": "8px",
        "apple-md": "12px",
        "apple-lg": "16px",
        "apple-xl": "20px",
      },
      boxShadow: {
        // Deep shadows with slight tint
        "apple-sm": "0 1px 2px rgba(0,0,0,0.5), 0 0 1px rgba(255,255,255,0.1)",
        apple: "0 4px 12px rgba(0,0,0,0.6), 0 0 1px rgba(255,255,255,0.15)",
        "apple-md": "0 8px 24px rgba(0,0,0,0.7), 0 0 1px rgba(255,255,255,0.15)",
        "apple-lg": "0 16px 48px rgba(0,0,0,0.8), 0 0 1px rgba(255,255,255,0.2)",
        // Glow shadows for vibrancy
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
      },
      animation: {
        "fade-in": "fadeIn 0.25s ease-out",
        "slide-up": "slideUp 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
        "slide-down": "slideDown 0.35s cubic-bezier(0.32, 0.72, 0, 1)",
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
        shimmer: {
          "0%": { backgroundPosition: "-200% 0" },
          "100%": { backgroundPosition: "200% 0" },
        },
      },
    },
  },
  plugins: [],
};
