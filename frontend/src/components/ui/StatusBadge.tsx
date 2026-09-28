export function StatusBadge({ children, tone = "success" }: { children: string; tone?: "success" | "processing" | "error" }) {
  return <span className={`status status--${tone}`}>{children}</span>;
}
