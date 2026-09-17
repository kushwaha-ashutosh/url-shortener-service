CREATE TABLE IF NOT EXISTS links (
    code       TEXT PRIMARY KEY,
    long_url   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS clicks (
    id         BIGSERIAL PRIMARY KEY,
    code       TEXT NOT NULL REFERENCES links(code),
    clicked_at TIMESTAMPTZ NOT NULL,
    referrer   TEXT,
    user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_clicks_code ON clicks(code);
CREATE INDEX IF NOT EXISTS idx_clicks_clicked_at ON clicks(clicked_at);
