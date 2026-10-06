import { useEffect, useId, useRef } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

type ModalProps = {
  open: boolean;
  title: string;
  busy?: boolean;
  onClose: () => void;
  children: ReactNode;
};

// 使用模态 dialog 的焦点约束和背景隔离，外观由应用样式统一控制。
export function Modal({ open, title, busy = false, onClose, children }: ModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!open || !dialog) return;
    const trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    dialog.showModal();
    document.body.style.overflow = "hidden";
    return () => {
      dialog.close();
      document.body.style.overflow = previousOverflow;
      if (trigger?.isConnected) trigger.focus();
    };
  }, [open]);

  return createPortal(
    <dialog ref={dialogRef} className="modal" aria-labelledby={titleId} aria-busy={busy} onCancel={(event) => {
      event.preventDefault();
      if (!busy) onClose();
    }}>
      <h2 id={titleId}>{title}</h2>
      {children}
    </dialog>,
    document.body,
  );
}
