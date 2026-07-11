import { describe, expect, it } from "vitest";
import { renderMathIn } from "./math.ts";

function renderInto(html: string): HTMLDivElement {
  const el = document.createElement("div");
  el.innerHTML = html;
  renderMathIn(el);
  return el;
}

describe("renderMathIn", () => {
  it("renders single-$ inline math", () => {
    const el = renderInto("<p>Energy is $E = mc^2$ per Einstein.</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.querySelectorAll(".katex-display").length).toBe(0);
  });

  it("renders $$...$$ as display math", () => {
    const el = renderInto("<p>$$E = mc^2$$</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.querySelectorAll(".katex-display").length).toBe(1);
  });

  it("renders \\(...\\) as inline math", () => {
    const el = renderInto("<p>Energy is \\(E = mc^2\\) per Einstein.</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.querySelectorAll(".katex-display").length).toBe(0);
  });

  it("renders \\[...\\] as display math", () => {
    const el = renderInto("<p>\\[E = mc^2\\]</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.querySelectorAll(".katex-display").length).toBe(1);
  });

  it("renders a \\begin{equation}...\\end{equation} block", () => {
    const el = renderInto("<p>\\begin{equation}E = mc^2\\end{equation}</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.querySelectorAll(".katex-display").length).toBe(1);
  });

  it("leaves escaped currency untouched", () => {
    const el = renderInto("<p>That'll be \\$5, please.</p>");
    expect(el.querySelectorAll(".katex").length).toBe(0);
    expect(el.textContent).toContain("$5");
  });

  it("leaves plain text without delimiters unchanged", () => {
    const el = renderInto("<p>Nothing mathy here, just plain text.</p>");
    expect(el.querySelectorAll(".katex").length).toBe(0);
    expect(el.textContent).toBe("Nothing mathy here, just plain text.");
  });

  it("does not render math inside <pre> or <code>", () => {
    const el = renderInto("<pre><code>const price = \"$5\"; // $x^2$</code></pre>");
    expect(el.querySelectorAll(".katex").length).toBe(0);
    expect(el.textContent).toBe('const price = "$5"; // $x^2$');
  });

  it("renders math while leaving surrounding text intact", () => {
    const el = renderInto("<p>Before $x$ after.</p>");
    expect(el.querySelectorAll(".katex").length).toBe(1);
    expect(el.textContent).toContain("Before");
    expect(el.textContent).toContain("after.");
  });
});
