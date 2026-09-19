import { useState } from "react";
import { ApiError, createLink, type Link } from "../api";

interface Props {
  onCreated: (link: Link) => void;
}

export function CreateLinkForm({ onCreated }: Props) {
  const [url, setUrl] = useState("");
  const [customCode, setCustomCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const link = await createLink(url, customCode);
      onCreated(link);
      setUrl("");
      setCustomCode("");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Could not reach the API — is it running?");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="create-form" onSubmit={handleSubmit}>
      <div className="field-row">
        <input
          type="text"
          placeholder="https://example.com/some/long/path"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          required
        />
        <input
          type="text"
          placeholder="custom-code (optional)"
          value={customCode}
          onChange={(e) => setCustomCode(e.target.value)}
          className="custom-code-input"
        />
        <button type="submit" disabled={submitting}>
          {submitting ? "Creating…" : "Shorten"}
        </button>
      </div>
      {error && <p className="error-text">{error}</p>}
    </form>
  );
}
