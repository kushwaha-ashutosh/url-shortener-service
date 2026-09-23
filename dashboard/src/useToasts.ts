import { useState } from "react";

export interface ToastMessage {
  id: number;
  text: string;
  tone: "success" | "error";
}

let nextId = 1;

export function useToasts() {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  function push(text: string, tone: ToastMessage["tone"] = "success") {
    const id = nextId++;
    setToasts((prev) => [...prev, { id, text, tone }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 2800);
  }

  return { toasts, push };
}
