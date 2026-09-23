import { useEffect, useState } from "react";
import type { ToastMessage } from "../useToasts";

export function ToastStack({ toasts }: { toasts: ToastMessage[] }) {
  return (
    <div className="toast-stack" aria-live="polite">
      {toasts.map((t) => (
        <ToastItem key={t.id} toast={t} />
      ))}
    </div>
  );
}

function ToastItem({ toast }: { toast: ToastMessage }) {
  const [leaving, setLeaving] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setLeaving(true), 2400);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className={`toast toast-${toast.tone} ${leaving ? "toast-leaving" : ""}`}>
      {toast.text}
    </div>
  );
}
