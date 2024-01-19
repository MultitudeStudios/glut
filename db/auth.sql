CREATE SCHEMA auth;

CREATE TABLE auth.users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username text UNIQUE NOT NULL,
  email text NOT NULL,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL,
  verified_at timestamptz,
  last_login_at timestamptz,
  last_login_ip text
);

CREATE TABLE auth.roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text UNIQUE NOT NULL,
  description text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz,
  created_by uuid REFERENCES auth.users (id) ON DELETE SET NULL,
  updated_by uuid REFERENCES auth.users (id) ON DELETE SET NULL
);

CREATE TABLE auth.permissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text UNIQUE NOT NULL,
  description text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz,
  created_by uuid REFERENCES auth.users (id) ON DELETE SET NULL,
  updated_by uuid REFERENCES auth.users (id) ON DELETE SET NULL
);

CREATE TABLE auth.role_permissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  role_id uuid NOT NULL REFERENCES auth.roles (id) ON DELETE CASCADE,
  permission_id uuid NOT NULL REFERENCES auth.permissions (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL,
  created_by uuid REFERENCES auth.users (id) ON DELETE SET NULL,
  UNIQUE (role_id, permission_id)
);

CREATE TABLE auth.user_roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES auth.roles (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL,
  created_by uuid REFERENCES auth.users (id) ON DELETE SET NULL,
  UNIQUE (user_id, role_id)
);

CREATE TABLE auth.sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  token text UNIQUE NOT NULL,
  user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
  user_ip text NOT NULL,
  session_number int NOT NULL,
  created_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  CHECK (expires_at >= created_at),
  UNIQUE (user_id, session_number),
  CHECK (session_number BETWEEN 1 AND 10)
);

CREATE TABLE auth.session_roles (
  session_id text NOT NULL REFERENCES auth.sessions (token) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES auth.roles (id) ON DELETE CASCADE,
  PRIMARY KEY (session_id, role_id)
);

CREATE TABLE auth.tokens (
  id text PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
  kind text NOT NULL,
  created_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  meta hstore,
  UNIQUE (user_id, kind),
  CHECK (expires_at >= created_at)
);

CREATE TABLE auth.bans (
  user_id uuid PRIMARY KEY REFERENCES auth.users (id) ON DELETE CASCADE,
  reason text NOT NULL,
  description text,
  banned_at timestamptz NOT NULL,
  unbanned_at timestamptz NOT NULL,
  banned_by uuid REFERENCES auth.users (id) ON DELETE SET NULL,
  CHECK (unbanned_at >= banned_at)
);
