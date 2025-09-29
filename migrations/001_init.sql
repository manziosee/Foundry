CREATE TABLE images (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE,
  blob_key TEXT,
  checksum TEXT,
  state TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE blobs (
  id INTEGER PRIMARY KEY,
  image_id INTEGER REFERENCES images(id),
  path TEXT,
  size INTEGER,
  checksum TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE snapshots (
  id INTEGER PRIMARY KEY,
  image_id INTEGER REFERENCES images(id),
  snapshot_name TEXT,
  active INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);