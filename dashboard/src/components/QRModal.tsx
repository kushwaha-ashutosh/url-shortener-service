import { useEffect, useState } from "react";
import { qrCodeUrl, type Link } from "../api";

interface Props {
  link: Link;
  onClose: () => void;
  onError: (message: string) => void;
}

export function QRModal({ link, onClose, onError }: Props) {
  const [downloading, setDownloading] = useState(false);
  const src = qrCodeUrl(link.code);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  async function handleDownload() {
    setDownloading(true);
    try {
      // fetch+blob rather than a plain <a download> link: the API is
      // cross-origin during dev, and browsers largely ignore the
      // download attribute for cross-origin URLs, silently navigating
      // instead of downloading. A blob URL is always same-origin to
      // this page, so download works regardless of where the API is.
      const res = await fetch(src);
      if (!res.ok) throw new Error("failed to fetch QR code");
      const blob = await res.blob();
      const blobUrl = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = blobUrl;
      a.download = `${link.code}-qr.png`;
      a.click();
      URL.revokeObjectURL(blobUrl);
    } catch {
      onError("Couldn't download the QR code");
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>{link.code}</h3>
          <button type="button" className="icon-btn" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>
        <div className="qr-image-wrap">
          <img src={src} alt={`QR code for ${link.short_url}`} width={220} height={220} />
        </div>
        <p className="qr-caption">{link.short_url.replace(/^https?:\/\//, "")}</p>
        <button type="button" className="download-btn" onClick={handleDownload} disabled={downloading}>
          {downloading ? "Downloading…" : "Download PNG"}
        </button>
      </div>
    </div>
  );
}
