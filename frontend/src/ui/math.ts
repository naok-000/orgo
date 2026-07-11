// Client-side LaTeX rendering: org's LaTeX fragments ($...$, $$...$$,
// \(...\), \[...\], \begin{equation}...\end{equation} and friends) pass
// through internal/render untouched (HTML-escaped verbatim text) — go-org
// deliberately doesn't turn them into images or MathML. KaTeX's auto-render
// extension walks a rendered note body and typesets those fragments in
// place.
import renderMathInElement from "katex/contrib/auto-render";
import "katex/dist/katex.min.css";
import "../styles/math.css";

// KaTeX's documented default delimiter list (in its documented order), with
// single-`$` inline math inserted right after `$$`: `$$` must stay ahead of
// `$` in the list so the auto-render matcher prefers the longer delimiter
// when both could start a match at the same position.
const delimiters = [
  { left: "$$", right: "$$", display: true },
  { left: "$", right: "$", display: false },
  { left: "\\(", right: "\\)", display: false },
  { left: "\\begin{equation}", right: "\\end{equation}", display: true },
  { left: "\\begin{align}", right: "\\end{align}", display: true },
  { left: "\\begin{alignat}", right: "\\end{alignat}", display: true },
  { left: "\\begin{gather}", right: "\\end{gather}", display: true },
  { left: "\\begin{CD}", right: "\\end{CD}", display: true },
  { left: "\\[", right: "\\]", display: true },
];

// renderMathIn typesets LaTeX fragments found in el's text content in
// place. Safe to call on any note body; elements without math are
// untouched.
export function renderMathIn(el: HTMLElement): void {
  renderMathInElement(el, {
    delimiters,
    throwOnError: false,
    trust: false,
  });
}
