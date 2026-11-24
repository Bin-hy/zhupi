CREATE TABLE sites (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  key             TEXT    NOT NULL UNIQUE,
  name            TEXT    NOT NULL UNIQUE,
  display_name    TEXT    NOT NULL DEFAULT '',
  status          TEXT    NOT NULL DEFAULT 'active',
  moderation_mode TEXT    NOT NULL DEFAULT 'post',
  theme_color     TEXT    NOT NULL DEFAULT '#087f5b',
  default_locale  TEXT    NOT NULL DEFAULT 'zh-CN',
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL
);

CREATE TABLE site_origins (
  site_id INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  origin  TEXT    NOT NULL,
  PRIMARY KEY (site_id, origin)
);

CREATE TABLE users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  display_name  TEXT    NOT NULL DEFAULT '',
  avatar_url    TEXT    NOT NULL DEFAULT '',
  status        TEXT    NOT NULL DEFAULT 'active',
  is_superadmin INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL
);

CREATE TABLE identities (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider       TEXT    NOT NULL,
  site_id        INTEGER NOT NULL DEFAULT 0,
  issuer         TEXT    NOT NULL,
  subject        TEXT    NOT NULL,
  handle         TEXT    NOT NULL DEFAULT '',
  email          TEXT    NOT NULL DEFAULT '',
  email_verified INTEGER NOT NULL DEFAULT 0,
  avatar_url     TEXT    NOT NULL DEFAULT '',
  created_at     INTEGER NOT NULL,
  last_login_at  INTEGER NOT NULL,
  UNIQUE (provider, site_id, issuer, subject)
);

CREATE INDEX idx_identities_user ON identities(user_id);

CREATE TABLE sessions (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash   TEXT    NOT NULL UNIQUE,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_agent   TEXT    NOT NULL DEFAULT '',
  ip           TEXT    NOT NULL DEFAULT '',
  created_at   INTEGER NOT NULL,
  expires_at   INTEGER NOT NULL,
  last_seen_at INTEGER NOT NULL,
  revoked_at   INTEGER
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE site_admins (
  site_id    INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  granted_by INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (site_id, user_id)
);

CREATE TABLE site_bans (
  site_id    INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  banned_by  INTEGER NOT NULL,
  reason     TEXT    NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  PRIMARY KEY (site_id, user_id)
);

CREATE TABLE oidc_providers (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id             INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  name                TEXT    NOT NULL,
  issuer              TEXT    NOT NULL,
  client_id           TEXT    NOT NULL,
  client_secret_cipher BLOB   NOT NULL,
  scopes              TEXT    NOT NULL DEFAULT '["openid","email","profile"]',
  allowed_email_domains TEXT  NOT NULL DEFAULT '[]',
  enabled             INTEGER NOT NULL DEFAULT 1,
  created_at          INTEGER NOT NULL,
  updated_at          INTEGER NOT NULL,
  UNIQUE (site_id, issuer, client_id)
);

CREATE TABLE auth_flows (
  state_hash    TEXT    PRIMARY KEY,
  mode          TEXT    NOT NULL,
  provider_type TEXT    NOT NULL,
  provider_id   INTEGER NOT NULL DEFAULT 0,
  issuer        TEXT    NOT NULL DEFAULT '',
  site_id       INTEGER NOT NULL,
  user_id       INTEGER NOT NULL DEFAULT 0,
  return_to     TEXT    NOT NULL,
  code_verifier TEXT    NOT NULL DEFAULT '',
  nonce         TEXT    NOT NULL DEFAULT '',
  expires_at    INTEGER NOT NULL
);

CREATE INDEX idx_auth_flows_expires ON auth_flows(expires_at);

CREATE TABLE annotations (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id     INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  page_key    TEXT    NOT NULL,
  page_title  TEXT    NOT NULL DEFAULT '',
  anchor_json TEXT    NOT NULL,
  quote_exact TEXT    NOT NULL,
  section     TEXT    NOT NULL DEFAULT '',
  status      TEXT    NOT NULL DEFAULT 'open',
  moderation  TEXT    NOT NULL DEFAULT 'approved',
  visibility  TEXT    NOT NULL DEFAULT 'visible',
  created_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);

CREATE INDEX idx_annotations_page ON annotations(site_id, page_key, visibility, moderation);
CREATE INDEX idx_annotations_queue ON annotations(site_id, moderation, id);

CREATE TABLE comments (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  site_id       INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  annotation_id INTEGER NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content       TEXT    NOT NULL,
  moderation    TEXT    NOT NULL DEFAULT 'approved',
  visibility    TEXT    NOT NULL DEFAULT 'visible',
  created_at    INTEGER NOT NULL
);

CREATE INDEX idx_comments_annotation ON comments(site_id, annotation_id);
CREATE INDEX idx_comments_queue ON comments(site_id, moderation, id);
