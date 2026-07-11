// Minimal DOM-builder helper so the app can stay vanilla TS without pulling
// in a UI framework. Deliberately tiny.

type Props = Partial<{
  className: string;
  text: string;
  html: string;
  attrs: Record<string, string>;
  on: Record<string, EventListenerOrEventListenerObject>;
}>;

type Child = Node | string | null | undefined | false;

export function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  props: Props = {},
  children: Child[] = [],
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (props.className) node.className = props.className;
  if (props.text !== undefined) node.textContent = props.text;
  if (props.html !== undefined) node.innerHTML = props.html;
  if (props.attrs) {
    for (const [k, v] of Object.entries(props.attrs)) node.setAttribute(k, v);
  }
  if (props.on) {
    for (const [ev, fn] of Object.entries(props.on)) node.addEventListener(ev, fn);
  }
  for (const c of children) {
    if (c == null || c === false) continue;
    node.append(typeof c === "string" ? document.createTextNode(c) : c);
  }
  return node;
}

export function clear(node: Element): void {
  node.replaceChildren();
}
