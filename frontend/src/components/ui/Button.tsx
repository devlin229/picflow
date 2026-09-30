import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  children: ReactNode;
};

export function Button({ variant = "primary", className = "", children, onClick, ...props }: ButtonProps) {
  return (
    <button className={`button button--${variant} ${className}`} onClick={onClick} {...props}>
      {children}
    </button>
  );
}
