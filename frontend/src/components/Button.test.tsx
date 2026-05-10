import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Button } from "./Button";

describe("Button", () => {
  it("renders with default variant", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument();
  });

  it("is disabled when loading", () => {
    render(<Button loading>Loading</Button>);
    const btn = screen.getByRole("button", { name: "Loading" });
    expect(btn).toBeDisabled();
  });

  it("is disabled when disabled prop is true", () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole("button", { name: "Disabled" })).toBeDisabled();
  });

  it("shows spinner when loading", () => {
    render(<Button loading>Loading</Button>);
    const btn = screen.getByRole("button");
    expect(btn.querySelector("svg")).toBeInTheDocument();
  });
});
