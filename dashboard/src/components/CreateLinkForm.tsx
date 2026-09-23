import { useState } from "react";
import { ApiError, createLink, type Link } from "../api";

interface Props {
  onCreated: (link: Link) => void;
  onError: (message: string) => void;
}

export function CreateLinkForm({ onCreated, onError }: Props) {
  const [url, setUrl] = useState("");
  const [customCode, setCustomCode] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFieldError(null);
    setSubmitting(true);
    try {
      const link = await createLink(url, customCode);
      onCreated(link);
      setUrl("");
      setCustomCode("");
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Could not reach the API — is it running?";
      setFieldError(message);
      onError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="create-form" onSubmit={handleSubmit}>
      <div className="field-row">
        <div className="field-group url-field">
          <input
            type="text"
            placeholder="https://example.com/some/long/path"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            aria-label="URL to shorten"
            required
          />
        </div>
        <div className="field-group">
          <input
            type="text"
            placeholder="custom-code (optional)"
            value={customCode}
            onChange={(e) => setCustomCode(e.target.value)}
            aria-label="Custom short code"
            className="custom-code-input"
          />
        </div>
        <button type="submit" disabled={submitting}>
          {submitting ? (
            <span className="spinner" />
          ) : (
            <>
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.6">
                <path d="M8 3v10M3 8h10" strokeLinecap="round" />
              </svg>
              Shorten
            </>
          )}
        </button>
      </div>
      <p className="field-hint">
        Custom code: 3–32 characters, letters/numbers/hyphens/underscores. Leave blank for a
        random one.
      </p>
      {fieldError && <p className="error-text">{fieldError}</p>}
    </form>
  );
}
