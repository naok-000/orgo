// Minimal ambient module declaration for katex/contrib/auto-render: the
// katex package ships types for its core API (types/katex.d.ts) but the
// auto-render contrib script has no "types" export condition, so TypeScript
// can't resolve it on its own. This covers just the surface orgo uses.
declare module "katex/contrib/auto-render" {
  import type { KatexOptions } from "katex";

  interface Delimiter {
    left: string;
    right: string;
    display: boolean;
  }

  interface AutoRenderOptions extends KatexOptions {
    delimiters?: Delimiter[];
    ignoredTags?: string[];
    ignoredClasses?: string[];
    errorCallback?: (msg: string, err: unknown) => void;
    preProcess?: (math: string) => string;
  }

  export default function renderMathInElement(element: HTMLElement, options?: AutoRenderOptions): void;
}
